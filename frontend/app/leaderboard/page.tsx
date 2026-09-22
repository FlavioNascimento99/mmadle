"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { Header, Shell } from "@/components/Header";
import { LeaderboardTable } from "@/components/LeaderboardTable";
import {
  fetchLeaderboard,
  setLeaderboardOptIn,
  type Leaderboard as LeaderboardData,
  type Pool,
} from "@/lib/api";
import { LangProvider, useLang } from "@/lib/i18n";
import { useAuth } from "@/lib/useAuth";

/**
 * Public leaderboard of opted-in players: composite score (12 − tries per
 * solved daily game, min 1), wins, win rate, streaks and average tries.
 * Everyone can view; only signed-in accounts can toggle their visibility.
 */
export default function LeaderboardPage() {
  return (
    <LangProvider>
      <LeaderboardView />
    </LangProvider>
  );
}

function LeaderboardView() {
  const { t } = useLang();
  const { user, authLoading, logout } = useAuth();
  const [pool, setPool] = useState<Pool>("all");
  const [lb, setLb] = useState<LeaderboardData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [optIn, setOptIn] = useState(false);
  const [optBusy, setOptBusy] = useState(false);

  useEffect(() => {
    // The scored board is public; refetch on sign-in/out so `my` and the
    // highlight stay fresh.
    let cancelled = false;
    fetchLeaderboard(pool)
      .then((data) => {
        if (!cancelled) {
          setLb(data);
          setError(null);
        }
      })
      .catch(() => {
        if (!cancelled) setError(t("leaderboard.loadError"));
      });
    return () => {
      cancelled = true;
    };
  }, [pool, user?.id, t]);

  useEffect(() => {
    setOptIn(user?.leaderboard_opt_in ?? false);
  }, [user?.id, user?.leaderboard_opt_in]);

  const toggleOptIn = useCallback(async () => {
    if (optBusy || !user) return;
    setOptBusy(true);
    try {
      const target = !optIn;
      await setLeaderboardOptIn(target);
      setOptIn(target);
      setLb((prev) =>
        prev ? { ...prev, my: prev.my ? { ...prev.my, opt_in: target, entry: target ? prev.my.entry : null } : prev.my } : prev,
      );
      // Refresh so the entry appears (or leaves) immediately.
      const data = await fetchLeaderboard(pool);
      setLb(data);
    } catch {
      setError(t("leaderboard.optFailure"));
    } finally {
      setOptBusy(false);
    }
  }, [optBusy, optIn, pool, t, user]);

  const my = lb?.my;

  return (
    <Shell>
      <Header gameDate="" user={user} onSignIn={() => {}} onSignOut={() => void logout()} />
      <main className="mx-auto max-w-5xl space-y-6 px-4 py-8">
        <div>
          <h2 className="font-display text-5xl uppercase text-bone">
            <span className="text-blood">{t("leaderboard.title")}</span>
          </h2>
          <p className="mt-1 text-sm text-steel">
            <Link href="/" className="underline">{t("stats.back")}</Link>
          </p>
        </div>

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

        {!authLoading && !user && (
          <p className="border-3 border-bone bg-ink px-3 py-2 text-sm text-bone">
            {t("leaderboard.guestCta")}
          </p>
        )}

        {!authLoading && user && (
          <section aria-label={t("leaderboard.optIn.title")} className="border-3 border-bone bg-ink p-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="font-bold text-bone">{t("leaderboard.optIn.title")}</h3>
                <p className="mt-1 max-w-xl text-xs text-steel">{t("leaderboard.optIn.body")}</p>
              </div>
              <button
                onClick={() => void toggleOptIn()}
                disabled={optBusy}
                role="switch"
                aria-checked={optIn}
                aria-label={t("leaderboard.optIn.title")}
                className={`press whitespace-nowrap border-2 px-4 py-1.5 font-mono text-xs font-bold uppercase tracking-widest ${
                  optIn ? "border-bone bg-bone text-ink" : "border-bone/25 bg-transparent text-steel"
                } ${optBusy ? "opacity-60" : ""}`}
              >
                {t(optIn ? "leaderboard.optIn.on" : "leaderboard.optIn.off")}
              </button>
            </div>
            {my && my.entry && (
              <p className="mt-3 text-xs text-steel">
                {t("leaderboard.youRank", { rank: my.entry.rank, score: my.entry.score })}
              </p>
            )}
          </section>
        )}

        {error ? (
          <p className="border-3 border-blood bg-bruise px-3 py-2 text-sm font-semibold text-bone" role="alert">
            {error}
          </p>
        ) : lb ? (
          <section aria-label={t("leaderboard.title")} className="border-3 border-bone bg-ink p-4">
            <LeaderboardTable entries={lb.rankings} currentUsername={user?.username} />
            <p className="mt-3 font-mono text-[11px] text-ash">{t("leaderboard.scoreNote")}</p>
          </section>
        ) : (
          <p className="text-sm text-steel">{t("leaderboard.loading")}</p>
        )}
      </main>
    </Shell>
  );
}