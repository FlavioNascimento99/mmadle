import type { Comparison, GuessOutcome } from "@/lib/api";
import { comparisonMeta } from "@/lib/game";
import { useLang } from "@/lib/i18n";
import { FighterPhoto } from "./FighterPhoto";

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
  const { t, lang } = useLang();
  const meta = comparisonMeta(comparison, lang);
  const landed = comparison === "correct";
  const missed = comparison === "incorrect";
  return (
    <div
      role="cell"
      aria-label={`${label}: ${display}, ${meta.label}`}
      title={`${label}: ${display} (${meta.label})`}
      className={`flex min-w-0 flex-col gap-1 border-2 p-2 ${
        landed
          ? "animate-land border-ink bg-blood text-bone"
          : missed
            ? "border-ink/20 bg-ink/[0.04] text-ash"
            : "border-ink bg-bone text-ink"
      }`}
    >
      <span className={`font-mono text-[10px] font-bold uppercase tracking-[0.18em] ${landed ? "text-bone/80" : "opacity-70"}`}>
        {label}
      </span>
      <span className="truncate font-mono text-sm font-bold uppercase tracking-wide">{display}</span>
      <span className="flex items-center gap-1.5" aria-hidden="true">
        <span
          className={`inline-flex h-5 w-5 items-center justify-center border text-xs font-black leading-none ${
            landed
              ? "border-bone/60 bg-bone/15 text-bone"
              : missed
                ? "border-ink/20 text-ash"
                : "border-ink bg-ink text-bone"
          }`}
        >
          {meta.symbol}
        </span>
        <span className={`text-[11px] font-bold uppercase tracking-wide ${landed ? "text-bone" : missed ? "text-ash" : "text-bruise"}`}>
          {hint ?? (landed ? t("table.match") : t("table.noMatch"))}
        </span>
      </span>
    </div>
  );
}

const orderedHint = (comparison: Comparison, up: string, down: string) =>
  comparison === "higher" ? up : comparison === "lower" ? down : undefined;

export function GuessTable({ guesses }: { guesses: GuessOutcome[] }) {
  const { t } = useLang();
  if (guesses.length === 0) {
    return (
      <div className="border-2 border-dashed border-bone/25 p-8 text-center font-mono text-xs tracking-wide text-ash">
        {t("table.empty")}
      </div>
    );
  }
  return (
    <ol className="space-y-4" role="table" aria-label={t("table.history")}>
      {guesses
        .map((g, i) => ({ g, n: i + 1 }))
        .reverse()
        .map(({ g, n }, order) => (
          <li
            key={`${g.fighter_id}-${n}`}
            role="row"
            style={{ animationDelay: `${Math.min(order, 5) * 45}ms` }}
            className={`animate-guess-in border-3 border-ink bg-bone p-3 text-ink sm:p-4 ${
              g.correct ? "shadow-blood" : "shadow-hard"
            }`}
          >
            <div className="flex gap-3 sm:gap-4">
              <div className="hidden shrink-0 flex-col items-center gap-2 sm:flex">
                <FighterPhoto name={g.fighter_name} url={g.photo_url} credit={g.photo_credit} size="md" />
                <span className="font-mono text-[11px] font-bold tracking-[0.14em] text-ash">
                  {t("table.guessN", { n })}
                </span>
              </div>
              <div className="min-w-0 flex-1">
                <div className="mb-3 flex items-center gap-2">
                  <span className="microlabel shrink-0 border-2 border-ink bg-ink px-1.5 py-0.5 text-bone">
                    #{n}
                  </span>
                  <FighterPhoto name={g.fighter_name} url={g.photo_url} credit={g.photo_credit} size="sm" className="sm:hidden" />
                  <p className="min-w-0 flex-1 truncate font-display text-2xl uppercase leading-none tracking-wide sm:text-3xl">
                    {g.fighter_name}
                  </p>
                  {g.correct && (
                    <span className="shrink-0 border-2 border-ink bg-blood px-2 py-0.5 font-mono text-[11px] font-bold uppercase tracking-[0.18em] text-bone">
                      {t("table.correct")}
                    </span>
                  )}
                </div>
                <div className="grid grid-cols-2 gap-1.5 sm:grid-cols-3 lg:grid-cols-5" role="rowgroup">
                  <Cell
                    label={t("table.attrAge")}
                    display={`${g.results.age.value}`}
                    comparison={g.results.age.comparison}
                    hint={orderedHint(g.results.age.comparison, t("table.older"), t("table.younger"))}
                  />
                  <Cell label={t("table.attrDivision")} display={g.results.division.value} comparison={g.results.division.comparison} />
                  <Cell
                    label={t("table.attrHeight")}
                    display={`${g.results.height.value} cm`}
                    comparison={g.results.height.comparison}
                    hint={orderedHint(g.results.height.comparison, t("table.taller"), t("table.shorter"))}
                  />
                  <Cell label={t("table.attrNation")} display={g.results.nationality.value} comparison={g.results.nationality.comparison} />
                  <Cell label={t("table.attrEvent")} display={g.results.last_event.value} comparison={g.results.last_event.comparison} />
                </div>
              </div>
            </div>
          </li>
        ))}
    </ol>
  );
}
