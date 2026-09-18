"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { searchFighters, type Pool, type SearchResult } from "@/lib/api";
import { useClickOutside } from "@/lib/useClickOutside";
import { FighterOption } from "./FighterOption";

type Props = {
  pool: Pool;
  disabled: boolean;
  guessedIds: Set<number>;
  onSelect: (fighter: SearchResult) => void;
};

export function SearchBar({ pool, disabled, guessedIds, onSelect }: Props) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const boxRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (timer.current) clearTimeout(timer.current);
    if (query.trim().length < 2) {
      setResults([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    timer.current = setTimeout(async () => {
      try {
        setResults(await searchFighters(query.trim(), pool));
        setOpen(true);
      } finally {
        setLoading(false);
      }
    }, 220);
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [query, pool]);

  const close = useCallback(() => setOpen(false), []);
  useClickOutside(boxRef, close);

  const pick = (f: SearchResult) => {
    onSelect(f);
    setQuery("");
    setResults([]);
    setOpen(false);
  };

  return (
    <div ref={boxRef} className="relative">
      <label htmlFor="fighter-search" className="sr-only">
        Search fighter
      </label>
      <input
        id="fighter-search"
        type="text"
        autoComplete="off"
        disabled={disabled}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onFocus={() => results.length > 0 && setOpen(true)}
        placeholder={disabled ? "Solved for today" : "Search fighters"}
        className="w-full border-3 border-ink bg-bone px-4 py-3.5 text-lg font-semibold text-ink shadow-blood placeholder:font-normal placeholder:text-ink/50 focus-visible:shadow-hard focus-visible:outline-blood disabled:opacity-50"
      />
      {loading && (
        <p className="mt-2 text-xs text-steel" role="status">
          Searching…
        </p>
      )}
      {open && results.length > 0 && (
        <ul
          role="listbox"
          aria-label="Matching fighters"
          className="absolute z-10 mt-2 max-h-80 w-full overflow-auto border-3 border-ink bg-bone shadow-blood-lg"
        >
          {results.map((f) => (
            <FighterOption key={f.id} fighter={f} guessed={guessedIds.has(f.id)} onPick={pick} />
          ))}
        </ul>
      )}
      {open && query.trim().length >= 2 && !loading && results.length === 0 && (
        <p className="mt-2 text-xs text-steel">No fighters match “{query}”. Try a first or last name, or browse all fighters.</p>
      )}
    </div>
  );
}
