import type { ReactNode } from "react";
import Link from "next/link";
import type { AuthUser } from "@/lib/api";
import { LANGS, useLang } from "@/lib/i18n";

function Legend({ symbol, className, children }: { symbol: string; className: string; children: ReactNode }) {
  return (
    <li className="flex items-start gap-2">
      <span className={`inline-flex h-6 w-6 shrink-0 items-center justify-center border-2 border-ink font-bold ${className}`}>
        {symbol}
      </span>
      <span>{children}</span>
    </li>
  );
}

function LangToggle() {
  const { lang, setLang, t } = useLang();
  return (
    <div className="flex border-3 border-bone" role="group" aria-label={t("lang.group")}>
      {LANGS.map((l) => (
        <button
          key={l.value}
          onClick={() => setLang(l.value)}
          aria-pressed={lang === l.value}
          title={l.label}
          className={`px-2 py-1.5 text-xs font-bold tracking-wide ${
            lang === l.value ? "bg-bone text-ink" : "bg-ink text-steel"
          }`}
        >
          {l.short}
        </button>
      ))}
    </div>
  );
}

export function Header({
  gameDate,
  user,
  onSignIn,
  onSignOut,
}: {
  gameDate: string;
  user: AuthUser | null;
  onSignIn: () => void;
  onSignOut: () => void;
}) {
  const { t } = useLang();
  return (
    <header className="border-b-4 border-blood">
      <div className="mx-auto flex max-w-5xl items-end justify-between gap-4 px-4 pb-4 pt-6">
        <div>
          <h1 className="font-display text-6xl uppercase leading-[0.85] tracking-tight text-bone sm:text-7xl">
            MMA<span className="text-blood">dle</span>
          </h1>
          <p className="mt-2 text-sm text-steel">
            {gameDate ? t("header.taglineDate", { date: gameDate }) : t("header.tagline")}
          </p>
        </div>
        <div className="flex items-start gap-2">
          {user ? (
            <details className="relative">
              <summary
                className="press cursor-pointer list-none whitespace-nowrap border-3 border-bone bg-blood px-3 py-1.5 font-semibold text-bone shadow-blood"
                aria-label={t("header.account", { name: user.username })}
              >
                {user.username}
              </summary>
              <div className="absolute right-0 z-20 mt-3 w-64 border-3 border-ink bg-bone p-4 text-sm text-ink shadow-blood-lg">
                <p className="break-all font-bold">@{user.username}</p>
                <p className="mt-1 text-steel">{t("header.signedIn")}</p>
                <Link
                  href="/stats"
                  className="press mt-3 block border-3 border-ink bg-bone px-3 py-1.5 text-center font-semibold text-ink"
                >
                  {t("header.myStats")}
                </Link>
                {user.role === "admin" && (
                  <Link
                    href="/admin"
                    className="press mt-3 block border-3 border-ink bg-blood px-3 py-1.5 text-center font-semibold text-bone"
                  >
                    {t("header.metrics")}
                  </Link>
                )}
                <button
                  onClick={onSignOut}
                  className="press mt-3 w-full border-3 border-ink bg-ink px-3 py-1.5 font-semibold text-bone"
                >
                  {t("header.signOut")}
                </button>
              </div>
            </details>
          ) : (
            <button
              onClick={onSignIn}
              className="press whitespace-nowrap border-3 border-bone bg-blood px-3 py-1.5 font-semibold text-bone shadow-blood"
            >
              {t("header.signIn")}
            </button>
          )}
          <LangToggle />
          <details className="relative">
          <summary className="press cursor-pointer list-none whitespace-nowrap border-3 border-bone bg-ink px-3 py-1.5 font-semibold text-bone shadow-blood">
            {t("header.howTo")}
          </summary>
          <div className="absolute right-0 z-20 mt-3 w-72 border-3 border-ink bg-bone p-4 text-sm text-ink shadow-blood-lg">
            <p className="mb-3 font-bold">{t("header.howIntro")}</p>
            <ul className="space-y-2">
              <Legend symbol="✓" className="bg-blood text-bone">{t("header.legLanded")}</Legend>
              <Legend symbol="↑" className="bg-bone text-ink">{t("header.legHigher")}</Legend>
              <Legend symbol="↓" className="bg-bone text-ink">{t("header.legLower")}</Legend>
              <Legend symbol="✗" className="bg-ink text-steel">{t("header.legMiss")}</Legend>
            </ul>
            <p className="mt-3">
              {t("header.howHints")}{" "}
              {t("header.howPools", { pool: t("pool.men") })}
            </p>
          </div>
          </details>
        </div>
      </div>
    </header>
  );
}

export function Shell({ children }: { children: ReactNode }) {
  return <div className="min-h-screen bg-ink text-bone">{children}</div>;
}
