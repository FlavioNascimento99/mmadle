import { describe, expect, it } from "vitest";
import { initials } from "./fighter";

describe("initials", () => {
  it("uses the first and last name", () => {
    expect(initials("Alex Pereira")).toBe("AP");
    expect(initials("Ian Machado Garry")).toBe("IG");
  });

  it("handles single names and stray whitespace", () => {
    expect(initials("  Poatan ")).toBe("P");
    expect(initials("")).toBe("?");
  });

  it("keeps accented letters", () => {
    expect(initials("Jiří Procházka")).toBe("JP");
  });
});
