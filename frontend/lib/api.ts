import { z } from "zod";

/** Unset in production: the Worker serves the UI and proxies /api/* same-origin. */
const apiBase = () => process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "") ?? "";

/** Which fighters take part; each pool has its own daily target. */
export const PoolSchema = z.enum(["all", "men"]);
export type Pool = z.infer<typeof PoolSchema>;

/** Minimal fighter payload for autocomplete. No game attributes leak here. */
export const SearchResultSchema = z.object({
  id: z.number(),
  name: z.string(),
  nickname: z.string().nullable(),
  photo_url: z.string().nullable(),
  photo_credit: z.string().nullable(),
  division: z.string().nullable(),
  nationality: z.string(),
});
export type SearchResult = z.infer<typeof SearchResultSchema>;

export const ComparisonSchema = z.enum([
  "correct",
  "incorrect",
  "higher",
  "lower",
]);
export type Comparison = z.infer<typeof ComparisonSchema>;

const resultOf = <T extends z.ZodTypeAny>(value: T) =>
  z.object({ value, comparison: ComparisonSchema });

/** Structured guess outcome. Mirrors the backend contract; validated at runtime. */
export const GuessOutcomeSchema = z.object({
  fighter_id: z.number(),
  fighter_name: z.string(),
  // Optional so games saved before photos existed still restore.
  photo_url: z.string().nullish(),
  photo_credit: z.string().nullish(),
  results: z.object({
    age: resultOf(z.number()),
    division: resultOf(z.string()),
    height: resultOf(z.number()),
    record: resultOf(z.string()),
    nationality: resultOf(z.string()),
    last_event: resultOf(z.string()),
  }),
  correct: z.boolean(),
});
export type GuessOutcome = z.infer<typeof GuessOutcomeSchema>;

export const TodaySchema = z.object({
  date: z.string(),
  status: z.string(),
});
export type Today = z.infer<typeof TodaySchema>;

export const HintSchema = z.object({
  kind: z.string(),
  label: z.string(),
  value: z.string(),
});
export type Hint = z.infer<typeof HintSchema>;

/** Unlocked hints plus the guess count that unlocks the next one (null when none remain). */
export const HintsSchema = z.object({
  hints: z.array(HintSchema),
  next_at: z.number().nullable(),
});
export type Hints = z.infer<typeof HintsSchema>;

async function parseOrThrow<T>(
  res: Response,
  schema: z.ZodType<T>,
  context: string,
): Promise<T> {
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`${context} failed (${res.status}): ${body.slice(0, 200)}`);
  }
  const json = await res.json();
  const parsed = schema.safeParse(json);
  if (!parsed.success) {
    throw new Error(`${context}: unexpected API response shape`);
  }
  return parsed.data;
}

export async function fetchToday(): Promise<Today> {
  const res = await fetch(`${apiBase()}/api/game/today`, {
    cache: "no-store",
  });
  return parseOrThrow(res, TodaySchema, "Loading today's game");
}

export async function searchFighters(q: string, pool: Pool): Promise<SearchResult[]> {
  const res = await fetch(
    `${apiBase()}/api/fighters/search?q=${encodeURIComponent(q)}&pool=${pool}`,
    { cache: "no-store" },
  );
  if (!res.ok) return [];
  const json = await res.json();
  const parsed = z.array(SearchResultSchema).safeParse(json);
  return parsed.success ? parsed.data : [];
}

export async function listFighters(pool: Pool): Promise<SearchResult[]> {
  const res = await fetch(`${apiBase()}/api/fighters?pool=${pool}`, { cache: "no-store" });
  return parseOrThrow(res, z.array(SearchResultSchema), "Loading fighters");
}

export async function submitGuess(fighterId: number, pool: Pool): Promise<GuessOutcome> {
  const res = await fetch(`${apiBase()}/api/game/guess`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    // Sends the session cookie when signed in so the guess is recorded
    // server-side; guests have no cookie and are unaffected.
    credentials: "include",
    body: JSON.stringify({ fighter_id: fighterId, pool }),
  });
  return parseOrThrow(res, GuessOutcomeSchema, "Submitting guess");
}

export async function fetchHints(guesses: number, pool: Pool): Promise<Hints> {
  const res = await fetch(`${apiBase()}/api/game/hints?guesses=${guesses}&pool=${pool}`, {
    cache: "no-store",
  });
  return parseOrThrow(res, HintsSchema, "Loading hints");
}

/** Signed-in account. Password hashes never leave the backend. */
export const AuthUserSchema = z.object({
  id: z.number(),
  email: z.string(),
  display_name: z.string().nullable(),
  created_at: z.string(),
});
export type AuthUser = z.infer<typeof AuthUserSchema>;

const authInit = (method: string, body?: unknown): RequestInit => ({
  method,
  headers: { "Content-Type": "application/json" },
  // Same-origin in production; cross-origin localhost in dev (the backend
  // allows credentials for the configured origins).
  credentials: "include",
  ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
});

async function authError(res: Response, context: string): Promise<Error> {
  const body = await res.text().catch(() => "");
  return new Error(`${context} failed (${res.status}): ${body.slice(0, 200)}`);
}

export async function register(
  email: string,
  password: string,
  displayName?: string,
): Promise<AuthUser> {
  const res = await fetch(
    `${apiBase()}/api/auth/register`,
    authInit("POST", {
      email,
      password,
      ...(displayName ? { display_name: displayName } : {}),
    }),
  );
  if (!res.ok) throw await authError(res, "Creating account");
  return parseOrThrow(res, AuthUserSchema, "Creating account");
}

export async function login(email: string, password: string): Promise<AuthUser> {
  const res = await fetch(`${apiBase()}/api/auth/login`, authInit("POST", { email, password }));
  if (!res.ok) throw await authError(res, "Signing in");
  return parseOrThrow(res, AuthUserSchema, "Signing in");
}

export async function logout(): Promise<void> {
  const res = await fetch(`${apiBase()}/api/auth/logout`, authInit("POST", {}));
  if (!res.ok) throw await authError(res, "Signing out");
}

/** The signed-in account, or null for guests (401 is not an error here). */
export async function fetchMe(): Promise<AuthUser | null> {
  const res = await fetch(`${apiBase()}/api/auth/me`, { credentials: "include" });
  if (res.status === 401) return null;
  return parseOrThrow(res, AuthUserSchema, "Loading account");
}

/** Server-side guesses for one game, restored on any device. */
export async function fetchMyGuesses(pool: Pool, date: string): Promise<GuessOutcome[]> {
  const res = await fetch(
    `${apiBase()}/api/me/guesses?pool=${pool}&date=${encodeURIComponent(date)}`,
    { credentials: "include", cache: "no-store" },
  );
  return parseOrThrow(res, z.array(GuessOutcomeSchema), "Loading your guesses");
}

export const ImportResultSchema = z.object({
  guesses: z.array(GuessOutcomeSchema),
  imported: z.number(),
});
export type ImportResult = z.infer<typeof ImportResultSchema>;

/** Re-evaluates local fighter ids server-side and stores them (import). */
export async function importGuesses(
  pool: Pool,
  date: string,
  fighterIds: number[],
): Promise<ImportResult> {
  const res = await fetch(
    `${apiBase()}/api/me/import`,
    authInit("POST", { pool, date, fighter_ids: fighterIds }),
  );
  if (!res.ok) throw await authError(res, "Importing guesses");
  return parseOrThrow(res, ImportResultSchema, "Importing guesses");
}
