import { describe, expect, it } from "vitest";
import {
  findTypeaheadIndex,
  firstNameOf,
  groupRoster,
  initials,
  lastNameOf,
  rosterInitial,
} from "./fighter";

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

describe("roster grouping", () => {
  it("uses the first character of the first name", () => {
    expect(rosterInitial("Alex Pereira")).toBe("A");
    expect(rosterInitial("  ilia topuria")).toBe("I");
    expect(rosterInitial("Jiří Procházka")).toBe("J");
    expect(rosterInitial("")).toBe("#");
  });

  it("splits first and last names", () => {
    expect(firstNameOf("Ian Machado Garry")).toBe("Ian");
    expect(lastNameOf("Ian Machado Garry")).toBe("Garry");
    expect(lastNameOf("Poatan")).toBe("");
  });

  it("groups a roster into alphabetical sections", () => {
    const groups = groupRoster([
      { name: "Alex Pereira" },
      { name: "Charles Oliveira" },
      { name: "Amanda Nunes" },
    ]);
    expect(groups.map((g) => g.letter)).toEqual(["A", "C"]);
    expect(groups[0].fighters.map((f) => f.name)).toEqual([
      "Alex Pereira",
      "Amanda Nunes",
    ]);
  });
});

describe("roster type-ahead", () => {
  const roster = [
    { name: "Alex Pereira" },
    { name: "Charles Oliveira" },
    { name: "Ilia Topuria" },
  ];

  it("matches the first character of the first or last name", () => {
    // "p" matches Pereira (last name of index 0), wrapping from index 1.
    expect(findTypeaheadIndex(roster, 1, "p")).toBe(0);
    // "t" matches Topuria (last name of index 2).
    expect(findTypeaheadIndex(roster, 0, "t")).toBe(2);
    // Case-insensitive first-name match.
    expect(findTypeaheadIndex(roster, 2, "C")).toBe(1);
  });

  it("returns -1 when nothing matches", () => {
    expect(findTypeaheadIndex(roster, 0, "z")).toBe(-1);
    expect(findTypeaheadIndex([], 0, "a")).toBe(-1);
  });
});
