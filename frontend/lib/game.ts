import type { Comparison, GuessOutcome } from "./api";

/**
 * Accessible metadata for each comparison state. UI must never rely on
 * color alone: every state has a symbol AND a text label.
 */
export function comparisonMeta(comparison: Comparison): {
  symbol: string;
  label: string;
  classes: string;
} {
  switch (comparison) {
    case "correct":
      return {
        symbol: "✓",
        label: "correct",
        classes: "bg-emerald-500/20 text-emerald-300 border-emerald-500/40",
      };
    case "higher":
      return {
        symbol: "↑",
        label: "target is higher",
        classes: "bg-amber-500/20 text-amber-300 border-amber-500/40",
      };
    case "lower":
      return {
        symbol: "↓",
        label: "target is lower",
        classes: "bg-sky-500/20 text-sky-300 border-sky-500/40",
      };
    case "incorrect":
    default:
      return {
        symbol: "✗",
        label: "incorrect",
        classes: "bg-zinc-500/20 text-zinc-300 border-zinc-500/40",
      };
  }
}

/** Emoji grid for shareable results (future: share button). */
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

export function storageKey(gameDate: string): string {
  return `mmadle-guesses-${gameDate}`;
}
