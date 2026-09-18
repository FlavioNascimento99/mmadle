import { z } from "zod";

const apiBase = () =>
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "") ?? "http://localhost:8080";

/** Minimal fighter payload for autocomplete. No game attributes leak here. */
export const SearchResultSchema = z.object({
  id: z.number(),
  name: z.string(),
  nickname: z.string().nullable(),
  photo_url: z.string().nullable(),
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

export async function searchFighters(q: string): Promise<SearchResult[]> {
  const res = await fetch(
    `${apiBase()}/api/fighters/search?q=${encodeURIComponent(q)}`,
    { cache: "no-store" },
  );
  if (!res.ok) return [];
  const json = await res.json();
  const parsed = z.array(SearchResultSchema).safeParse(json);
  return parsed.success ? parsed.data : [];
}

export async function submitGuess(fighterId: number): Promise<GuessOutcome> {
  const res = await fetch(`${apiBase()}/api/game/guess`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ fighter_id: fighterId }),
  });
  return parseOrThrow(res, GuessOutcomeSchema, "Submitting guess");
}
