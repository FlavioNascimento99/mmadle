"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Header, Shell } from "@/components/Header";
import { fetchMyStats, type UserStats } from "@/lib/api";
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
  const { user, authLoading, logout } = useAuth();
  const [stats, setStats] = useState<UserStats | null>(null);
  const [pool, setPool] = useState<"all" | "men">("all");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (authLoading || !user) return;
    fetchMyStats()
      .then(setStats)
      .catch((e) => setError(e instanceof Error ? e.message : "Could not load stats"));
  }, [authLoading, user]);

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
            My <span className="text-blood">stats</span>
          </h2>
          <p className="mt-1 text-sm text-steel">
            <Link href="/" className="underline">← Back to the game</Link>
          </p>
        </div>

        {authLoading ? (
          <p className="text-sm text-steel">Loading…</p>
        ) : !user ? (
          <p className="border-3 border-bone bg-ink px-3 py-2 text-sm text-bone">
            Sign in to keep your stats across devices — guests play without an account and leave no trace.
          </p>
        ) : error ? (
          <p className="border-3 border-blood bg-bruise px-3 py-2 text-sm font-semibold text-bone" role="alert">
            {error}
          </p>
        ) : stats && ps ? (
          <>
            <div className="flex border-3 border-bone" role="group" aria-label="Pool">
              {(["all", "men"] as const).map((p) => (
                <button
                  key={p}
                  onClick={() => setPool(p)}
                  aria-pressed={pool === p}
                  className={`px-3 py-1.5 font-semibold ${pool === p ? "bg-blood text-bone" : "bg-ink text-steel"}`}
                >
                  {p === "all" ? "All fighters" : "Men only"}
                </button>
              ))}
            </div>

            <section aria-label="Headline stats">
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                <Card label="Days played" value={String(stats.days_played)} sub={`${stats.games_total} games`} />
                <Card label="Accuracy" value={`${Math.round(ps.win_rate * 100)}%`} sub={`${ps.games_won} of ${ps.games_played} won`} />
                <Card label="Tries / day" value={stats.avg_tries_per_day.toFixed(1)} sub={`${pool === "all" ? "all fighters" : "men only"}`} />
                <Card label="Streak" value={`${ps.current_streak}🔥`} sub={`best ${ps.max_streak}`} />
              </div>
            </section>

            <section aria-label="Guess distribution" className="border-3 border-bone bg-ink p-4">
              <h3 className="mb-1 font-bold text-bone">Guesses needed to solve</h3>
              <p className="mb-3 text-xs text-steel">Won games only · average {ps.avg_tries_to_win.toFixed(1)} tries</p>
              {distKeys.length === 0 ? (
                <p className="text-sm text-steel">No wins yet — solve one to start the chart.</p>
              ) : (
                <ul className="space-y-1.5">
                  {distKeys.map((n) => (
                    <li key={n} className="flex items-center gap-2 text-sm">
                      <span className="w-8 shrink-0 font-mono text-xs text-steel">{n}</span>
                      <span
                        className="h-5 min-w-1 bg-blood"
                        style={{ width: `${distMax > 0 ? Math.max(3, ((ps.distribution[String(n)] ?? 0) / distMax) * 100) : 0}%` }}
                        role="img"
                        aria-label={`${ps.distribution[String(n)]} wins in ${n} tries`}
                      />
                      <span className="font-semibold text-bone">{ps.distribution[String(n)]}</span>
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <section aria-label="Recent games" className="border-3 border-bone bg-ink p-4">
              <h3 className="mb-3 font-bold text-bone">Recent games</h3>
              {stats.recent_games.length === 0 ? (
                <p className="text-sm text-steel">Nothing played yet.</p>
              ) : (
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="text-steel">
                      <th className="py-1">Day</th>
                      <th className="py-1">Pool</th>
                      <th className="py-1 text-right">Tries</th>
                      <th className="py-1 text-right">Result</th>
                    </tr>
                  </thead>
                  <tbody>
                    {stats.recent_games.map((g, i) => (
                      <tr key={`${g.date}-${g.pool}-${i}`} className="border-t border-bone/20 text-bone">
                        <td className="py-1 font-mono text-xs">{g.date}</td>
                        <td className="py-1">{g.pool === "all" ? "All" : "Men"}</td>
                        <td className="py-1 text-right">{g.guesses}</td>
                        <td className="py-1 text-right">{g.won ? "✓ Solved" : "✗ Missed"}</td>
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
