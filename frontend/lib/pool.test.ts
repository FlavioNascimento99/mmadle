import { describe, expect, it } from "vitest";
import { parseStoredPool } from "./pool";

describe("parseStoredPool", () => {
  it("restores a saved mode", () => {
    expect(parseStoredPool("men")).toBe("men");
    expect(parseStoredPool("all")).toBe("all");
  });

  it("falls back to all fighters on missing or unknown values", () => {
    expect(parseStoredPool(null)).toBe("all");
    expect(parseStoredPool("women")).toBe("all");
  });
});
