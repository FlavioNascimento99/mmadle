import { z } from "zod";
import { GuessOutcomeSchema, type Comparison, type GuessOutcome, type Pool } from "./api";
import { DICTS, type Lang } from "./i18n";

/**
 * Accessible metadata for each comparison state. UI must never rely on
 * color alone: every state has a symbol AND a text label.
 */
export function comparisonMeta(comparison: Comparison, lang: Lang = "en"): {
  symbol: string;
  label: string;
  classes: string;
} {
  const d = DICTS[lang];
  switch (comparison) {
    case "correct":
      return {
        symbol: "✓",
        label: d["cmp.correct"],
        classes: "bg-blood text-bone",
      };
    case "higher":
      return {
        symbol: "↑",
        label: d["cmp.higher"],
        classes: "bg-bone text-ink",
      };
    case "lower":
      return {
        symbol: "↓",
        label: d["cmp.lower"],
        classes: "bg-bone text-ink",
      };
    case "incorrect":
    default:
      return {
        symbol: "✗",
        label: d["cmp.incorrect"],
        classes: "bg-ink/[0.05] text-ash",
      };
  }
}

/** Emoji grid row for one guess in the shareable result. */
export function outcomeToEmoji(outcome: GuessOutcome): string {
  const cell = (c: Comparison) =>
    c === "correct" ? "🟩" : c === "incorrect" ? "🟥" : c === "higher" ? "🔼" : "🔽";
  const r = outcome.results;
  return [
    cell(r.age.comparison),
    cell(r.division.comparison),
    cell(r.height.comparison),
    cell(r.record.comparison),
    cell(r.nationality.comparison),
    cell(r.last_event.comparison),
  ].join("");
}

export function storageKey(gameDate: string, pool: Pool): string {
  return `mmadle-guesses-${pool}-${gameDate}`;
}

export function shareText(gameDate: string, pool: Pool, guesses: GuessOutcome[]): string {
  const mode = pool === "men" ? " (men only)" : "";
  const tries = `${guesses.length} ${guesses.length === 1 ? "try" : "tries"}`;
  return [`MMAdle ${gameDate}${mode} — ${tries}`, ...guesses.map(outcomeToEmoji)].join("\n");
}

/** Restores a saved game from localStorage; anything invalid starts fresh. */
export function parseSavedGuesses(raw: string | null): GuessOutcome[] {
  if (!raw) return [];
  try {
    const parsed = z.array(GuessOutcomeSchema).safeParse(JSON.parse(raw));
    return parsed.success ? parsed.data : [];
  } catch {
    return [];
  }
}
