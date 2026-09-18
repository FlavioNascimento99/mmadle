import { useCallback, useEffect, useState } from "react";
import type { GuessOutcome, Pool } from "./api";
import { parseSavedGuesses, storageKey } from "./game";

type SavedGame = { key: string; guesses: GuessOutcome[] };

function readSaved(key: string): GuessOutcome[] {
  try {
    return parseSavedGuesses(localStorage.getItem(key));
  } catch {
    return [];
  }
}

/**
 * Guesses for one game (date + mode), persisted in localStorage. Guesses are
 * held together with their storage key so switching modes can never write one
 * game's guesses under another game's key.
 */
export function useSavedGuesses(gameDate: string, pool: Pool) {
  const key = gameDate ? storageKey(gameDate, pool) : "";
  const [game, setGame] = useState<SavedGame>({ key: "", guesses: [] });

  if (game.key !== key) {
    setGame({ key, guesses: key ? readSaved(key) : [] });
  }

  useEffect(() => {
    if (!game.key) return;
    try {
      localStorage.setItem(game.key, JSON.stringify(game.guesses));
    } catch {
      // Storage full/blocked: game still works in-memory.
    }
  }, [game]);

  const addGuess = useCallback(
    (outcome: GuessOutcome) => setGame((prev) => ({ ...prev, guesses: [...prev.guesses, outcome] })),
    [],
  );

  return { guesses: game.key === key ? game.guesses : [], addGuess };
}
