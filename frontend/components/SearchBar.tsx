"use client";

import { useEffect, useRef, useState } from "react";
import { searchFighters, type SearchResult } from "@/lib/api";

type Props = {
  disabled: boolean;
  guessedIds: Set<number>;
  onSelect: (fighter: SearchResult) => void;
};

export function SearchBar({ disabled, guessedIds, onSelect }: Props) {
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
        setResults(await searchFighters(query.trim()));
        setOpen(true);
      } finally {
        setLoading(false);
      }
    }, 220);
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [query]);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (boxRef.current && !boxRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

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
        placeholder={disabled ? "Game complete — come back tomorrow!" : "🔎 Search fighter... (e.g. Topuria)"}
        className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-4 py-3 text-zinc-100 placeholder:text-zinc-500 focus:border-red-500 focus:outline-none disabled:opacity-50"
      />
      {loading && (
        <p className="mt-1 text-xs text-zinc-500" role="status">
          Searching…
        </p>
      )}
      {open && results.length > 0 && (
        <ul
          role="listbox"
          aria-label="Matching fighters"
          className="absolute z-10 mt-1 max-h-72 w-full overflow-auto rounded-xl border border-zinc-700 bg-zinc-900 shadow-xl"
        >
          {results.map((f) => {
            const already = guessedIds.has(f.id);
            return (
              <li key={f.id}>
                <button
                  type="button"
                  role="option"
                  aria-selected="false"
                  disabled={already}
                  onClick={() => pick(f)}
                  className="flex w-full items-center justify-between gap-2 px-4 py-2.5 text-left hover:bg-zinc-800 disabled:opacity-40"
                >
                  <span>
                    <span className="block font-medium text-zinc-100">
                      {f.name}
                      {already ? " (guessed)" : ""}
                    </span>
                    <span className="block text-xs text-zinc-400">
                      {[f.nickname && `“${f.nickname}”`, f.division, f.nationality]
                        .filter(Boolean)
                        .join(" · ")}
                    </span>
                  </span>
                  <span aria-hidden="true" className="text-zinc-500">
                    →
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
      {open && query.trim().length >= 2 && !loading && results.length === 0 && (
        <p className="mt-1 text-xs text-zinc-500">No fighters match “{query}”.</p>
      )}
    </div>
  );
}
