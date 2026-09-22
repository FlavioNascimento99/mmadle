"use client";

import type { LeaderboardEntry } from "@/lib/api";
import { useLang } from "@/lib/i18n";

/**
 * Public ranking table. Highlights the signed-in player's row (passed as
 * username, not id, since the backend never exposes ids to the public).
 */
export function LeaderboardTable({
  entries,
  currentUsername,
}: {
  entries: LeaderboardEntry[];
  currentUsername?: string | null;
}) {
  const { t } = useLang();
  if (entries.length === 0) {
    return <p className="text-sm text-steel">{t("leaderboard.empty")}</p>;
  }
  return (
    <table className="w-full text-left text-sm">
      <thead>
        <tr className="text-steel">
          <th className="py-1 pr-2">{t("leaderboard.thRank")}</th>
          <th className="py-1">{t("leaderboard.thPlayer")}</th>
          <th className="py-1 text-right">{t("leaderboard.thScore")}</th>
          <th className="py-1 text-right">{t("leaderboard.thWins")}</th>
          <th className="py-1 text-right">{t("leaderboard.thRate")}</th>
          <th className="py-1 text-right">{t("leaderboard.thStreak")}</th>
          <th className="py-1 text-right">{t("leaderboard.thBest")}</th>
          <th className="py-1 text-right">{t("leaderboard.thAvgTries")}</th>
        </tr>
      </thead>
      <tbody>
        {entries.map((e) => {
          const mine = currentUsername != null && e.username === currentUsername;
          return (
            <tr
              key={`${e.rank}-${e.username}`}
              className={`border-t border-bone/20 ${mine ? "bg-blood/20" : ""}`}
            >
              <td className={`py-1 pr-2 font-mono text-xs ${mine ? "font-bold text-blood" : "text-steel"}`}>
                {e.rank}
              </td>
              <td className="py-1 font-semibold text-bone">
                @{e.username}
                {mine && <span className="ml-1 text-xs font-normal text-blood">{t("leaderboard.you")}</span>}
              </td>
              <td className="py-1 text-right font-display text-bone">{e.score}</td>
              <td className="py-1 text-right text-bone">{e.wins}</td>
              <td className="py-1 text-right text-bone">{Math.round(e.win_rate * 100)}%</td>
              <td className="py-1 text-right text-bone">{e.current_streak}</td>
              <td className="py-1 text-right text-steel">{e.max_streak}</td>
              <td className="py-1 text-right text-steel">{e.avg_tries.toFixed(1)}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}