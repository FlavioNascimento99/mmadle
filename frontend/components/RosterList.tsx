"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { listFighters, type Pool, type SearchResult } from "@/lib/api";
import { findTypeaheadIndex, groupRoster } from "@/lib/fighter";
import { useClickOutside } from "@/lib/useClickOutside";
import { useLang } from "@/lib/i18n";
import { FighterOption } from "./FighterOption";

type Props = {
  pool: Pool;
  disabled: boolean;
  guessedIds: Set<number>;
  onSelect: (fighter: SearchResult) => void;
};

const optionId = (index: number) => `roster-option-${index}`;

export function RosterList({ pool, disabled, guessedIds, onSelect }: Props) {
  const { t } = useLang();
  const [open, setOpen] = useState(false);
  const [roster, setRoster] = useState<SearchResult[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  /** Highlighted (keyboard/mouse) option; Enter confirms it. */
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const boxRef = useRef<HTMLDivElement>(null);
  const toggleRef = useRef<HTMLButtonElement>(null);
  const optionRefs = useRef<(HTMLButtonElement | null)[]>([]);

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
    } catch {
      setError(t("roster.loadError"));
    }
  };

  const pick = (f: SearchResult) => {
    if (guessedIds.has(f.id)) return;
    onSelect(f);
    setOpen(false);
  };

  const groups = useMemo(() => (roster ? groupRoster(roster) : []), [roster]);

  /** Display order (grouped); keyboard navigation indexes into this array. */
  const ordered = useMemo(() => groups.flatMap((g) => g.fighters), [groups]);

  const focusIndex = useCallback(
    (index: number | null) => {
      if (index === null || ordered.length === 0) return;
      setActiveIndex(index);
      // Wait a tick so the button exists when navigating right after open.
      requestAnimationFrame(() => optionRefs.current[index]?.focus());
    },
    [ordered.length],
  );

  const step = useCallback(
    (from: number | null, delta: 1 | -1): number | null => {
      if (ordered.length === 0) return null;
      let i = from ?? (delta === 1 ? -1 : ordered.length);
      for (let n = 0; n < ordered.length; n += 1) {
        i = (i + delta + ordered.length) % ordered.length;
        if (!guessedIds.has(ordered[i].id)) return i;
      }
      return from;
    },
    [ordered, guessedIds],
  );

  const jumpToEdge = useCallback(
    (edge: "first" | "last"): number | null => {
      if (ordered.length === 0) return null;
      if (edge === "first") {
        const idx = ordered.findIndex((f) => !guessedIds.has(f.id));
        return idx === -1 ? null : idx;
      }
      for (let i = ordered.length - 1; i >= 0; i -= 1) {
        if (!guessedIds.has(ordered[i].id)) return i;
      }
      return null;
    },
    [ordered, guessedIds],
  );

  // When the list opens (or finishes loading), highlight the first
  // selectable fighter and move focus into the list for keyboard users.
  useEffect(() => {
    if (!open || ordered.length === 0) return;
    focusIndex(jumpToEdge("first"));
  }, [open, ordered, focusIndex, jumpToEdge]);

  // Return focus to the toggle when the list closes via keyboard.
  const closeAndRefocus = useCallback(() => {
    setOpen(false);
    requestAnimationFrame(() => toggleRef.current?.focus());
  }, []);

  const onListKeyDown = (e: React.KeyboardEvent) => {
    if (ordered.length === 0) return;
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        focusIndex(step(activeIndex, 1));
        break;
      case "ArrowUp":
        e.preventDefault();
        focusIndex(step(activeIndex, -1));
        break;
      case "Home":
        e.preventDefault();
        focusIndex(jumpToEdge("first"));
        break;
      case "End":
        e.preventDefault();
        focusIndex(jumpToEdge("last"));
        break;
      case "Enter":
        e.preventDefault();
        if (activeIndex !== null && !guessedIds.has(ordered[activeIndex].id)) {
          pick(ordered[activeIndex]);
        }
        break;
      case "Escape":
        e.preventDefault();
        closeAndRefocus();
        break;
      default:
        // Single-letter type-ahead: jump to the next fighter whose first or
        // last name starts with the typed character (cycles with repeats).
        if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
          const next = findTypeaheadIndex(ordered, activeIndex ?? -1, e.key);
          if (next !== -1) {
            e.preventDefault();
            const target = guessedIds.has(ordered[next].id)
              ? step(next, 1) ?? next
              : next;
            focusIndex(target);
          }
        }
    }
  };

  const renderGroups = () => {
    let flatIndex = -1;
    return groups.map((g) => (
      <div key={g.letter}>
        <p
          aria-hidden="true"
          className="sticky top-0 border-y-2 border-ink bg-ink px-3 py-1 text-xs font-bold uppercase tracking-widest text-bone"
        >
          {g.letter}
        </p>
        <ul role="group" aria-label={t("roster.group", { letter: g.letter })}>
          {g.fighters.map((f) => {
            flatIndex += 1;
            const current = flatIndex;
            return (
              <FighterOption
                key={f.id}
                ref={(el) => {
                  optionRefs.current[current] = el;
                }}
                fighter={f}
                guessed={guessedIds.has(f.id)}
                active={activeIndex === current}
                optionId={optionId(current)}
                onPick={pick}
                onHighlight={() => setActiveIndex(current)}
              />
            );
          })}
        </ul>
      </div>
    ));
  };

  return (
    <div ref={boxRef} className="relative shrink-0">
      <button
        type="button"
        ref={toggleRef}
        disabled={disabled}
        onClick={toggle}
        onKeyDown={(e) => {
          if ((e.key === "ArrowDown" || e.key === "Enter") && !open) {
            e.preventDefault();
            toggle();
          }
        }}
        aria-expanded={open}
        aria-controls="roster-list"
        aria-haspopup="listbox"
        className={`press whitespace-nowrap border-3 border-ink px-4 py-3.5 font-mono text-xs font-bold uppercase tracking-[0.14em] disabled:opacity-50 ${
          open ? "bg-ink text-bone shadow-hard" : "bg-bone text-ink shadow-hard"
        }`}
      >
        {t("roster.all")} <span aria-hidden="true">{open ? "▴" : "▾"}</span>
      </button>
      {open && (
        <div
          id="roster-list"
          className="absolute right-0 z-10 mt-2 w-[min(28rem,calc(100vw-2rem))] overflow-hidden border-3 border-ink bg-bone text-ink shadow-hard"
        >
          {error ? (
            <p className="px-4 py-3 text-sm font-semibold text-blood" role="alert">
              {error}
            </p>
          ) : roster === null ? (
            <p className="px-4 py-3 text-sm" role="status">
              {t("roster.loading")}
            </p>
          ) : (
            <>
              <p className="border-b-3 border-ink bg-ink px-4 py-2 text-sm font-semibold text-bone">
                {t("roster.count", { n: roster.length })}
                <span className="ml-2 font-normal text-bone/70">{t("roster.keyboardHint")}</span>
              </p>
              <div
                role="listbox"
                aria-label={t("roster.list")}
                aria-activedescendant={activeIndex !== null ? optionId(activeIndex) : undefined}
                tabIndex={-1}
                onKeyDown={onListKeyDown}
                className="max-h-96 overflow-auto"
              >
                {renderGroups()}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}
