"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { searchFighters, type Pool, type SearchResult } from "@/lib/api";
import { useClickOutside } from "@/lib/useClickOutside";
import { useLang } from "@/lib/i18n";
import { FighterOption } from "./FighterOption";

type Props = {
  pool: Pool;
  disabled: boolean;
  guessedIds: Set<number>;
  onSelect: (fighter: SearchResult) => void;
};

const optionId = (index: number) => `search-option-${index}`;

function canStealFocus(): boolean {
  if (typeof window === "undefined") return true;
  try {
    // Touch devices: programmatic focus pops the keyboard, so only
    // auto-focus on fine-pointer (desktop/laptop) devices.
    return window.matchMedia("(pointer: fine)").matches;
  } catch {
    return true;
  }
}

export function SearchBar({ pool, disabled, guessedIds, onSelect }: Props) {
  const { t } = useLang();
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  /** Keyboard-highlighted result; Enter confirms it. */
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const boxRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const optionRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const queryToken = useRef(0);

  const focusInput = useCallback(() => {
    if (disabled || !canStealFocus()) return;
    requestAnimationFrame(() => inputRef.current?.focus());
  }, [disabled]);

  // Initial focus while the game is unsolved; remounts per pool (key={pool}).
  useEffect(() => {
    focusInput();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // If the board re-enables the input (new round), hand focus back.
  useEffect(() => {
    if (!disabled) focusInput();
  }, [disabled, focusInput]);

  useEffect(() => {
    if (timer.current) clearTimeout(timer.current);
    // Single character is enough to start searching (backend accepts q>=1,
    // capped at 8 rows ordered by name).
    if (query.trim().length < 1) {
      setResults([]);
      setActiveIndex(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    const token = ++queryToken.current;
    timer.current = setTimeout(async () => {
      try {
        const rows = await searchFighters(query.trim(), pool);
        // Drop stale responses when the user kept typing.
        if (queryToken.current !== token) return;
        setResults(rows);
        setActiveIndex(firstSelectable(rows, guessedIds));
        setOpen(rows.length > 0);
      } finally {
        if (queryToken.current === token) setLoading(false);
      }
    }, 220);
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [query, pool, guessedIds]);

  const close = useCallback(() => {
    setOpen(false);
    setActiveIndex(null);
  }, []);
  useClickOutside(boxRef, close);

  const pick = (f: SearchResult) => {
    if (guessedIds.has(f.id)) return;
    onSelect(f);
    setQuery("");
    setResults([]);
    close();
    focusInput();
  };

  const focusOption = useCallback((index: number | null) => {
    if (index === null) return;
    setActiveIndex(index);
    requestAnimationFrame(() => optionRefs.current[index]?.scrollIntoView({ block: "nearest" }));
  }, []);

  const step = useCallback(
    (from: number | null, delta: 1 | -1): number | null => {
      if (results.length === 0) return null;
      let i = from ?? (delta === 1 ? -1 : results.length);
      for (let n = 0; n < results.length; n += 1) {
        i = (i + delta + results.length) % results.length;
        if (!guessedIds.has(results[i].id)) return i;
      }
      return from;
    },
    [results, guessedIds],
  );

  const onInputKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case "ArrowDown":
        if (!open && results.length > 0) {
          e.preventDefault();
          setOpen(true);
          focusOption(firstSelectable(results, guessedIds));
        } else if (open) {
          e.preventDefault();
          focusOption(step(activeIndex, 1));
        }
        break;
      case "ArrowUp":
        if (open) {
          e.preventDefault();
          focusOption(step(activeIndex, -1));
        }
        break;
      case "Home":
        if (open) {
          e.preventDefault();
          focusOption(firstSelectable(results, guessedIds));
        }
        break;
      case "End":
        if (open) {
          e.preventDefault();
          focusOption(lastSelectable(results, guessedIds));
        }
        break;
      case "Enter": {
        // Confirm the highlight; fall back to the first selectable match so
        // typing a full name + Enter works without touching arrows.
        const target = activeIndex ?? firstSelectable(results, guessedIds);
        if (open && target !== null && !guessedIds.has(results[target].id)) {
          e.preventDefault();
          pick(results[target]);
        }
        break;
      }
      case "Escape":
        if (open) {
          e.preventDefault();
          close();
        }
        break;
    }
  };

  return (
    <div ref={boxRef} className="relative">
      <label htmlFor="fighter-search" className="microlabel mb-2 block text-steel">
        {t("search.label")} <span aria-hidden="true" className="text-blood">— {t("search.ph")}</span>
      </label>
      <div className="relative">
        <span aria-hidden="true" className="pointer-events-none absolute left-0 top-0 flex h-full w-2 bg-blood" />
        <input
          id="fighter-search"
          ref={inputRef}
          type="text"
          autoComplete="off"
          autoFocus={!disabled}
          disabled={disabled}
          value={query}
          role="combobox"
          aria-expanded={open && results.length > 0}
          aria-controls="fighter-search-list"
          aria-activedescendant={activeIndex !== null ? optionId(activeIndex) : undefined}
          aria-autocomplete="list"
          onChange={(e) => setQuery(e.target.value)}
          onFocus={() => results.length > 0 && setOpen(true)}
          onKeyDown={onInputKeyDown}
          placeholder={disabled ? t("search.solvedPh") : t("search.ph")}
          className="w-full border-3 border-ink bg-bone py-3.5 pl-5 pr-16 text-lg font-semibold text-ink shadow-hard placeholder:font-normal placeholder:text-ink/45 focus-visible:shadow-blood focus-visible:outline-blood disabled:opacity-50"
        />
        <kbd
          aria-hidden="true"
          className="pointer-events-none absolute right-3 top-1/2 hidden -translate-y-1/2 border-2 border-ink/20 bg-paper px-1.5 py-0.5 font-mono text-[10px] font-bold text-ash sm:block"
        >
          ↵ ENTER
        </kbd>
        {loading && (
          <p className="absolute left-0 right-0 top-full z-10 mt-2 font-mono text-[11px] tracking-wide text-ash" role="status">
            {t("search.searching")}
          </p>
        )}
        {open && results.length > 0 && (
          <ul
            id="fighter-search-list"
            role="listbox"
            aria-label={t("search.matches")}
            className="absolute left-0 right-0 top-full z-10 mt-2 max-h-80 overflow-auto border-3 border-ink bg-bone shadow-hard"
          >
            {query.trim().length === 1 && !loading && (
              <li aria-hidden="true" className="border-b-2 border-ink px-4 py-2 font-mono text-[11px] tracking-wide text-ash">
                {t("search.keepTyping")}
              </li>
            )}
            {results.map((f, i) => (
              <FighterOption
                key={f.id}
                ref={(el) => {
                  optionRefs.current[i] = el;
                }}
                fighter={f}
                guessed={guessedIds.has(f.id)}
                active={activeIndex === i}
                optionId={optionId(i)}
                onPick={pick}
                onHighlight={() => setActiveIndex(i)}
              />
            ))}
          </ul>
        )}
        {open && query.trim().length >= 1 && !loading && results.length === 0 && (
          <p className="absolute left-0 right-0 top-full z-10 mt-2 border-3 border-ink bg-bone px-3 py-2 text-sm text-ink shadow-hard">
            {t("search.noMatch", { q: query })}
          </p>
        )}
      </div>
    </div>
  );
}

function firstSelectable(rows: SearchResult[], guessedIds: Set<number>): number | null {
  const i = rows.findIndex((f) => !guessedIds.has(f.id));
  return i === -1 ? null : i;
}

function lastSelectable(rows: SearchResult[], guessedIds: Set<number>): number | null {
  for (let i = rows.length - 1; i >= 0; i -= 1) {
    if (!guessedIds.has(rows[i].id)) return i;
  }
  return null;
}
