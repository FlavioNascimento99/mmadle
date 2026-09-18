import { useCallback, useEffect, useState } from "react";
import type { Pool } from "./api";
import { parseStoredPool, POOL_STORAGE_KEY } from "./pool";

/** The selected game mode, remembered across visits. */
export function usePool(): [Pool, (pool: Pool) => void] {
  const [pool, setPoolState] = useState<Pool>("all");

  // Read after mount so the server render and first client render match.
  useEffect(() => {
    try {
      setPoolState(parseStoredPool(localStorage.getItem(POOL_STORAGE_KEY)));
    } catch {
      // Storage blocked: stay on all fighters.
    }
  }, []);

  const setPool = useCallback((next: Pool) => {
    setPoolState(next);
    try {
      localStorage.setItem(POOL_STORAGE_KEY, next);
    } catch {
      // Storage blocked: the choice lasts for this visit only.
    }
  }, []);

  return [pool, setPool];
}
