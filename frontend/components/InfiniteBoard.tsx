"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  createRound,
  fetchInfiniteRecord,
  guessInfinite,
  type AuthUser,
  type InfiniteGuess,
  type SearchResult,
} from "@/lib/api";
import { useLang } from "@/lib/i18n";
import { usePool } from "@/lib/usePool";
import { FighterPhoto } from "./FighterPhoto";
import { GuessTable } from "./GuessTable";
import { ModeToggle } from "./ModeToggle";
import { RosterList } from "./RosterList";
import { SearchBar } from "./SearchBar";

const MAX_LIVES = 5;

const guestBestKey = (pool: string) => `mmadle-infinite-best-${pool}`;

function readGuestBest(pool: string): number {
  try {
    const raw = localStorage.getItem(guestBestKey(pool));
    const n = raw === null ? 0 : Number.parseInt(raw, 10);
    return Number.isFinite(n) && n > 0 ? n : 0;
  } catch {
    return 0;
  }
}

function writeGuestBest(pool: string, best: number) {
  try {
    localStorage.setItem(guestBestKey(pool), String(best));
  } catch {
    // Storage blocked: the run still works in-memory.
  }
}

function Lives({ left }: { left: number }) {
  const { t } = useLang();
  return (
    <p className="flex items-center gap-1.5" role="img" aria-label={t("inf.livesOf", { left, total: MAX_LIVES })}>
      <span className="text-xs font-bold uppercase tracking-widest text-steel">{t("inf.lives")}</span>
      {Array.from({ length: MAX_LIVES }, (_, i) => (
        <span key={i} aria-hidden="true" className={`font-display text-2xl leading-none ${i < left ? "text-blood" : "text-steel/30"}`}>
          ♥
        </span>
      ))}
    </p>
  );
}

/**
 * Infinity (survival) mode. Each round hides a random fighter: a miss costs
 * one of 5 lives, a solve restores (next round starts full), zero kills the
 * round and reveals the answer. Account streaks live server-side; guests
 * keep their best in this browser.
 */
