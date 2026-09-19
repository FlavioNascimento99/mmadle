"use client";

import { useEffect, useState } from "react";
import { fetchHints, type Hints, type Pool } from "@/lib/api";
import { useLang } from "@/lib/i18n";

type Props = {
  pool: Pool;
  guessCount: number;
};

export function HintPanel({ pool, guessCount }: Props) {
  const { t } = useLang();
  const [hints, setHints] = useState<Hints | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let stale = false;
    setFailed(false);
    fetchHints(guessCount, pool)
      .then((result) => !stale && setHints(result))
      .catch(() => !stale && setFailed(true));
    return () => {
      stale = true;
    };
  }, [pool, guessCount]);

  if (failed) {
    return (
      <p className="text-sm text-steel" role="alert">
        {t("hints.unavailable")}
      </p>
    );
  }
  if (!hints) return null;

  const remaining = hints.next_at === null ? 0 : hints.next_at - guessCount;

  return (
    <section aria-label={t("hints.title")} aria-live="polite" className="space-y-3">
      {hints.hints.length > 0 && (
        <ul className="flex flex-wrap gap-3">
          {hints.hints.map((hint) => (
            <li key={hint.kind} className="border-3 border-ink bg-bone px-3 py-2 text-ink shadow-blood">
              <span className="block text-xs font-bold uppercase tracking-wide text-bruise">{hint.label}</span>
              <span className="block font-display text-xl uppercase leading-tight">{hint.value}</span>
            </li>
          ))}
        </ul>
      )}
      {remaining > 0 && (
        <p className="text-sm text-steel">
          {t("hints.next", { n: remaining, unit: t(remaining === 1 ? "hints.guessOne" : "hints.guessMany") })}
        </p>
      )}
    </section>
  );
}
