import { describe, expect, it } from "vitest";
import { DICTS, LANGS } from "./i18n";

const varsOf = (s: string): string[] => Array.from(s.matchAll(/\{(\w+)\}/g)).map((m) => m[1]).sort();

describe("i18n dictionaries", () => {
  it("supports exactly en and pt-BR", () => {
    expect(LANGS.map((l) => l.value).sort()).toEqual(["en", "pt-BR"]);
  });

  it("has identical key sets in every language", () => {
    const keys = Object.keys(DICTS.en).sort();
    expect(keys.length).toBeGreaterThan(50);
    for (const lang of LANGS.map((l) => l.value)) {
      expect(Object.keys(DICTS[lang]).sort()).toEqual(keys);
    }
  });

  it("has no empty translations", () => {
    for (const lang of LANGS.map((l) => l.value)) {
      for (const [key, value] of Object.entries(DICTS[lang])) {
        expect(value.trim().length, `${lang}:${key}`).toBeGreaterThan(0);
      }
    }
  });

  it("uses the same interpolation variables across languages", () => {
    for (const key of Object.keys(DICTS.en)) {
      const want = varsOf(DICTS.en[key as keyof typeof DICTS.en]);
      expect(varsOf(DICTS["pt-BR"][key as keyof typeof DICTS.en]), key).toEqual(want);
    }
  });

  it("translates the headline strings (spot check)", () => {
    expect(DICTS["pt-BR"]["header.signIn"]).toBe("Entrar");
    expect(DICTS["pt-BR"]["table.attrNation"]).toBe("País");
    expect(DICTS["pt-BR"]["win.tryMany"]).toContain("{n}");
    expect(DICTS.en["header.signIn"]).toBe("Sign in");
  });
});
