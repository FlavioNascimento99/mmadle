import type { Comparison, GuessOutcome } from "@/lib/api";
import { comparisonMeta } from "@/lib/game";

function Cell({
  label,
  display,
  comparison,
  hint,
}: {
  label: string;
  display: string;
  comparison: Comparison;
  hint?: string;
}) {
  const meta = comparisonMeta(comparison);
  return (
    <div
      role="cell"
      aria-label={`${label}: ${display}, ${meta.label}`}
      title={`${label}: ${meta.label}`}
      className={`flex min-w-0 flex-col items-center gap-0.5 rounded-lg border px-2 py-2 text-center ${meta.classes}`}
    >
      <span className="text-[10px] font-semibold uppercase tracking-wide opacity-70">
        {label}
      </span>
      <span className="truncate text-sm font-bold">{display}</span>
      <span className="text-xs font-medium" aria-hidden="true">
        {meta.symbol} {comparison === "correct" ? "✓" === meta.symbol ? "match" : "" : hint ?? meta.label}
      </span>
    </div>
  );
}

export function GuessTable({ guesses }: { guesses: GuessOutcome[] }) {
  if (guesses.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-zinc-700 p-8 text-center text-sm text-zinc-400">
        No guesses yet. Search a fighter above to start today&apos;s game.
      </div>
    );
  }
  return (
    <div className="space-y-3" role="table" aria-label="Guess history">
      {[...guesses].reverse().map((g) => (
        <div
          key={`${g.fighter_id}-${guesses.indexOf(g)}`}
          role="row"
          className="rounded-xl border border-zinc-800 bg-zinc-900/60 p-3"
        >
          <div className="mb-2 flex items-center justify-between gap-2">
            <p className="truncate font-bold text-zinc-50">
              <span className="mr-2 inline-flex h-6 w-6 items-center justify-center rounded-full bg-zinc-700 text-xs">
                {guesses.indexOf(g) + 1}
              </span>
              {g.fighter_name}
            </p>
            {g.correct && (
              <span className="rounded-full bg-emerald-500/20 px-2 py-0.5 text-xs font-bold text-emerald-300">
                ✓ Correct!
              </span>
            )}
          </div>
          <div className="grid grid-cols-2 gap-1.5 sm:grid-cols-3 lg:grid-cols-6" role="rowgroup">
            <Cell
              label="Age"
              display={`${g.results.age.value}`}
              comparison={g.results.age.comparison}
              hint={g.results.age.comparison === "higher" ? "↑ older" : g.results.age.comparison === "lower" ? "↓ younger" : "match"}
            />
            <Cell label="Division" display={g.results.division.value} comparison={g.results.division.comparison} />
            <Cell
              label="Height"
              display={`${g.results.height.value} cm`}
              comparison={g.results.height.comparison}
              hint={g.results.height.comparison === "higher" ? "↑ taller" : g.results.height.comparison === "lower" ? "↓ shorter" : "match"}
            />
            <Cell label="Record" display={g.results.record.value} comparison={g.results.record.comparison} />
            <Cell label="Nation" display={g.results.nationality.value} comparison={g.results.nationality.comparison} />
            <Cell label="Last event" display={g.results.last_event.value} comparison={g.results.last_event.comparison} />
          </div>
        </div>
      ))}
    </div>
  );
}
