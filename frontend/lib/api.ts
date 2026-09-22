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

/** How many players solved today's daily game in one pool, logged in or not. */
export const DailyStatsSchema = z.object({
  date: z.string(),
  pool: z.string(),
  solvers: z.number(),
});
export type DailyStats = z.infer<typeof DailyStatsSchema>;

export async function fetchDailyStats(pool: Pool): Promise<DailyStats> {
  const res = await fetch(`${apiBase()}/api/game/stats?pool=${pool}`, {
    cache: "no-store",
  });
  return parseOrThrow(res, DailyStatsSchema, "Loading daily stats");
}

/** Signed-in account. Password hashes never leave the backend. */
export const AuthUserSchema = z.object({
  id: z.number(),
  username: z.string(),
  role: z.enum(["player", "admin"]),
  created_at: z.string(),
  // Whether the account appears on the public leaderboard.
  leaderboard_opt_in: z.boolean(),
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

export async function register(username: string, password: string): Promise<AuthUser> {
  const res = await fetch(
    `${apiBase()}/api/auth/register`,
    authInit("POST", { username, password }),
  );
  if (!res.ok) throw await authError(res, "Creating account");
  return parseOrThrow(res, AuthUserSchema, "Creating account");
}

export async function login(username: string, password: string): Promise<AuthUser> {
  const res = await fetch(`${apiBase()}/api/auth/login`, authInit("POST", { username, password }));
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

const DayCountSchema = z.object({ date: z.string(), count: z.number() });

/** Product metrics from our own game records (admin only). */
export const MetricsOverviewSchema = z.object({
  days: z.number(),
  since: z.string(),
  signups_total: z.number(),
  signups_by_day: z.array(DayCountSchema),
  players_by_day: z.array(DayCountSchema),
  guesses_by_day: z.array(DayCountSchema),
  games_total: z.number(),
  games_won: z.number(),
  win_rate: z.number(),
  avg_guesses_to_win: z.number(),
  by_pool: z.array(
    z.object({ pool: z.string(), games: z.number(), won: z.number(), guesses: z.number() }),
  ),
  top_fighters: z.array(
    z.object({ fighter_id: z.number(), name: z.string(), guesses: z.number() }),
  ),
});
export type MetricsOverview = z.infer<typeof MetricsOverviewSchema>;

export async function fetchOverview(days: number): Promise<MetricsOverview> {
  const res = await fetch(`${apiBase()}/api/admin/metrics/overview?days=${days}`, {
    credentials: "include",
    cache: "no-store",
  });
  return parseOrThrow(res, MetricsOverviewSchema, "Loading metrics");
}

/** Cloudflare Worker infra (requests/errors/CPU), proxied so the API token stays server-side. */
export const CFWorkersSchema = z.object({
  days: z.number(),
  since: z.string(),
  script: z.string(),
  requests: z.number(),
  errors: z.number(),
  by_day: z.array(
    z.object({
      date: z.string(),
      requests: z.number(),
      errors: z.number(),
      subrequests: z.number(),
      cpu_time_p50_us: z.number(),
      cpu_time_p99_us: z.number(),
    }),
  ),
});
export type CFWorkers = z.infer<typeof CFWorkersSchema>;

/** Worker infra, or null when the Cloudflare secrets are not configured (501). */
export async function fetchCFWorkers(days: number): Promise<CFWorkers | null> {
  const res = await fetch(`${apiBase()}/api/admin/cloudflare/workers?days=${days}`, {
    credentials: "include",
    cache: "no-store",
  });
  if (res.status === 501) return null;
  return parseOrThrow(res, CFWorkersSchema, "Loading Cloudflare metrics");
}

const PoolStatsSchema = z.object({
  games_played: z.number(),
  games_won: z.number(),
  win_rate: z.number(),
  current_streak: z.number(),
  max_streak: z.number(),
  avg_tries: z.number(),
  avg_tries_to_win: z.number(),
  distribution: z.record(z.string(), z.number()),
  days_played: z.number(),
});
export type PoolStats = z.infer<typeof PoolStatsSchema>;

/** Personal statistics for the signed-in player. */
export const UserStatsSchema = z.object({
  pools: z.record(z.string(), PoolStatsSchema),
  games_total: z.number(),
  games_won: z.number(),
  win_rate: z.number(),
  days_played: z.number(),
  avg_tries_per_day: z.number(),
  recent_games: z.array(
    z.object({
      date: z.string(),
      pool: PoolSchema,
      guesses: z.number(),
      won: z.boolean(),
    }),
  ),
});
export type UserStats = z.infer<typeof UserStatsSchema>;

export async function fetchMyStats(): Promise<UserStats> {
  const res = await fetch(`${apiBase()}/api/me/stats`, {
    credentials: "include",
    cache: "no-store",
  });
  return parseOrThrow(res, UserStatsSchema, "Loading your stats");
}

/** An opened infinity-round: opaque id plus full lives. Never the target. */
export const RoundSchema = z.object({
  round_id: z.string(),
  lives: z.number(),
  pool: PoolSchema,
});
export type Round = z.infer<typeof RoundSchema>;

/** Guess outcome plus round state. Answer only arrives with death; streak
 * fields only for signed-in players (guests track bests in the browser). */
export const InfiniteGuessSchema = GuessOutcomeSchema.extend({
  lives_left: z.number(),
  solved: z.boolean(),
  round_over: z.boolean(),
  answer: z
    .object({
      name: z.string(),
      photo_url: z.string().nullable(),
      photo_credit: z.string().nullable(),
    })
    .nullish(),
  streak: z.number().optional(),
  best: z.number().optional(),
  new_best: z.boolean().optional(),
});
export type InfiniteGuess = z.infer<typeof InfiniteGuessSchema>;

export async function createRound(pool: Pool): Promise<Round> {
  const res = await fetch(`${apiBase()}/api/infinite/rounds`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ pool }),
  });
  return parseOrThrow(res, RoundSchema, "Starting round");
}

