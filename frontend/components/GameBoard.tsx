"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  fetchMyGuesses,
  fetchToday,
  importGuesses,
  submitGuess,
  type AuthUser,
  type GuessOutcome,
  type SearchResult,
} from "@/lib/api";
import { shareText } from "@/lib/game";
import { usePool } from "@/lib/usePool";
import { useSavedGuesses } from "@/lib/useSavedGuesses";
import { useLang } from "@/lib/i18n";
import { SearchBar } from "./SearchBar";
import { DailySolvers } from "./DailySolvers";
import { GuessTable } from "./GuessTable";
import { HintPanel } from "./HintPanel";
import { ModeToggle } from "./ModeToggle";
import { RosterList } from "./RosterList";
import { WinReveal } from "./WinReveal";

/**
 * Daily game board. Guests play exactly as before (localStorage only).
 * Signed-in players get their server history restored on any device, and
 * device-only guesses sync into the account automatically, silently.
 */
export function GameBoard({ user }: { user: AuthUser | null }) {
  const { t } = useLang();
  const [gameDate, setGameDate] = useState("");
  const [pool, setPool] = usePool();
  const { guesses, addGuess, setGuesses } = useSavedGuesses(gameDate, pool);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const [copied, setCopied] = useState(false);
  const [serverGuesses, setServerGuesses] = useState<GuessOutcome[] | null>(null);
  const [importing, setImporting] = useState(false);
  // Account+pool+date keys already synced (or syncing), so sign-in, pool
  // switches and repeated renders never double-submit the same import.
  const importTried = useRef<Set<string>>(new Set());

  useEffect(() => {
    (async () => {
      try {
        setGameDate((await fetchToday()).date);
      } catch {
        setError(t("board.loadError"));
      } finally {
        setReady(true);
      }
    })();
  }, [t]);

  // Signed-in: load the server history for this game. When this device has
  // nothing saved, restore it directly; device-only guesses auto-sync below.
  useEffect(() => {
    if (!user || !gameDate) {
      setServerGuesses(null);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const mine = await fetchMyGuesses(pool, gameDate);
        if (!cancelled) setServerGuesses(mine);
      } catch {
        if (!cancelled) setServerGuesses(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [user, gameDate, pool]);

  useEffect(() => {
    if (user && serverGuesses && serverGuesses.length > 0 && guesses.length === 0) {
      setGuesses(serverGuesses);
    }
  }, [user, serverGuesses, guesses.length, setGuesses]);

  const guessedIds = useMemo(
    () => new Set(guesses.map((g) => g.fighter_id)),
    [guesses],
  );
  const winner = guesses.find((g) => g.correct);
  const won = winner !== undefined;

  const serverIds = useMemo(
    () => new Set((serverGuesses ?? []).map((g) => g.fighter_id)),
    [serverGuesses],
  );
  const localOnlyIds = useMemo(
    () => guesses.map((g) => g.fighter_id).filter((id) => !serverIds.has(id)),
    [guesses, serverIds],
  );
  // Device guesses the account lacks sync automatically, silently.
  // Failures surface in the error line and retry on the next board change
  // (the key is released), so a blip never strands guesses or the error.
  useEffect(() => {
    if (!user || !gameDate || serverGuesses === null || localOnlyIds.length === 0 || importing) {
      return;
    }
    const key = `${user.id}|${pool}|${gameDate}`;
    if (importTried.current.has(key)) return;
    importTried.current.add(key);
    (async () => {
      setImporting(true);
      try {
        const result = await importGuesses(pool, gameDate, localOnlyIds);
        const merged = [...guesses];
        const known = new Set(merged.map((g) => g.fighter_id));
        for (const outcome of result.guesses) {
          if (!known.has(outcome.fighter_id)) merged.push(outcome);
        }
        setGuesses(merged);
        setServerGuesses(merged);
        setError(null);
      } catch {
        importTried.current.delete(key);
        setError(t("board.syncFailed"));
      } finally {
        setImporting(false);
      }
    })();
  }, [user, gameDate, pool, serverGuesses, localOnlyIds, guesses, importing, setGuesses, t]);

  const onSelect = useCallback(
    async (fighter: SearchResult) => {
      if (won || submitting || guessedIds.has(fighter.id)) return;
      setSubmitting(true);
      setError(null);
      try {
        // The cookie (when signed in) makes the backend record the guess;
        // the local copy keeps working for guests and offline.
        addGuess(await submitGuess(fighter.id, pool));
      } catch {
        setError(t("board.guessFailed"));
      } finally {
        setSubmitting(false);
      }
    },
    [won, submitting, guessedIds, addGuess, pool, t],
  );

  const share = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(shareText(gameDate, pool, guesses));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setError(t("board.clipboard"));
    }
  }, [guesses, gameDate, pool, t]);

  if (!ready) {
    return <p className="text-sm text-steel">{t("board.loading")}</p>;
  }

  return (
    <div className="space-y-6">
      <ModeToggle pool={pool} disabled={submitting} onChange={setPool} />
      <DailySolvers pool={pool} refreshKey={guesses.length} />

      <div className="flex items-start gap-2">
        <div className="min-w-0 flex-1">
          <SearchBar key={pool} pool={pool} disabled={won} guessedIds={guessedIds} onSelect={onSelect} />
        </div>
        <RosterList key={pool} pool={pool} disabled={won} guessedIds={guessedIds} onSelect={onSelect} />
      </div>

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

      {guesses.length > 0 && (
        <p className="text-sm text-steel" aria-live="polite">
          {t("board.guesses")} <span className="font-display text-xl text-bone">{guesses.length}</span>
          {won ? t("board.solved") : ""}
        </p>
      )}

      {gameDate && !won && <HintPanel pool={pool} guessCount={guesses.length} />}

      {winner && (
        <WinReveal winner={winner} attempts={guesses.length} copied={copied} onShare={share} />
      )}

      <GuessTable guesses={guesses} />
    </div>
  );
}
