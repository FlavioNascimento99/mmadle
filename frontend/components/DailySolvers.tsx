"use client";

import { useEffect, useState } from "react";
import { fetchDailyStats, type Pool } from "@/lib/api";
import { useLang } from "@/lib/i18n";

type Props = {
  pool: Pool;
  refreshKey: number;
  /** This device's guess count — rendered in the same line so the board shows a single status number row. */
  guessCount: number;
  won: boolean;
};

/**
 * Single board status line: "Guesses N · M players solved today".
 * Previously the board rendered two lookalike number rows (DailySolvers on
 * top, "Guesses: N" below), which read as a duplicated stat. One centered
 * line with one aria-live region replaces both; each side degrades
 * gracefully when the other is unknown (solvers fetch failed / no guesses).
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

  return (
    <p className="text-center text-sm text-steel" aria-live="polite">
      {guessCount > 0 && (
        <>
          {t("board.guesses")} <span className="font-display text-xl text-bone">{guessCount}</span>
          {won ? t("board.solved") : ""}
        </>
      )}
      {guessCount > 0 && solvers !== null && (
        <span aria-hidden="true" className="mx-2 text-steel/60">
          ·
        </span>
      )}
      {solvers !== null && (
        <>
          <span className="font-display text-xl text-bone">{solvers}</span>{" "}
          {t(solvers === 1 ? "solvers.one" : "solvers.many", { n: solvers })}
        </>
      )}
    </p>
  );
}
