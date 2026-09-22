import { afterEach, describe, expect, it, vi } from "vitest";
import {
  fetchAdminUsers,
  fetchDailyStats,
  fetchHints,
  fetchLeaderboard,
  listFighters,
  searchFighters,
  setLeaderboardOptIn,
  setUserActive,
  submitGuess,
} from "./api";

const stubFetch = (status: number, body: unknown) =>
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response(JSON.stringify(body), { status })),
  );

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("listFighters", () => {
  it("requests the pool's roster and returns validated fighters", async () => {
    const roster = [
      {
        id: 3,
        name: "Alex Pereira",
        nickname: "Poatan",
        photo_url: null,
        photo_credit: null,
        division: "Light Heavyweight",
        nationality: "Brazil",
      },
    ];
    stubFetch(200, roster);

    await expect(listFighters("men")).resolves.toEqual(roster);
    expect(fetch).toHaveBeenCalledWith("/api/fighters?pool=men", { cache: "no-store" });
  });

  it("throws on a non-OK response so the UI can show an error", async () => {
    stubFetch(500, { error: "list_failed" });
    await expect(listFighters("all")).rejects.toThrow(/500/);
  });

  it("throws on an unexpected payload shape", async () => {
    stubFetch(200, [{ id: "x" }]);
    await expect(listFighters("all")).rejects.toThrow(/unexpected API response shape/);
  });
});

describe("searchFighters", () => {
  it("sends the query and pool", async () => {
    stubFetch(200, []);
    await searchFighters("o'mal ley", "men");
    expect(fetch).toHaveBeenCalledWith("/api/fighters/search?q=o'mal%20ley&pool=men", { cache: "no-store" });
  });
});

describe("submitGuess", () => {
  it("posts the fighter and pool", async () => {
    stubFetch(500, {});
    await expect(submitGuess(7, "men")).rejects.toThrow();
    expect(fetch).toHaveBeenCalledWith("/api/game/guess", expect.objectContaining({
      method: "POST",
      body: JSON.stringify({ fighter_id: 7, pool: "men" }),
    }));
  });
});

describe("fetchHints", () => {
  it("requests hints for the guess count and pool", async () => {
    const hints = {
      hints: [{ kind: "nationality", label: "Nationality", value: "Georgia" }],
      next_at: 5,
    };
    stubFetch(200, hints);
    await expect(fetchHints(3, "all")).resolves.toEqual(hints);
    expect(fetch).toHaveBeenCalledWith("/api/game/hints?guesses=3&pool=all", { cache: "no-store" });
  });

  it("accepts a fully unlocked result", async () => {
    stubFetch(200, { hints: [], next_at: null });
    await expect(fetchHints(0, "all")).resolves.toEqual({ hints: [], next_at: null });
  });
});

describe("fetchDailyStats", () => {
  it("requests the solvers count for the pool", async () => {
    const stats = { date: "2026-09-18", pool: "men", solvers: 3 };
    stubFetch(200, stats);
    await expect(fetchDailyStats("men")).resolves.toEqual(stats);
    expect(fetch).toHaveBeenCalledWith("/api/game/stats?pool=men", { cache: "no-store" });
  });

  it("throws on an unexpected payload shape", async () => {
    stubFetch(200, { solvers: "many" });
    await expect(fetchDailyStats("all")).rejects.toThrow(/unexpected API response shape/);
  });
});

describe("admin users", () => {
  it("fetches the paginated account listing", async () => {
    const page = {
      total: 2,
      active: 1,
      users: [
        { id: 2, username: "octagon_fan", role: "player", is_active: false, created_at: "2026-09-17T10:00:00Z", last_login_at: null, games: 3, won: 1 },
      ],
    };
    stubFetch(200, page);
    await expect(fetchAdminUsers("oct", 20, 0)).resolves.toEqual(page);
    expect(fetch).toHaveBeenCalledWith("/api/admin/users?q=oct&limit=20&offset=0", expect.objectContaining({
      credentials: "include",
    }));
  });

  it("posts the activation switch", async () => {
    stubFetch(200, { user_id: 2, active: false });
    await expect(setUserActive(2, false)).resolves.toEqual({ user_id: 2, active: false });
    expect(fetch).toHaveBeenCalledWith("/api/admin/users/active", expect.objectContaining({
      method: "POST",
      body: JSON.stringify({ user_id: 2, active: false }),
    }));
  });

  it("surfaces toggle failures", async () => {
    stubFetch(404, { error: "user_not_found" });
    await expect(setUserActive(999, true)).rejects.toThrow(/404/);
  });
});

describe("leaderboard", () => {
  const entry = {
    rank: 1,
    username: "octagon_fan",
    score: 21,
    games: 2,
    wins: 2,
    win_rate: 1,
    current_streak: 2,
    max_streak: 2,
    avg_tries: 3.5,
  };

  it("fetches the public ranking for a pool with credentials", async () => {
    const page = { pool: "all", total: 1, rankings: [entry], my: null };
    stubFetch(200, page);
    await expect(fetchLeaderboard("all", 50, 0)).resolves.toEqual(page);
    expect(fetch).toHaveBeenCalledWith("/api/leaderboard?pool=all&limit=50&offset=0", expect.objectContaining({
      credentials: "include",
    }));
  });

  it("keeps the signed-in player's own entry and opt-in state", async () => {
    const page = { pool: "men", total: 1, rankings: [entry], my: { opt_in: true, entry } };
    stubFetch(200, page);
    const got = await fetchLeaderboard("men");
    expect(got.my?.opt_in).toBe(true);
    expect(got.my?.entry?.rank).toBe(1);
  });

  it("throws on malformed ranking rows", async () => {
    stubFetch(200, { pool: "all", total: 1, rankings: [{ rank: "one" }], my: null });
    await expect(fetchLeaderboard("all")).rejects.toThrow(/unexpected API response shape/);
  });

  it("posts the opt-in toggle", async () => {
    stubFetch(200, { user_id: 2, leaderboard_opt_in: true });
    await expect(setLeaderboardOptIn(true)).resolves.toEqual({ user_id: 2, leaderboard_opt_in: true });
    expect(fetch).toHaveBeenCalledWith("/api/me/leaderboard", expect.objectContaining({
      method: "POST",
      credentials: "include",
      body: JSON.stringify({ opt_in: true }),
    }));
  });

  it("surfaces opt-in failures", async () => {
    stubFetch(500, { error: "leaderboard_failed" });
    await expect(setLeaderboardOptIn(false)).rejects.toThrow(/500/);
  });
});
