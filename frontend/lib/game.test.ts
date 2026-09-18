import { describe, expect, it } from "vitest";
import { comparisonMeta, outcomeToEmoji, storageKey } from "./game";
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
  it("namespaces saved guesses per game date", () => {
    expect(storageKey("2026-09-18")).toBe("mmadle-guesses-2026-09-18");
  });
});
