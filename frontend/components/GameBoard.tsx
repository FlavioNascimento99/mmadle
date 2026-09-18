"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { fetchToday, submitGuess, type SearchResult } from "@/lib/api";
import { shareText } from "@/lib/game";
import { usePool } from "@/lib/usePool";
import { useSavedGuesses } from "@/lib/useSavedGuesses";
import { SearchBar } from "./SearchBar";
import { GuessTable } from "./GuessTable";
import { HintPanel } from "./HintPanel";
import { ModeToggle } from "./ModeToggle";
import { RosterList } from "./RosterList";
import { WinReveal } from "./WinReveal";

export function GameBoard() {
  const [gameDate, setGameDate] = useState("");
  const [pool, setPool] = usePool();
  const { guesses, addGuess } = useSavedGuesses(gameDate, pool);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const [copied, setCopied] = useState(false);

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

  const guessedIds = useMemo(
    () => new Set(guesses.map((g) => g.fighter_id)),
    [guesses],
  );
  const winner = guesses.find((g) => g.correct);
  const won = winner !== undefined;

  const onSelect = useCallback(
    async (fighter: SearchResult) => {
      if (won || submitting || guessedIds.has(fighter.id)) return;
      setSubmitting(true);
      setError(null);
      try {
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
