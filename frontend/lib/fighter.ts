/** Fallback mark for fighters without a free-licensed photo. */
export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  const first = parts[0][0];
  const last = parts.length > 1 ? parts[parts.length - 1][0] : "";
  return (first + last).toUpperCase();
}

/** First token of a display name ("Alex" from "Alex Pereira"). */
export function firstNameOf(name: string): string {
  return name.trim().split(/\s+/).filter(Boolean)[0] ?? "";
}

/** Last token of a display name ("Pereira" from "Alex Pereira", "" when single). */
export function lastNameOf(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  return parts.length > 1 ? parts[parts.length - 1] : "";
}

/**
 * Section key for the roster list: uppercased first character of the first
 * name (A–Z, accented letters kept as-is, anything else falls into "#").
 */
export function rosterInitial(name: string): string {
  const first = firstNameOf(name)[0];
  if (!first) return "#";
  const upper = first.toUpperCase();
  // Letters change between cases; digits/symbols do not.
  return upper.toLowerCase() !== upper.toUpperCase() ? upper : "#";
}

export type RosterGroup<T extends { name: string }> = {
  letter: string;
  fighters: T[];
};

/** Group a (pre-sorted) roster into alphabetical sections by first-name initial. */
export function groupRoster<T extends { name: string }>(fighters: T[]): RosterGroup<T>[] {
  const buckets: Record<string, T[]> = {};
  const order: string[] = [];
  for (const f of fighters) {
    const letter = rosterInitial(f.name);
    if (!buckets[letter]) {
      buckets[letter] = [];
      order.push(letter);
    }
    buckets[letter].push(f);
  }
  order.sort((a, b) => a.localeCompare(b));
  return order.map((letter) => ({ letter, fighters: buckets[letter] }));
}

/**
 * Type-ahead jump: next index (wrapping, after `fromIndex`) whose first or
 * last name starts with `char` (case-insensitive). Returns -1 when nothing
 * matches. Powers the "press a letter to jump" roster UX.
 */
export function findTypeaheadIndex<T extends { name: string }>(
  fighters: T[],
  fromIndex: number,
  char: string,
): number {
  const target = char.trim()[0]?.toLowerCase();
  if (!target || fighters.length === 0) return -1;
  for (let step = 1; step <= fighters.length; step += 1) {
    const i = (fromIndex + step + fighters.length) % fighters.length;
    const first = firstNameOf(fighters[i].name)[0]?.toLowerCase();
    const last = lastNameOf(fighters[i].name)[0]?.toLowerCase();
    if (first === target || last === target) return i;
  }
  return -1;
}
