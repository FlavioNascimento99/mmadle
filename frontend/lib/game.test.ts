import { describe, expect, it } from "vitest";
import { comparisonMeta, outcomeToEmoji, parseSavedGuesses, shareText, storageKey } from "./game";
import { GuessOutcomeSchema } from "./api";

describe("comparisonMeta", () => {
  it("covers every comparison with symbol + label (never color-only)", () => {
    for (const c of ["correct", "incorrect", "higher", "lower"] as const) {
      const meta = comparisonMeta(c);
      expect(meta.symbol.length).toBeGreaterThan(0);
      expect(meta.label.length).toBeGreaterThan(0);
      expect(meta.classes).toContain("text-");
    }
  });

  it("maps higher/lower to arrows, correct/incorrect to marks", () => {
    expect(comparisonMeta("correct").symbol).toBe("✓");
    expect(comparisonMeta("incorrect").symbol).toBe("✗");
    expect(comparisonMeta("higher").symbol).toBe("↑");
    expect(comparisonMeta("lower").symbol).toBe("↓");
  });
});

describe("GuessOutcomeSchema", () => {
  it("accepts the backend contract and rejects target leaks", () => {
    const ok = {
      fighter_id: 2,
      fighter_name: "Guesser Fighter",
      results: {
        age: { value: 31, comparison: "higher" },
        division: { value: "Lightweight", comparison: "correct" },
        height: { value: 180, comparison: "higher" },
        record: { value: "18-4-0", comparison: "incorrect" },
        nationality: { value: "USA", comparison: "incorrect" },
        last_event: { value: "UFC 319", comparison: "incorrect" },
      },
      correct: false,
    };
    expect(GuessOutcomeSchema.safeParse(ok).success).toBe(true);
    expect(
      GuessOutcomeSchema.safeParse({ ...ok, target: "leak" }).success,
    ).toBe(true); // extra keys stripped, not trusted
    expect(
      GuessOutcomeSchema.safeParse({ ...ok, correct: "yes" }).success,
    ).toBe(false);
  });
});

describe("outcomeToEmoji", () => {
  it("renders six cells in attribute order", () => {
    const outcome = GuessOutcomeSchema.parse({
      fighter_id: 2,
      fighter_name: "G",
      results: {
        age: { value: 31, comparison: "higher" },
        division: { value: "Lightweight", comparison: "correct" },
        height: { value: 180, comparison: "higher" },
        record: { value: "18-4-0", comparison: "incorrect" },
        nationality: { value: "USA", comparison: "incorrect" },
        last_event: { value: "UFC 319", comparison: "incorrect" },
      },
      correct: false,
    });
    expect(outcomeToEmoji(outcome)).toBe("🔼🟩🔼🟥🟥🟥");
  });
});

describe("storageKey", () => {
  it("namespaces saved guesses per mode and game date", () => {
    expect(storageKey("2026-09-18", "all")).toBe("mmadle-guesses-all-2026-09-18");
    expect(storageKey("2026-09-18", "men")).toBe("mmadle-guesses-men-2026-09-18");
  });
});

describe("shareText", () => {
  const guess = GuessOutcomeSchema.parse({
    fighter_id: 2,
    fighter_name: "G",
    results: {
      age: { value: 31, comparison: "correct" },
      division: { value: "Lightweight", comparison: "correct" },
      height: { value: 180, comparison: "correct" },
      record: { value: "18-4-0", comparison: "correct" },
      nationality: { value: "USA", comparison: "correct" },
      last_event: { value: "UFC 319", comparison: "correct" },
    },
    correct: true,
  });

  it("names the mode only for men-only games", () => {
    expect(shareText("2026-09-18", "all", [guess])).toBe("MMAdle 2026-09-18 — 1 try\n🟩🟩🟩🟩🟩🟩");
    expect(shareText("2026-09-18", "men", [guess, guess])).toBe(
      "MMAdle 2026-09-18 (men only) — 2 tries\n🟩🟩🟩🟩🟩🟩\n🟩🟩🟩🟩🟩🟩",
    );
  });
});

const savedOutcome = {
  fighter_id: 2,
  fighter_name: "G",
  photo_url: "https://upload.wikimedia.org/g.jpg",
  photo_credit: "Author / CC BY 3.0",
  results: {
    age: { value: 31, comparison: "higher" },
    division: { value: "Lightweight", comparison: "correct" },
    height: { value: 180, comparison: "higher" },
    record: { value: "18-4-0", comparison: "incorrect" },
    nationality: { value: "USA", comparison: "incorrect" },
    last_event: { value: "UFC 319", comparison: "incorrect" },
  },
  correct: false,
};

describe("GuessOutcomeSchema photo fields", () => {
  it("keeps the guessed fighter's photo and credit", () => {
    const parsed = GuessOutcomeSchema.parse(savedOutcome);
    expect(parsed.photo_url).toBe(savedOutcome.photo_url);
    expect(parsed.photo_credit).toBe(savedOutcome.photo_credit);
  });

  it("accepts a fighter without a photo", () => {
    const parsed = GuessOutcomeSchema.parse({ ...savedOutcome, photo_url: null, photo_credit: null });
    expect(parsed.photo_url).toBeNull();
  });
});

describe("parseSavedGuesses", () => {
  it("restores a valid saved game", () => {
    expect(parseSavedGuesses(JSON.stringify([savedOutcome]))).toEqual([savedOutcome]);
  });

  it("restores saves made before photos existed", () => {
    const { photo_url: _url, photo_credit: _credit, ...legacy } = savedOutcome;
    const [restored] = parseSavedGuesses(JSON.stringify([legacy]));
    expect(restored.fighter_id).toBe(2);
    expect(restored.photo_url ?? null).toBeNull();
  });

  it("starts fresh on missing or corrupt data", () => {
    expect(parseSavedGuesses(null)).toEqual([]);
    expect(parseSavedGuesses("{not json")).toEqual([]);
    expect(parseSavedGuesses(JSON.stringify([{ fighter_id: "x" }]))).toEqual([]);
  });
});
