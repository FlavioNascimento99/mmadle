"use client";

import { useCallback, useRef, useState } from "react";
import { listFighters, type Pool, type SearchResult } from "@/lib/api";
import { useClickOutside } from "@/lib/useClickOutside";
import { FighterOption } from "./FighterOption";

type Props = {
  pool: Pool;
  disabled: boolean;
  guessedIds: Set<number>;
  onSelect: (fighter: SearchResult) => void;
};

export function RosterList({ pool, disabled, guessedIds, onSelect }: Props) {
  const [open, setOpen] = useState(false);
  const [roster, setRoster] = useState<SearchResult[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const boxRef = useRef<HTMLDivElement>(null);

  const close = useCallback(() => setOpen(false), []);
  useClickOutside(boxRef, close);

  // The roster is fetched on first open and kept; a failed load retries on the next open.
  const toggle = async () => {
    const next = !open;
    setOpen(next);
    if (!next || roster) return;
    setError(null);
    try {
      setRoster(await listFighters(pool));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not load fighters");
    }
  };

  const pick = (f: SearchResult) => {
    onSelect(f);
    setOpen(false);
  };

  return (
    <div ref={boxRef} className="relative shrink-0">
      <button
        type="button"
        disabled={disabled}
        onClick={toggle}
        aria-expanded={open}
        aria-controls="roster-list"
        className="press w-full border-3 border-ink bg-blood px-4 py-3.5 text-lg font-bold text-bone shadow-[4px_4px_0_0_#FAFAF7] disabled:opacity-50"
      >
        All fighters <span aria-hidden="true">{open ? "▴" : "▾"}</span>
      </button>
      {open && (
        <div
          id="roster-list"
          className="absolute right-0 z-10 mt-2 w-[min(28rem,calc(100vw-2rem))] overflow-hidden border-3 border-ink bg-bone text-ink shadow-blood-lg"
        >
          {error ? (
            <p className="px-4 py-3 text-sm font-semibold text-blood" role="alert">
              {error}
            </p>
          ) : roster === null ? (
            <p className="px-4 py-3 text-sm" role="status">
              Loading fighters…
            </p>
          ) : (
            <>
              <p className="border-b-3 border-ink bg-ink px-4 py-2 text-sm font-semibold text-bone">
                {roster.length} fighters
              </p>
              <ul role="listbox" aria-label="All fighters" className="max-h-96 overflow-auto">
                {roster.map((f) => (
                  <FighterOption key={f.id} fighter={f} guessed={guessedIds.has(f.id)} onPick={pick} />
                ))}
              </ul>
            </>
          )}
        </div>
      )}
    </div>
  );
}
