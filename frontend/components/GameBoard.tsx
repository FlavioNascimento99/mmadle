"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  fetchToday,
  submitGuess,
  type GuessOutcome,
  type SearchResult,
} from "@/lib/api";
import { outcomeToEmoji, storageKey } from "@/lib/game";
import { SearchBar } from "./SearchBar";
import { GuessTable } from "./GuessTable";

export function GameBoard() {
  const [gameDate, setGameDate] = useState("");
  const [guesses, setGuesses] = useState<GuessOutcome[]>([]);
  const [won, setWon] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ready, setReady] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const today = await fetchToday();
        setGameDate(today.date);
        try {
          const saved = localStorage.getItem(storageKey(today.date));
          if (saved) {
            const parsed = JSON.parse(saved) as GuessOutcome[];
            setGuesses(parsed);
            setWon(parsed.some((g) => g.correct));
          }
        } catch {
          // Corrupt save: start fresh.
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : "Could not load game");
      } finally {
        setReady(true);
      }
    })();
  }, []);

  useEffect(() => {
    if (!ready || !gameDate) return;
    try {
      localStorage.setItem(storageKey(gameDate), JSON.stringify(guesses));
    } catch {
      // Storage full/blocked: game still works in-memory.
    }
  }, [guesses, gameDate, ready]);

  const guessedIds = useMemo(
    () => new Set(guesses.map((g) => g.fighter_id)),
    [guesses],
  );

  const onSelect = useCallback(
    async (fighter: SearchResult) => {
      if (won || submitting || guessedIds.has(fighter.id)) return;
      setSubmitting(true);
      setError(null);
      try {
        const outcome = await submitGuess(fighter.id);
        setGuesses((prev) => [...prev, outcome]);
        if (outcome.correct) setWon(true);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Guess failed");
      } finally {
        setSubmitting(false);
      }
    },
    [won, submitting, guessedIds],
  );

  const share = useCallback(async () => {
    const lines = guesses.map(outcomeToEmoji).join("\n");
    const text = `MMAdle ${gameDate} — ${guesses.length} ${
      guesses.length === 1 ? "try" : "tries"
    }\n${lines}`;
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setError("Clipboard blocked — select the text manually.");
    }
  }, [guesses, gameDate]);

  if (!ready) {
    return <p className="text-sm text-zinc-400">Loading today&apos;s game…</p>;
  }

  return (
    <div className="space-y-5">
      <SearchBar disabled={won} guessedIds={guessedIds} onSelect={onSelect} />

      {submitting && (
        <p className="text-sm text-zinc-400" role="status">
          Evaluating guess…
        </p>
      )}
      {error && (
        <p className="rounded-lg border border-red-500/40 bg-red-500/10 px-3 py-2 text-sm text-red-300" role="alert">
          {error}
        </p>
      )}

      {guesses.length > 0 && (
        <p className="text-sm text-zinc-400" aria-live="polite">
          Attempts: <span className="font-bold text-zinc-100">{guesses.length}</span>
          {won ? " — solved! 🎉" : ""}
        </p>
      )}

      {won && (
        <div className="rounded-xl border border-emerald-500/40 bg-emerald-500/10 p-4 text-center">
          <p className="text-lg font-extrabold text-emerald-300">
            🎉 Correct! You found today&apos;s fighter in {guesses.length}{" "}
            {guesses.length === 1 ? "try" : "tries"}.
          </p>
          <button
            type="button"
            onClick={share}
            className="mt-3 rounded-lg border border-emerald-500/50 px-4 py-1.5 text-sm font-semibold text-emerald-200 hover:bg-emerald-500/20"
          >
            {copied ? "Copied! ✓" : "Share result 📋"}
          </button>
        </div>
      )}

      <GuessTable guesses={guesses} />
    </div>
  );
}
