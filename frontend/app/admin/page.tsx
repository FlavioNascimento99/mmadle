"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { Header, Shell } from "@/components/Header";
import {
  fetchCFWorkers,
  fetchOverview,
  type CFWorkers,
  type MetricsOverview,
} from "@/lib/api";
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

function Bars({ rows, max }: { rows: { date: string; count: number }[]; max: number }) {
  if (rows.length === 0) return <p className="text-sm text-steel">No data in this window.</p>;
  return (
    <ul className="space-y-1.5">
      {rows.map((r) => (
        <li key={r.date} className="flex items-center gap-2 text-sm">
          <span className="w-24 shrink-0 font-mono text-xs text-steel">{r.date.slice(5)}</span>
          <span
            className="h-5 min-w-1 bg-blood"
            style={{ width: `${max > 0 ? Math.max(2, (r.count / max) * 100) : 0}%` }}
            role="img"
            aria-label={`${r.count} on ${r.date}`}
          />
          <span className="font-semibold text-bone">{r.count}</span>
        </li>
      ))}
    </ul>
  );
}

/**
 * Admin metrics interface (role-gated server-side too: non-admins get 404
 * from the API, so this page doubles as a friendly locked door).
 */
export default function AdminPage() {
  const { user, authLoading, logout } = useAuth();
  const [days, setDays] = useState(30);
  const [overview, setOverview] = useState<MetricsOverview | null>(null);
  const [cf, setCf] = useState<CFWorkers | "missing" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [ov, workers] = await Promise.all([fetchOverview(days), fetchCFWorkers(days)]);
      setOverview(ov);
      setCf(workers ?? "missing");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not load metrics");
    } finally {
      setLoading(false);
    }
  }, [days]);

  useEffect(() => {
    if (!authLoading && user?.role === "admin") void load();
  }, [authLoading, user, load]);

  const isAdmin = user?.role === "admin";

  return (
    <Shell>
      <Header
        gameDate=""
        user={user}
        onSignIn={() => {}}
        onSignOut={() => void logout()}
      />
      <main className="mx-auto max-w-5xl space-y-8 px-4 py-8">
        <div className="flex flex-wrap items-end justify-between gap-4">
          <div>
            <h2 className="font-display text-5xl uppercase text-bone">
              Met<span className="text-blood">rics</span>
            </h2>
            <p className="mt-1 text-sm text-steel">
              <Link href="/" className="underline">← Back to the game</Link>
            </p>
          </div>
          <div className="flex border-3 border-bone" role="group" aria-label="Window">
            {[7, 30, 90].map((d) => (
              <button
                key={d}
                onClick={() => setDays(d)}
                aria-pressed={days === d}
                className={`px-3 py-1.5 font-semibold ${days === d ? "bg-blood text-bone" : "bg-ink text-steel"}`}
              >
                {d}d
              </button>
            ))}
          </div>
        </div>

        {authLoading || (loading && isAdmin) ? (
          <p className="text-sm text-steel">Loading metrics…</p>
        ) : !isAdmin ? (
          <p className="border-3 border-blood bg-bruise px-3 py-2 text-sm font-semibold text-bone" role="alert">
            Admins only. {user ? "Your account is a player account." : "Sign in with an admin account."}
          </p>
        ) : error ? (
          <p className="border-3 border-blood bg-bruise px-3 py-2 text-sm font-semibold text-bone" role="alert">
            {error}
          </p>
        ) : (
          overview && (
            <>
              <section aria-label="Game">
                <h3 className="mb-3 font-display text-2xl uppercase text-bone">Game <span className="text-sm text-steel">(signed-in players; guests stay in their browsers)</span></h3>
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                  <Card label="Games" value={String(overview.games_total)} sub={`since ${overview.since}`} />
                  <Card label="Win rate" value={`${Math.round(overview.win_rate * 100)}%`} sub={`${overview.games_won} won`} />
                  <Card label="Avg guesses to win" value={overview.avg_guesses_to_win.toFixed(1)} />
                  <Card label="Accounts" value={String(overview.signups_total)} sub="total signups" />
                </div>
              </section>

              <section aria-label="Daily activity" className="grid gap-6 md:grid-cols-2">
                <div className="border-3 border-bone bg-ink p-4">
                  <h4 className="mb-3 font-bold text-bone">Guesses per day</h4>
                  <Bars rows={overview.guesses_by_day} max={Math.max(...overview.guesses_by_day.map((r) => r.count), 0)} />
                </div>
                <div className="border-3 border-bone bg-ink p-4">
                  <h4 className="mb-3 font-bold text-bone">Active players per day</h4>
                  <Bars rows={overview.players_by_day} max={Math.max(...overview.players_by_day.map((r) => r.count), 0)} />
                </div>
              </section>

              <section aria-label="Pools and fighters" className="grid gap-6 md:grid-cols-2">
                <div className="border-3 border-bone bg-ink p-4">
                  <h4 className="mb-3 font-bold text-bone">By pool</h4>
                  <table className="w-full text-left text-sm">
                    <thead>
                      <tr className="text-steel">
                        <th className="py-1">Pool</th>
                        <th className="py-1 text-right">Games</th>
                        <th className="py-1 text-right">Won</th>
                        <th className="py-1 text-right">Guesses</th>
                      </tr>
                    </thead>
                    <tbody>
                      {overview.by_pool.map((p) => (
                        <tr key={p.pool} className="border-t border-bone/20 text-bone">
                          <td className="py-1 font-semibold">{p.pool}</td>
                          <td className="py-1 text-right">{p.games}</td>
                          <td className="py-1 text-right">{p.won}</td>
                          <td className="py-1 text-right">{p.guesses}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                <div className="border-3 border-bone bg-ink p-4">
                  <h4 className="mb-3 font-bold text-bone">Most guessed fighters</h4>
                  {overview.top_fighters.length === 0 ? (
                    <p className="text-sm text-steel">No guesses yet.</p>
                  ) : (
                    <ol className="space-y-1 text-sm">
                      {overview.top_fighters.map((f, i) => (
                        <li key={f.fighter_id} className="flex justify-between gap-2 text-bone">
                          <span><span className="text-steel">{i + 1}.</span> {f.name}</span>
                          <span className="font-semibold">{f.guesses}</span>
                        </li>
                      ))}
                    </ol>
                  )}
                </div>
              </section>

              <section aria-label="Infrastructure">
                <h3 className="mb-3 font-display text-2xl uppercase text-bone">Worker <span className="text-sm text-steel">(Cloudflare edge)</span></h3>
                {cf === null ? (
                  <p className="text-sm text-steel">Loading…</p>
                ) : cf === "missing" ? (
                  <div className="border-3 border-bone bg-ink p-4 text-sm text-bone">
                    <p className="font-bold">Cloudflare API not connected.</p>
                    <p className="mt-1 text-steel">The dashboard already shows Worker requests, errors and CPU time. To embed them here, set the secrets:</p>
                    <pre className="mt-2 overflow-x-auto border-3 border-bone bg-ink p-3 font-mono text-xs text-steel">npx wrangler secret put CLOUDFLARE_API_TOKEN{"\n"}npx wrangler secret put CLOUDFLARE_ACCOUNT_ID</pre>
                    <p className="mt-2 text-steel">The token needs the Analytics:Read permission; it never leaves the server (the browser only sees these aggregated numbers).</p>
                  </div>
                ) : (
                  <>
                    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
                      <Card label="Requests" value={String(cf.requests)} sub={`script ${cf.script}`} />
                      <Card label="Errors" value={String(cf.errors)} sub="script threw / resources / internal" />
                      <Card label="Error rate" value={cf.requests > 0 ? `${((cf.errors / cf.requests) * 100).toFixed(2)}%` : "—"} />
                    </div>
                    <div className="mt-4 border-3 border-bone bg-ink p-4">
                      <h4 className="mb-3 font-bold text-bone">Requests per day (CPU p50 / p99 µs)</h4>
                      <table className="w-full text-left font-mono text-xs">
                        <thead>
                          <tr className="text-steel">
                            <th className="py-1">Day</th>
                            <th className="py-1 text-right">Req</th>
                            <th className="py-1 text-right">Err</th>
                            <th className="py-1 text-right">CPU p50</th>
                            <th className="py-1 text-right">CPU p99</th>
                          </tr>
                        </thead>
                        <tbody>
                          {cf.by_day.map((d) => (
                            <tr key={d.date} className="border-t border-bone/20 text-bone">
                              <td className="py-1">{d.date}</td>
                              <td className="py-1 text-right">{d.requests}</td>
                              <td className="py-1 text-right">{d.errors}</td>
                              <td className="py-1 text-right">{Math.round(d.cpu_time_p50_us)}</td>
                              <td className="py-1 text-right">{Math.round(d.cpu_time_p99_us)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </>
                )}
              </section>
            </>
          )
        )}
      </main>
    </Shell>
  );
}