export function InfiniteBoard({ user }: { user: AuthUser | null }) {
  const { t } = useLang();
  const [pool, setPool] = usePool();
  const [roundId, setRoundId] = useState<string | null>(null);
  const [lives, setLives] = useState(MAX_LIVES);
  const [guesses, setGuesses] = useState<InfiniteGuess[]>([]);
  const [starting, setStarting] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [best, setBest] = useState(0);
  const [streak, setStreak] = useState(0);
  const [newBest, setNewBest] = useState(false);

  const start = useCallback(
    async (nextPool: typeof pool) => {
      setStarting(true);
      setError(null);
      try {
        const round = await createRound(nextPool);
        setRoundId(round.round_id);
        setLives(round.lives);
        setGuesses([]);
        setNewBest(false);
        if (user) {
          const rec = await fetchInfiniteRecord(nextPool);
          setBest(rec.best_streak);
          setStreak(rec.current_streak);
        } else {
          setBest(readGuestBest(nextPool));
          setStreak(0);
        }
      } catch {
        setError(t("inf.startFailed"));
        setRoundId(null);
      } finally {
        setStarting(false);
      }
    },
    [user, t],
  );

  useEffect(() => {
    void start(pool);
    // Round lifecycle follows pool + account only: a language switch must
    // never abandon a live round, and logging in/out starts fresh so the
    // new identity's best loads with a clean round.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pool, user]);

  const roundOver = guesses.length > 0 && guesses[guesses.length - 1].round_over;
  const solved = roundOver && guesses[guesses.length - 1].solved;
  const answer = solved ? null : (guesses.find((g) => g.answer)?.answer ?? null);
  const guessedIds = useMemo(() => new Set(guesses.map((g) => g.fighter_id)), [guesses]);

  const onSelect = useCallback(
    async (fighter: SearchResult) => {
      if (!roundId || roundOver || submitting || guessedIds.has(fighter.id)) return;
      setSubmitting(true);
      setError(null);
      try {
        const outcome = await guessInfinite(roundId, fighter.id);
        setGuesses((prev) => [...prev, outcome]);
        setLives(outcome.lives_left);
        if (outcome.solved) {
          if (user) {
            if (outcome.streak !== undefined) setStreak(outcome.streak);
            if (outcome.best !== undefined) setBest(outcome.best);
            setNewBest(outcome.new_best === true);
          } else {
            const next = streak + 1;
            const top = Math.max(best, next);
            setStreak(next);
            setBest(top);
            setNewBest(next > best);
            writeGuestBest(pool, top);
          }
        } else if (outcome.round_over) {
          // Death resets the run; the account best survives server-side.
          if (user) {
            if (outcome.streak !== undefined) setStreak(outcome.streak);
            if (outcome.best !== undefined) setBest(outcome.best);
          } else {
            setStreak(0);
          }
          setNewBest(false);
        }
      } catch {
        setError(t("inf.guessFailed"));
      } finally {
        setSubmitting(false);
      }
    },
    [roundId, roundOver, submitting, guessedIds, user, pool, streak, best, t],
  );

  return (
    <div className="space-y-6">
      <ModeToggle pool={pool} disabled={submitting || starting} onChange={setPool} />

      <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
        <Lives left={lives} />
        <p className="text-sm text-steel" aria-live="polite">
          {t("inf.streakLine", { n: streak, m: best })}
          {newBest && <span className="ml-2 font-bold text-blood">{t("inf.newBest")}</span>}
        </p>
        <p className="text-xs text-steel">{user ? t("inf.bestAccount") : t("inf.bestGuest")}</p>
      </div>

      {starting ? (
        <p className="text-sm text-steel" role="status">
          {t("inf.starting")}
        </p>
      ) : (
        !roundOver && (
          <div className="flex items-start gap-2">
            <div className="min-w-0 flex-1">
              <SearchBar key={pool} pool={pool} disabled={false} guessedIds={guessedIds} onSelect={onSelect} />
            </div>
            <RosterList key={pool} pool={pool} disabled={false} guessedIds={guessedIds} onSelect={onSelect} />
          </div>
        )
      )}

      {submitting && (
        <p className="text-sm text-steel" role="status">
          {t("board.evaluating")}
        </p>
      )}
      {error && (
        <p className="border-3 border-blood bg-bruise px-3 py-2 text-sm font-semibold text-bone" role="alert">
          {error}
        </p>
      )}

      {solved && (
        <div className="border-3 border-bone bg-blood px-4 py-3 text-bone shadow-blood" role="status">
          <p className="font-display text-2xl uppercase">{t("inf.solved")}</p>
          <button
            onClick={() => void start(pool)}
            className="press mt-2 border-3 border-bone bg-ink px-4 py-2 font-semibold text-bone"
          >
            {t("inf.next")}
          </button>
        </div>
      )}

      {!solved && roundOver && answer && (
        <div className="border-3 border-ink bg-bone p-4 text-ink shadow-blood-lg" role="status">
          <p className="font-display text-2xl uppercase text-blood">{t("inf.dead")}</p>
          <div className="mt-3 flex items-center gap-3">
            <FighterPhoto name={answer.name} url={answer.photo_url} credit={answer.photo_credit} size="md" />
            <p className="font-display text-3xl uppercase leading-none">
              {t("inf.answerWas")} {answer.name}
            </p>
          </div>
          <button
            onClick={() => void start(pool)}
            className="press mt-3 border-3 border-ink bg-ink px-4 py-2 font-semibold text-bone"
          >
            {t("inf.retry")}
          </button>
        </div>
      )}

      {guesses.length > 0 && (
        <p className="text-sm text-steel" aria-live="polite">
          {t("board.guesses")} <span className="font-display text-xl text-bone">{guesses.length}</span>
        </p>
      )}

      <GuessTable guesses={guesses} />
    </div>
  );
}
