import { afterEach, describe, expect, it, vi } from "vitest";
import { fetchDailyStats, fetchHints, listFighters, searchFighters, submitGuess } from "./api";

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
