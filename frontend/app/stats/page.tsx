"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Header, Shell } from "@/components/Header";
import { fetchMyStats, type UserStats } from "@/lib/api";
import { LangProvider, useLang } from "@/lib/i18n";
import { useAuth } from "@/lib/useAuth";

function Card({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="border-3 border-bone bg-ink p-4 shadow-blood">
      <p className="text-xs uppercase tracking-widest text-steel">{label}</p>
      <p className="font-display text-4xl text-bone">{value}</p>
      {sub && <p className="mt-1 text-xs text-steel">{sub}</p>}
    </div>
  );
}

/**
 * Personal statistics for the signed-in player: days played, accuracy,
 * tries, streaks, guess distribution and recent games. Guests get an
 * invitation to join; the API answers 401 for them.
 */
export default function StatsPage() {
  return (
    <LangProvider>
      <StatsView />
    </LangProvider>
  );
}

function StatsView() {
  const { t } = useLang();
  const { user, authLoading, logout } = useAuth();
  const [stats, setStats] = useState<UserStats | null>(null);
  const [pool, setPool] = useState<"all" | "men">("all");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (authLoading || !user) return;
    fetchMyStats()
      .then(setStats)
      .catch(() => setError(t("stats.loadError")));
  }, [authLoading, user, t]);

  const ps = stats?.pools[pool];
  const distMax = Math.max(0, ...Object.values(ps?.distribution ?? {}));
  const distKeys = Object.keys(ps?.distribution ?? {})
    .map(Number)
    .sort((a, b) => a - b);

  return (
    <Shell>
      <Header gameDate="" user={user} onSignIn={() => {}} onSignOut={() => void logout()} />
      <main className="mx-auto max-w-5xl space-y-8 px-4 py-8">
        <div>
          <h2 className="font-display text-5xl uppercase text-bone">
            {t("stats.titleA")} <span className="text-blood">{t("stats.titleB")}</span>
          </h2>
          <p className="mt-1 text-sm text-steel">
            <Link href="/" className="underline">{t("stats.back")}</Link>
          </p>
        </div>

        {authLoading ? (
          <p className="text-sm text-steel">{t("stats.loading")}</p>
        ) : !user ? (
          <p className="border-3 border-bone bg-ink px-3 py-2 text-sm text-bone">
            {t("stats.guest")}
          </p>
        ) : error ? (
          <p className="border-3 border-blood bg-bruise px-3 py-2 text-sm font-semibold text-bone" role="alert">
            {error}
          </p>
        ) : stats && ps ? (
          <>
            <div className="flex border-3 border-bone" role="group" aria-label={t("stats.poolGroup")}>
              {(["all", "men"] as const).map((p) => (
                <button
                  key={p}
                  onClick={() => setPool(p)}
                  aria-pressed={pool === p}
                  className={`px-3 py-1.5 font-semibold ${pool === p ? "bg-blood text-bone" : "bg-ink text-steel"}`}
                >
                  {t(p === "all" ? "stats.poolAll" : "stats.poolMen")}
                </button>
              ))}
            </div>

            <section aria-label={t("stats.days")}>
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                <Card label={t("stats.days")} value={String(stats.days_played)} sub={t("stats.gamesSub", { n: stats.games_total })} />
                <Card label={t("stats.accuracy")} value={`${Math.round(ps.win_rate * 100)}%`} sub={t("stats.wonSub", { won: ps.games_won, played: ps.games_played })} />
                <Card label={t("stats.triesDay")} value={stats.avg_tries_per_day.toFixed(1)} sub={t(pool === "all" ? "stats.poolAll" : "stats.poolMen")} />
                <Card label={t("stats.streak")} value={`${ps.current_streak}🔥`} sub={t("stats.best", { n: ps.max_streak })} />
              </div>
            </section>

            <section aria-label={t("stats.distTitle")} className="border-3 border-bone bg-ink p-4">
              <h3 className="mb-1 font-bold text-bone">{t("stats.distTitle")}</h3>
              <p className="mb-3 text-xs text-steel">{t("stats.distSub", { n: ps.avg_tries_to_win.toFixed(1) })}</p>
              {distKeys.length === 0 ? (
                <p className="text-sm text-steel">{t("stats.noWins")}</p>
              ) : (
                <ul className="space-y-1.5">
                  {distKeys.map((n) => (
                    <li key={n} className="flex items-center gap-2 text-sm">
                      <span className="w-8 shrink-0 font-mono text-xs text-steel">{n}</span>
                      <span
                        className="h-5 min-w-1 bg-blood"
                        style={{ width: `${distMax > 0 ? Math.max(3, ((ps.distribution[String(n)] ?? 0) / distMax) * 100) : 0}%` }}
                        role="img"
                        aria-label={t("stats.winsIn", { c: ps.distribution[String(n)] ?? 0, n })}
                      />
                      <span className="font-semibold text-bone">{ps.distribution[String(n)]}</span>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <section aria-label={t("stats.recent")} className="border-3 border-bone bg-ink p-4">
              <h3 className="mb-3 font-bold text-bone">{t("stats.recent")}</h3>
              {stats.recent_games.length === 0 ? (
                <p className="text-sm text-steel">{t("stats.nothing")}</p>
              ) : (
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="text-steel">
                      <th className="py-1">{t("stats.thDay")}</th>
                      <th className="py-1">{t("stats.thPool")}</th>
                      <th className="py-1 text-right">{t("stats.thTries")}</th>
                      <th className="py-1 text-right">{t("stats.thResult")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {stats.recent_games.map((g, i) => (
                      <tr key={`${g.date}-${g.pool}-${i}`} className="border-t border-bone/20 text-bone">
                        <td className="py-1 font-mono text-xs">{g.date}</td>
                        <td className="py-1">{t(g.pool === "all" ? "stats.poolNameAll" : "stats.poolNameMen")}</td>
                        <td className="py-1 text-right">{g.guesses}</td>
                        <td className="py-1 text-right">{g.won ? t("stats.solved") : t("stats.missed")}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>
          </>
        ) : null}
      </main>
    </Shell>
  );
}