export async function guessInfinite(roundId: string, fighterId: number): Promise<InfiniteGuess> {
  const res = await fetch(`${apiBase()}/api/infinite/guess`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ round_id: roundId, fighter_id: fighterId }),
  });
  return parseOrThrow(res, InfiniteGuessSchema, "Submitting guess");
}

export const InfiniteRecordSchema = z.object({
  best_streak: z.number(),
  current_streak: z.number(),
});
export type InfiniteRecord = z.infer<typeof InfiniteRecordSchema>;

/** The account's survival run (auth required). */
export async function fetchInfiniteRecord(pool: Pool): Promise<InfiniteRecord> {
  const res = await fetch(`${apiBase()}/api/me/infinite/record?pool=${pool}`, {
    credentials: "include",
    cache: "no-store",
  });
  return parseOrThrow(res, InfiniteRecordSchema, "Loading record");
}

/** One account row for the admin users view. */
export const AdminUserSchema = z.object({
  id: z.number(),
  username: z.string(),
  role: z.string(),
  is_active: z.boolean(),
  created_at: z.string(),
  last_login_at: z.string().nullable(),
  games: z.number(),
  won: z.number(),
});
export type AdminUser = z.infer<typeof AdminUserSchema>;

export const UsersPageSchema = z.object({
  total: z.number(),
  active: z.number(),
  users: z.array(AdminUserSchema),
});
export type UsersPage = z.infer<typeof UsersPageSchema>;

export async function fetchAdminUsers(q: string, limit: number, offset: number): Promise<UsersPage> {
  const res = await fetch(
    `${apiBase()}/api/admin/users?q=${encodeURIComponent(q)}&limit=${limit}&offset=${offset}`,
    { credentials: "include", cache: "no-store" },
  );
  return parseOrThrow(res, UsersPageSchema, "Loading accounts");
}

export const SetActiveResultSchema = z.object({ user_id: z.number(), active: z.boolean() });
export type SetActiveResult = z.infer<typeof SetActiveResultSchema>;

export async function setUserActive(userId: number, active: boolean): Promise<SetActiveResult> {
  const res = await fetch(`${apiBase()}/api/admin/users/active`, authInit("POST", { user_id: userId, active }));
  if (!res.ok) throw await authError(res, "Updating account");
  return parseOrThrow(res, SetActiveResultSchema, "Updating account");
}

/** One ranked player on the public leaderboard. */
export const LeaderboardEntrySchema = z.object({
  rank: z.number(),
  username: z.string(),
  score: z.number(),
  games: z.number(),
  wins: z.number(),
  win_rate: z.number(),
  current_streak: z.number(),
  max_streak: z.number(),
  avg_tries: z.number(),
});
export type LeaderboardEntry = z.infer<typeof LeaderboardEntrySchema>;

/** The signed-in player's own state; entry is null until they opt in and rank. */
export const LeaderboardMySchema = z.object({
  opt_in: z.boolean(),
  entry: LeaderboardEntrySchema.nullable(),
});
export type LeaderboardMy = z.infer<typeof LeaderboardMySchema>;

export const LeaderboardSchema = z.object({
  pool: PoolSchema,
  total: z.number(),
  rankings: z.array(LeaderboardEntrySchema),
  // Guests and opted-out-after-login visitors still receive my: the UI uses
  // opt_in to render the toggle and entry to highlight the own row.
  my: LeaderboardMySchema.nullable(),
});
export type Leaderboard = z.infer<typeof LeaderboardSchema>;

/** Public ranking for one pool. my comes along for signed-in visitors. */
export async function fetchLeaderboard(
  pool: Pool,
  limit = 50,
  offset = 0,
): Promise<Leaderboard> {
  const res = await fetch(
    `${apiBase()}/api/leaderboard?pool=${pool}&limit=${limit}&offset=${offset}`,
    { credentials: "include", cache: "no-store" },
  );
  return parseOrThrow(res, LeaderboardSchema, "Loading leaderboard");
}

export const SetLeaderboardOptInSchema = z.object({
  user_id: z.number(),
  leaderboard_opt_in: z.boolean(),
});
export type SetLeaderboardOptInResult = z.infer<typeof SetLeaderboardOptInSchema>;

/** Toggles the public appearance of the account (auth required). */
export async function setLeaderboardOptIn(optIn: boolean): Promise<SetLeaderboardOptInResult> {
  const res = await fetch(
    `${apiBase()}/api/me/leaderboard`,
    authInit("POST", { opt_in: optIn }),
  );
  if (!res.ok) throw await authError(res, "Updating leaderboard visibility");
  return parseOrThrow(res, SetLeaderboardOptInSchema, "Updating leaderboard visibility");
}
