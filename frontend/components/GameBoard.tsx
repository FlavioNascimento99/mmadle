"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
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
import { SearchBar } from "./SearchBar";
import { GuessTable } from "./GuessTable";
import { HintPanel } from "./HintPanel";
import { ModeToggle } from "./ModeToggle";
import { RosterList } from "./RosterList";
import { WinReveal } from "./WinReveal";

/**
 * Daily game board. Guests play exactly as before (localStorage only).
 * Signed-in players get their server history restored on any device, plus a
 * one-tap import when this device holds local guesses the account lacks.
 */
export function GameBoard({ user }: { user: AuthUser | null }) {
  const [gameDate, setGameDate] = useState("");
  const [pool, setPool] = usePool();
  const { guesses, addGuess, setGuesses } = useSavedGuesses(gameDate, pool);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const [copied, setCopied] = useState(false);
  const [serverGuesses, setServerGuesses] = useState<GuessOutcome[] | null>(null);
  const [importing, setImporting] = useState(false);
  const [importDismissed, setImportDismissed] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        setGameDate((await fetchToday()).date);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Could not load game");
      } finally {
        setReady(true);
      }
    })();
  }, []);

  // Signed-in: load the server history for this game. When this device has
  // nothing saved, restore it directly; otherwise offer an import below.
  useEffect(() => {
    if (!user || !gameDate) {
      setServerGuesses(null);
      setImportDismissed(false);
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
  const showImportBanner =
    user !== null &&
    serverGuesses !== null &&
    localOnlyIds.length > 0 &&
    !importDismissed &&
    !importing;

  const onImport = useCallback(async () => {
    if (!user || !gameDate || localOnlyIds.length === 0) return;
    setImporting(true);
    setError(null);
    try {
      const result = await importGuesses(pool, gameDate, localOnlyIds);
      const merged = [...guesses];
      const known = new Set(merged.map((g) => g.fighter_id));
      for (const outcome of result.guesses) {
        if (!known.has(outcome.fighter_id)) merged.push(outcome);
      }
      setGuesses(merged);
      setServerGuesses(merged);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Import failed");
    } finally {
      setImporting(false);
    }
  }, [user, gameDate, localOnlyIds, pool, guesses, setGuesses]);

  const onSelect = useCallback(
    async (fighter: SearchResult) => {
      if (won || submitting || guessedIds.has(fighter.id)) return;
      setSubmitting(true);
      setError(null);
      try {
        // The cookie (when signed in) makes the backend record the guess;
        // the local copy keeps working for guests and offline.
        addGuess(await submitGuess(fighter.id, pool));
      } catch (e) {
        setError(e instanceof Error ? e.message : "Guess failed");
      } finally {
        setSubmitting(false);
      }
    },
    [won, submitting, guessedIds, addGuess, pool],
  );

  const share = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(shareText(gameDate, pool, guesses));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setError("Clipboard blocked — select the text manually.");
    }
  }, [guesses, gameDate, pool]);

  if (!ready) {
    return <p className="text-sm text-steel">Loading today&apos;s game…</p>;
  }

  return (
    <div className="space-y-6">
      <ModeToggle pool={pool} disabled={submitting} onChange={setPool} />

      {showImportBanner && (
        <div className="border-3 border-bone bg-bruise px-3 py-2 text-sm text-bone" role="status">
          <p className="font-semibold">
            You have {localOnlyIds.length} {localOnlyIds.length === 1 ? "guess" : "guesses"} on this
            device that {user ? `@${user.username}` : "your account"} doesn&apos;t.
          </p>
          <div className="mt-2 flex gap-2">
            <button
              onClick={onImport}
              disabled={importing}
              className="press border-3 border-bone bg-blood px-3 py-1 font-semibold text-bone disabled:opacity-60"
            >
              {importing ? "Importing…" : "Import to my account"}
            </button>
            <button
              onClick={() => setImportDismissed(true)}
              className="press border-3 border-bone bg-ink px-3 py-1 font-semibold text-steel"
            >
              Dismiss
            </button>
          </div>
        </div>
      )}

      <div className="flex items-start gap-2">
        <div className="min-w-0 flex-1">
          <SearchBar key={pool} pool={pool} disabled={won} guessedIds={guessedIds} onSelect={onSelect} />
        </div>
        <RosterList key={pool} pool={pool} disabled={won} guessedIds={guessedIds} onSelect={onSelect} />
      </div>

      {submitting && (
        <p className="text-sm text-steel" role="status">
          Evaluating guess…
        </p>
      )}
      {error && (
        <p className="border-3 border-blood bg-bruise px-3 py-2 text-sm font-semibold text-bone" role="alert">
          {error}
        </p>
      )}

      {guesses.length > 0 && (
        <p className="text-sm text-steel" aria-live="polite">
          Guesses: <span className="font-display text-xl text-bone">{guesses.length}</span>
          {won ? ", solved" : ""}
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
