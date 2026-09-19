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
  return (
    <div
      role="cell"
      aria-label={`${label}: ${display}, ${meta.label}`}
      title={`${label}: ${display} (${meta.label})`}
      className={`flex min-w-0 flex-col justify-between gap-1 border-2 border-ink p-2 ${meta.classes}`}
    >
      <span className="text-[11px] font-semibold opacity-80">{label}</span>
      <span className="truncate font-display text-lg leading-tight">{display}</span>
      <span className="text-xs font-bold" aria-hidden="true">
        {meta.symbol} {hint ?? (comparison === "correct" ? t("table.match") : t("table.noMatch"))}
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
      <div className="border-3 border-dashed border-bone/40 p-8 text-center text-sm text-steel">
        {t("table.empty")}
      </div>
    );
  }
  return (
    <ol className="space-y-5" role="table" aria-label={t("table.history")}>
      {guesses
        .map((g, i) => ({ g, n: i + 1 }))
        .reverse()
        .map(({ g, n }) => (
          <li
            key={`${g.fighter_id}-${n}`}
            role="row"
            className={`flex gap-3 border-3 border-ink bg-bone p-3 text-ink ${g.correct ? "shadow-blood-lg" : "shadow-[6px_6px_0_0_#7A0A06]"}`}
          >
            <div className="hidden flex-col items-center gap-2 sm:flex">
              <FighterPhoto name={g.fighter_name} url={g.photo_url} credit={g.photo_credit} size="md" />
              <span className="font-display text-sm">{t("table.guessN", { n })}</span>
            </div>
            <div className="min-w-0 flex-1">
              <div className="mb-2 flex items-center gap-2">
                <FighterPhoto name={g.fighter_name} url={g.photo_url} credit={g.photo_credit} size="sm" className="sm:hidden" />
                <p className="min-w-0 flex-1 truncate font-display text-2xl uppercase leading-none">
                  <span className="sm:hidden">{n}. </span>
                  {g.fighter_name}
                </p>
                {g.correct && (
                  <span className="shrink-0 border-2 border-ink bg-blood px-2 py-0.5 text-xs font-bold text-bone">
                    {t("table.correct")}
                  </span>
                )}
              </div>
              <div className="grid grid-cols-2 gap-1.5 sm:grid-cols-3 lg:grid-cols-6" role="rowgroup">
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
                <Cell label={t("table.attrRecord")} display={g.results.record.value} comparison={g.results.record.comparison} />
                <Cell label={t("table.attrNation")} display={g.results.nationality.value} comparison={g.results.nationality.comparison} />
                <Cell label={t("table.attrEvent")} display={g.results.last_event.value} comparison={g.results.last_event.comparison} />
              </div>
            </div>
          </li>
        ))}
    </ol>
  );
}
