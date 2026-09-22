"use client";

import { useEffect, useState } from "react";
import { fetchDailyStats, type Pool } from "@/lib/api";
import { useLang } from "@/lib/i18n";

type Props = {
  pool: Pool;
  refreshKey: number;
  /** This device's guess count. */
  guessCount: number;
  won: boolean;
};

const MAX_DOTS = 12;

/**
 * Game status card: today's attempts (visual dots + big count) beside the
 * global solvers count. One card, one aria-live region — replaces the old
 * pair of lookalike number rows.
 */
export function DailySolvers({ pool, refreshKey, guessCount, won }: Props) {
  const { t } = useLang();
  const [solvers, setSolvers] = useState<number | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const stats = await fetchDailyStats(pool);
        if (!cancelled) setSolvers(stats.solvers);
      } catch {
        if (!cancelled) setSolvers(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [pool, refreshKey]);

  if (solvers === null && guessCount === 0) return null;

  const shown = Math.min(guessCount, MAX_DOTS);
  const extra = guessCount - shown;

  return (
    <section
      aria-label={t("board.dailyGame")}
      aria-live="polite"
      className="border-3 border-ink bg-bone px-4 py-3 text-ink shadow-hard sm:px-5"
    >
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="microlabel text-bruise">
            {t("board.dailyGame")} · {t(pool === "men" ? "pool.men" : "pool.all")}
          </p>
          <div className="mt-1 flex items-baseline gap-2">
            <span className="font-display text-5xl leading-none">{guessCount}</span>
            <span className="font-mono text-[11px] font-bold uppercase tracking-[0.18em] text-ash">
              {t("status.attempts")}
              {won ? ` ${t("board.solved")}` : ""}
            </span>
            {won && (
              <span className="animate-land border-2 border-ink bg-blood px-2 py-0.5 font-mono text-[11px] font-bold uppercase tracking-[0.18em] text-bone">
                ✓ {t("table.correct")}
              </span>
            )}
          </div>
          {guessCount > 0 && (
            <div className="mt-2 flex items-center gap-1.5" aria-hidden="true">
              {Array.from({ length: shown }, (_, i) => (
                <span
                  key={i}
                  className={`h-3 w-3 border-2 border-ink ${i === shown - 1 && !won ? "bg-blood" : "bg-ink"}`}
                />
              ))}
              {extra > 0 && (
                <span className="font-mono text-[11px] font-bold text-ash">+{extra}</span>
              )}
            </div>
          )}
        </div>
        {solvers !== null && (
          <div className="shrink-0 border-l-2 border-ink/15 pl-4 text-right">
            <p className="font-display text-4xl leading-none">{solvers}</p>
            <p className="mt-1 font-mono text-[11px] font-bold uppercase tracking-[0.18em] text-ash">
              {t("status.solvedToday")}
            </p>
          </div>
        )}
      </div>
    </section>
  );
}
