"use client";

import { useEffect, useState } from "react";
import { fetchDailyStats, type Pool } from "@/lib/api";

/**
 * Public solvers count for today's daily game in the active pool.
 * Refreshes when the pool changes and after each submitted guess
 * (via refreshKey), so a fresh solve shows up without a reload.
 * Every solve counts, logged in or not (guests are keyed by an
 * anonymous browser identity issued on their first solve).
 */
export function DailySolvers({ pool, refreshKey }: { pool: Pool; refreshKey: number }) {
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

  if (solvers === null) return null;

  return (
    <p className="text-center text-sm text-steel" aria-live="polite">
      <span className="font-display text-xl text-bone">{solvers}</span>{" "}
      {solvers === 1 ? "player has" : "players have"} solved today
    </p>
  );
}
