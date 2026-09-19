import { afterEach, describe, expect, it, vi } from "vitest";
import {
  fetchCFWorkers,
  fetchMe,
  fetchMyGuesses,
  fetchMyStats,
  fetchOverview,
  importGuesses,
  login,
  logout,
  register,
} from "./api";

const stubFetch = (status: number, body: unknown) =>
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response(JSON.stringify(body), { status })),
  );

afterEach(() => {
  vi.unstubAllGlobals();
});

const account = {
  id: 1,
  username: "octagon_fan",
  role: "player",
  created_at: "2026-09-18T00:00:00Z",
};

describe("auth", () => {
  it("registers with username and password", async () => {
    stubFetch(201, account);
    await expect(register("octagon_fan", "a-correct-horse-battery9")).resolves.toEqual(account);
    expect(fetch).toHaveBeenCalledWith("/api/auth/register", expect.objectContaining({
      method: "POST",
      credentials: "include",
      body: JSON.stringify({
        username: "octagon_fan",
        password: "a-correct-horse-battery9",
      }),
    }));
  });

  it("logs in and surfaces server errors with the status", async () => {
    stubFetch(200, account);
    await expect(login("octagon_fan", "a-correct-horse-battery9")).resolves.toEqual(account);

    stubFetch(401, { error: "invalid_credentials" });
    await expect(login("octagon_fan", "wrong")).rejects.toThrow(/401/);
  });

  it("logs out with an empty JSON body", async () => {
    stubFetch(200, { status: "ok" });
    await expect(logout()).resolves.toBeUndefined();
    expect(fetch).toHaveBeenCalledWith("/api/auth/logout", expect.objectContaining({
      method: "POST",
      credentials: "include",
    }));
  });

  it("returns null for guests instead of throwing on 401", async () => {
    stubFetch(401, { error: "unauthenticated" });
    await expect(fetchMe()).resolves.toBeNull();
  });

  it("returns the account when signed in", async () => {
    stubFetch(200, account);
    await expect(fetchMe()).resolves.toEqual(account);
  });
});

describe("server guesses", () => {  it("fetches one game's history", async () => {
    stubFetch(200, []);
    await expect(fetchMyGuesses("men", "2026-09-18")).resolves.toEqual([]);
    expect(fetch).toHaveBeenCalledWith("/api/me/guesses?pool=men&date=2026-09-18", expect.objectContaining({
      credentials: "include",
    }));
  });

  it("imports local fighter ids for a game", async () => {
    stubFetch(200, { guesses: [], imported: 0 });
    await expect(importGuesses("all", "2026-09-18", [1, 2])).resolves.toEqual({
      guesses: [],
      imported: 0,
    });
    expect(fetch).toHaveBeenCalledWith("/api/me/import", expect.objectContaining({
      method: "POST",
      credentials: "include",
      body: JSON.stringify({ pool: "all", date: "2026-09-18", fighter_ids: [1, 2] }),
    }));
  });
});

describe("admin metrics", () => {
  const overview = {
    days: 30,
    since: "2026-08-20",
    signups_total: 5,
    signups_by_day: [],
    players_by_day: [{ date: "2026-09-18", count: 2 }],
    guesses_by_day: [{ date: "2026-09-18", count: 4 }],
    games_total: 2,
    games_won: 1,
    win_rate: 0.5,
    avg_guesses_to_win: 3,
    by_pool: [{ pool: "all", games: 2, won: 1, guesses: 4 }],
    top_fighters: [{ fighter_id: 1, name: "Conor McGregor", guesses: 2 }],
  };

  it("fetches the overview with credentials", async () => {
    stubFetch(200, overview);
    await expect(fetchOverview(30)).resolves.toEqual(overview);
    expect(fetch).toHaveBeenCalledWith("/api/admin/metrics/overview?days=30", expect.objectContaining({
      credentials: "include",
    }));
  });

  it("rejects non-admins so the page can show a locked door", async () => {
    stubFetch(404, { error: "not_found" });
    await expect(fetchOverview(30)).rejects.toThrow(/404/);
  });

  it("returns null when Cloudflare secrets are missing", async () => {
    stubFetch(501, { error: "cloudflare_not_configured" });
    await expect(fetchCFWorkers(7)).resolves.toBeNull();
  });

  it("parses the Cloudflare workers report", async () => {
    const report = {
      days: 7,
      since: "2026-09-12",
      script: "mmadle",
      requests: 100,
      errors: 2,
      by_day: [
        { date: "2026-09-18", requests: 100, errors: 2, subrequests: 5, cpu_time_p50_us: 1.5, cpu_time_p99_us: 9.25 },
      ],
    };
    stubFetch(200, report);
    await expect(fetchCFWorkers(7)).resolves.toEqual(report);
  });
});

describe("my stats", () => {
  it("fetches personal stats with credentials", async () => {
    const stats = {
      pools: {
        all: {
          games_played: 3, games_won: 2, win_rate: 2 / 3,
          current_streak: 1, max_streak: 1, avg_tries: 4, avg_tries_to_win: 3,
          distribution: { "2": 1, "4": 1 }, days_played: 3,
        },
        men: {
          games_played: 0, games_won: 0, win_rate: 0,
          current_streak: 0, max_streak: 0, avg_tries: 0, avg_tries_to_win: 0,
          distribution: {}, days_played: 0,
        },
      },
      games_total: 3,
      games_won: 2,
      win_rate: 2 / 3,
      days_played: 3,
      avg_tries_per_day: 4,
      recent_games: [{ date: "2026-09-18", pool: "all", guesses: 2, won: true }],
    };
    stubFetch(200, stats);
    await expect(fetchMyStats()).resolves.toEqual(stats);
    expect(fetch).toHaveBeenCalledWith("/api/me/stats", expect.objectContaining({
      credentials: "include",
    }));
  });

  it("surfaces 401 for guests", async () => {
    stubFetch(401, { error: "unauthenticated" });
    await expect(fetchMyStats()).rejects.toThrow(/401/);
  });
});
