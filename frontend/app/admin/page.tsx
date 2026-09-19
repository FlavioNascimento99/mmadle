"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { Header, Shell } from "@/components/Header";
import {
  fetchAdminUsers,
  fetchCFWorkers,
  fetchOverview,
  setUserActive,
  type AdminUser,
  type CFWorkers,
  type MetricsOverview,
  type UsersPage,
} from "@/lib/api";
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

function Bars({ rows, max }: { rows: { date: string; count: number }[]; max: number }) {
  const { t } = useLang();
  if (rows.length === 0) return <p className="text-sm text-steel">{t("admin.noData")}</p>;
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
  return (
    <LangProvider>
      <AdminView />
    </LangProvider>
  );
}

function AdminView() {
  const { t } = useLang();
  const { user, authLoading, logout } = useAuth();
  const [days, setDays] = useState(30);
  const [overview, setOverview] = useState<MetricsOverview | null>(null);
  const [cf, setCf] = useState<CFWorkers | "missing" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // Accounts listing (independent from the metrics above).
  const USERS_LIMIT = 20;
  const [usersPage, setUsersPage] = useState<UsersPage | null>(null);
  const [usersQ, setUsersQ] = useState("");
  const [usersOffset, setUsersOffset] = useState(0);
  const [usersLoading, setUsersLoading] = useState(false);
  const [usersError, setUsersError] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<number | null>(null);
  const usersReq = useRef(0);

  const loadUsers = useCallback(async (q: string, offset: number) => {
    const id = ++usersReq.current;
    setUsersLoading(true);
    setUsersError(null);
    try {
      const page = await fetchAdminUsers(q, USERS_LIMIT, offset);
      if (usersReq.current !== id) return; // superseded by a newer search
      setUsersPage(page);
    } catch {
      if (usersReq.current !== id) return;
      setUsersError(t("admin.usersError"));
    } finally {
      if (usersReq.current === id) setUsersLoading(false);
    }
  }, [t]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [ov, workers] = await Promise.all([fetchOverview(days), fetchCFWorkers(days)]);
      setOverview(ov);
      setCf(workers ?? "missing");
    } catch {
      setError(t("admin.loadError"));
    } finally {
      setLoading(false);
    }
  }, [days, t]);

  useEffect(() => {
    if (!authLoading && user?.role === "admin") void load();
  }, [authLoading, user, load]);

  const isAdmin = user?.role === "admin";

  // Live search (debounced) + pagination for the accounts listing.
  useEffect(() => {
    if (authLoading || !isAdmin) return;
    const id = setTimeout(() => void loadUsers(usersQ, usersOffset), 250);
    return () => clearTimeout(id);
  }, [authLoading, isAdmin, usersQ, usersOffset, loadUsers]);

  const onSearchUsers = (q: string) => {
    setUsersQ(q);
    setUsersOffset(0);
  };

  const onToggleActive = async (u: AdminUser) => {
    if (u.id === user?.id || togglingId !== null) return;
    setTogglingId(u.id);
    setUsersError(null);
    try {
      await setUserActive(u.id, !u.is_active);
      await loadUsers(usersQ, usersOffset);
    } catch {
      setUsersError(t("admin.toggleError"));
    } finally {
      setTogglingId(null);
    }
  };

  const fmtDateTime = (iso: string) => {
    const d = new Date(iso);
    return Number.isNaN(d.getTime())
      ? iso
      : d.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
  };

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
              {t("admin.titleA")}<span className="text-blood">{t("admin.titleB")}</span>
            </h2>
            <p className="mt-1 text-sm text-steel">
              <Link href="/" className="underline">{t("stats.back")}</Link>
            </p>
          </div>
          <div className="flex border-3 border-bone" role="group" aria-label={t("admin.window")}>
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
          <p className="text-sm text-steel">{t("admin.loading")}</p>
        ) : !isAdmin ? (
          <p className="border-3 border-blood bg-bruise px-3 py-2 text-sm font-semibold text-bone" role="alert">
            {t("admin.locked", { detail: user ? t("admin.lockedPlayer") : t("admin.lockedGuest") })}
          </p>
        ) : error ? (
          <p className="border-3 border-blood bg-bruise px-3 py-2 text-sm font-semibold text-bone" role="alert">
            {error}
          </p>
        ) : (
          overview && (
            <>
              <section aria-label={t("admin.game")}>
                <h3 className="mb-3 font-display text-2xl uppercase text-bone">{t("admin.game")} <span className="text-sm text-steel">{t("admin.gameNote")}</span></h3>
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                  <Card label={t("admin.games")} value={String(overview.games_total)} sub={t("admin.since", { date: overview.since })} />
                  <Card label={t("admin.winRate")} value={`${Math.round(overview.win_rate * 100)}%`} sub={t("admin.wonSub", { n: overview.games_won })} />
                  <Card label={t("admin.avgWin")} value={overview.avg_guesses_to_win.toFixed(1)} />
                  <Card label={t("admin.accounts")} value={String(overview.signups_total)} sub={t("admin.signupsSub")} />
                </div>
              </section>

              <section aria-label={t("admin.users")}>
                <h3 className="mb-3 font-display text-2xl uppercase text-bone">{t("admin.users")}</h3>
                {usersPage && (
                  <div className="mb-4 grid grid-cols-2 gap-4 sm:grid-cols-3">
                    <Card label={t("admin.accounts")} value={String(usersPage.total)} />
                    <Card label={t("admin.active")} value={String(usersPage.active)} />
                    <Card label={t("admin.deactivated")} value={String(usersPage.total - usersPage.active)} />
                  </div>
                )}
                <div className="border-3 border-bone bg-ink p-4">
                  <div className="mb-3">
                    <input
                      type="search"
                      value={usersQ}
                      onChange={(e) => onSearchUsers(e.target.value)}
                      placeholder={t("admin.searchUsers")}
                      aria-label={t("admin.searchUsers")}
                      className="w-full border-3 border-bone bg-bone px-3 py-1.5 text-sm text-ink placeholder:text-steel"
                    />
                  </div>
                  {usersError ? (
                    <p className="text-sm font-semibold text-blood" role="alert">{usersError}</p>
                  ) : usersPage === null || usersLoading ? (
                    <p className="text-sm text-steel">{t("admin.loading")}</p>
                  ) : usersPage.users.length === 0 ? (
                    <p className="text-sm text-steel">{t("admin.noData")}</p>
                  ) : (
                    <>
                      <div className="overflow-x-auto">
                        <table className="w-full text-left text-sm">
                          <thead>
                            <tr className="text-steel">
                              <th className="py-1 pr-2">{t("admin.thUser")}</th>
                              <th className="py-1 pr-2">{t("admin.thRole")}</th>
                              <th className="py-1 pr-2">{t("admin.thStatus")}</th>
                              <th className="py-1 pr-2">{t("admin.thCreated")}</th>
                              <th className="py-1 pr-2">{t("admin.thLastLogin")}</th>
                              <th className="py-1 pr-2 text-right">{t("admin.thGames")}</th>
                              <th className="py-1 pr-2 text-right">{t("admin.thWon")}</th>
                              <th className="py-1 text-right">{t("admin.thActions")}</th>
                            </tr>
                          </thead>
                          <tbody>
                            {usersPage.users.map((u) => {
                              const self = u.id === user?.id;
                              return (
                                <tr key={u.id} className="border-t border-bone/20 text-bone">
                                  <td className="py-1.5 pr-2 font-semibold">
                                    @{u.username}
                                    {self && <span className="ml-1 text-xs text-steel">({t("admin.you")})</span>}
                                  </td>
                                  <td className="py-1.5 pr-2">{u.role}</td>
                                  <td className="py-1.5 pr-2">
                                    <span className={`inline-block border-2 border-ink px-1.5 py-0.5 text-xs font-bold ${u.is_active ? "bg-blood text-bone" : "bg-bone text-ink"}`}>
                                      {u.is_active ? t("admin.active") : t("admin.inactive")}
                                    </span>
                                  </td>
                                  <td className="whitespace-nowrap py-1.5 pr-2 font-mono text-xs">{fmtDateTime(u.created_at)}</td>
                                  <td className="whitespace-nowrap py-1.5 pr-2 font-mono text-xs">
                                    {u.last_login_at ? fmtDateTime(u.last_login_at) : t("admin.never")}
                                  </td>
                                  <td className="py-1.5 pr-2 text-right">{u.games}</td>
                                  <td className="py-1.5 pr-2 text-right">{u.won}</td>
                                  <td className="py-1.5 text-right">
                                    {self ? (
                                      <span className="text-xs text-steel">—</span>
                                    ) : (
                                      <button
                                        type="button"
                                        disabled={togglingId !== null}
                                        onClick={() => void onToggleActive(u)}
                                        title={u.is_active ? t("admin.deactivate") : t("admin.activate")}
                                        aria-label={`${u.is_active ? t("admin.deactivate") : t("admin.activate")} @${u.username}`}
                                        aria-pressed={u.is_active}
                                        className={`press border-2 border-bone px-2 py-0.5 text-xs font-bold disabled:opacity-50 ${u.is_active ? "bg-ink text-steel" : "bg-blood text-bone"}`}
                                      >
                                        {togglingId === u.id ? "…" : u.is_active ? t("admin.deactivate") : t("admin.activate")}
                                      </button>
                                    )}
                                  </td>
                                </tr>
                              );
                            })}
                          </tbody>
                        </table>
                      </div>
                      <div className="mt-3 flex items-center justify-between text-sm">
                        <span className="text-steel">
                          {t("admin.pageOf", {
                            from: usersPage.total === 0 ? 0 : usersOffset + 1,
                            to: Math.min(usersOffset + USERS_LIMIT, usersPage.total),
                            total: usersPage.total,
                          })}
                        </span>
                        <span className="flex gap-2">
                          <button
                            type="button"
                            disabled={usersOffset === 0 || usersLoading}
                            onClick={() => setUsersOffset(Math.max(0, usersOffset - USERS_LIMIT))}
                            className="press border-2 border-bone px-2 py-0.5 font-semibold text-bone disabled:opacity-50"
                          >
                            {t("admin.prev")}
                          </button>
                          <button
                            type="button"
                            disabled={usersLoading || usersOffset + USERS_LIMIT >= usersPage.total}
                            onClick={() => setUsersOffset(usersOffset + USERS_LIMIT)}
                            className="press border-2 border-bone px-2 py-0.5 font-semibold text-bone disabled:opacity-50"
                          >
                            {t("admin.next")}
                          </button>
                        </span>
                      </div>
                    </>
                  )}
                </div>
              </section>

              <section aria-label={t("admin.guessesDay")} className="grid gap-6 md:grid-cols-2">
                <div className="border-3 border-bone bg-ink p-4">
                  <h4 className="mb-3 font-bold text-bone">{t("admin.guessesDay")}</h4>
                  <Bars rows={overview.guesses_by_day} max={Math.max(...overview.guesses_by_day.map((r) => r.count), 0)} />
                </div>
                <div className="border-3 border-bone bg-ink p-4">
                  <h4 className="mb-3 font-bold text-bone">{t("admin.playersDay")}</h4>
                  <Bars rows={overview.players_by_day} max={Math.max(...overview.players_by_day.map((r) => r.count), 0)} />
                </div>
              </section>

              <section aria-label={t("admin.byPool")} className="grid gap-6 md:grid-cols-2">
                <div className="border-3 border-bone bg-ink p-4">
                  <h4 className="mb-3 font-bold text-bone">{t("admin.byPool")}</h4>
                  <table className="w-full text-left text-sm">
                    <thead>
                      <tr className="text-steel">
                        <th className="py-1">{t("admin.thPool")}</th>
                        <th className="py-1 text-right">{t("admin.thGames")}</th>
                        <th className="py-1 text-right">{t("admin.thWon")}</th>
                        <th className="py-1 text-right">{t("admin.thGuesses")}</th>
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
                  <h4 className="mb-3 font-bold text-bone">{t("admin.topFighters")}</h4>
                  {overview.top_fighters.length === 0 ? (
                    <p className="text-sm text-steel">{t("admin.noGuesses")}</p>
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

              <section aria-label={t("admin.worker")}>
                <h3 className="mb-3 font-display text-2xl uppercase text-bone">{t("admin.worker")} <span className="text-sm text-steel">{t("admin.edge")}</span></h3>
                {cf === null ? (
                  <p className="text-sm text-steel">{t("stats.loading")}</p>
                ) : cf === "missing" ? (
                  <div className="border-3 border-bone bg-ink p-4 text-sm text-bone">
                    <p className="font-bold">{t("admin.cfMissingTitle")}</p>
                    <p className="mt-1 text-steel">{t("admin.cfMissingBody")}</p>
                    <pre className="mt-2 overflow-x-auto border-3 border-bone bg-ink p-3 font-mono text-xs text-steel">npx wrangler secret put CLOUDFLARE_API_TOKEN{"\n"}npx wrangler secret put CLOUDFLARE_ACCOUNT_ID</pre>
                    <p className="mt-2 text-steel">{t("admin.cfMissingNote")}</p>
                  </div>
                ) : (
                  <>
                    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
                      <Card label="Requests" value={String(cf.requests)} sub={t("admin.script", { name: cf.script })} />
                      <Card label="Errors" value={String(cf.errors)} sub={t("admin.errKinds")} />
                      <Card label={t("admin.errRate")} value={cf.requests > 0 ? `${((cf.errors / cf.requests) * 100).toFixed(2)}%` : "—"} />
                    </div>
                    <div className="mt-4 border-3 border-bone bg-ink p-4">
                      <h4 className="mb-3 font-bold text-bone">{t("admin.reqDay")}</h4>
                      <table className="w-full text-left font-mono text-xs">
                        <thead>
                          <tr className="text-steel">
                            <th className="py-1">{t("admin.thDay")}</th>
                            <th className="py-1 text-right">{t("admin.thReq")}</th>
                            <th className="py-1 text-right">{t("admin.thErr")}</th>
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
