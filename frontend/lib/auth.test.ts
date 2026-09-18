import { afterEach, describe, expect, it, vi } from "vitest";
import {
  fetchMe,
  fetchMyGuesses,
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
  email: "fan@mmadle.gg",
  display_name: "Fan",
  created_at: "2026-09-18T00:00:00Z",
};

describe("auth", () => {
  it("registers with email, password and display name", async () => {
    stubFetch(201, account);
    await expect(register("fan@mmadle.gg", "a-correct-horse-battery9", "Fan")).resolves.toEqual(account);
    expect(fetch).toHaveBeenCalledWith("/api/auth/register", expect.objectContaining({
      method: "POST",
      credentials: "include",
      body: JSON.stringify({
        email: "fan@mmadle.gg",
        password: "a-correct-horse-battery9",
        display_name: "Fan",
      }),
    }));
  });

  it("logs in and surfaces server errors with the status", async () => {
    stubFetch(200, account);
    await expect(login("fan@mmadle.gg", "a-correct-horse-battery9")).resolves.toEqual(account);

    stubFetch(401, { error: "invalid_credentials" });
    await expect(login("fan@mmadle.gg", "wrong")).rejects.toThrow(/401/);
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

describe("server guesses", () => {
  it("fetches one game's history", async () => {
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
