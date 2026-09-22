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
    <div className="flex border-2 border-bone/25" role="group" aria-label={t("lang.group")}>
      {LANGS.map((l) => (
        <button
          key={l.value}
          onClick={() => setLang(l.value)}
          aria-pressed={lang === l.value}
          title={l.label}
          className={`px-2 py-1.5 font-mono text-[11px] font-bold tracking-widest ${
            lang === l.value ? "bg-bone text-ink" : "bg-transparent text-steel hover:text-bone"
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
    <header className="sticky top-0 z-30 border-b-2 border-bone/15 bg-ink/95 backdrop-blur">
      <div className="mx-auto flex max-w-3xl flex-wrap items-center justify-between gap-x-3 gap-y-2 px-4 py-3">
        <div className="flex min-w-0 items-center gap-3">
          <Link href="/" className="flex shrink-0 items-stretch border-2 border-bone" aria-label="MMAdle">
            <span className="bg-bone px-2 py-1 font-display text-2xl uppercase leading-none text-ink">
              MMA
            </span>
            <span className="bg-blood px-2 py-1 font-display text-2xl uppercase leading-none text-bone">
              DLE
            </span>
          </Link>
          <div className="hidden min-w-0 sm:block">
            <p className="microlabel text-blood">
              {t("board.dailyGame")}
              {gameDate ? <span className="text-steel"> · {gameDate}</span> : null}
            </p>
            <p className="truncate text-xs text-steel">{t("header.tagline")}</p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {user ? (
            <details className="relative">
              <summary
                className="press cursor-pointer list-none whitespace-nowrap border-2 border-bone bg-bone px-3 py-1.5 font-mono text-xs font-bold text-ink"
                aria-label={t("header.account", { name: user.username })}
              >
                @{user.username}
              </summary>
              <div className="absolute right-0 z-20 mt-3 w-64 border-3 border-ink bg-bone p-4 text-sm text-ink shadow-hard">
                <p className="break-all font-bold">@{user.username}</p>
                <p className="mt-1 text-xs text-ash">{t("header.signedIn")}</p>
                <Link
                  href="/stats"
                  className="press mt-3 block border-2 border-ink bg-bone px-3 py-1.5 text-center font-semibold text-ink"
                >
                  {t("header.myStats")}
                </Link>
                {user.role === "admin" && (
                  <Link
                    href="/admin"
                    className="press mt-2 block border-2 border-ink bg-ink px-3 py-1.5 text-center font-semibold text-bone"
                  >
                    {t("header.metrics")}
                  </Link>
                )}
                <button
                  onClick={onSignOut}
                  className="press mt-2 w-full border-2 border-ink bg-ink px-3 py-1.5 font-semibold text-bone"
                >
                  {t("header.signOut")}
                </button>
              </div>
            </details>
          ) : (
            <button
              onClick={onSignIn}
              className="press whitespace-nowrap border-2 border-blood bg-blood px-3 py-1.5 font-mono text-xs font-bold uppercase tracking-widest text-bone"
            >
              {t("header.signIn")}
            </button>
          )}
          <LangToggle />
          <details className="relative">
            <summary className="press cursor-pointer list-none whitespace-nowrap border-2 border-bone/25 bg-transparent px-3 py-1.5 font-mono text-xs font-bold uppercase tracking-widest text-bone">
              {t("header.howTo")}
            </summary>
            <div className="absolute right-0 z-20 mt-3 w-72 border-3 border-ink bg-bone p-4 text-sm text-ink shadow-hard">
              <p className="mb-3 font-bold">{t("header.howIntro")}</p>
              <ul className="space-y-2">
                <Legend symbol="✓" className="bg-blood text-bone">{t("header.legLanded")}</Legend>
                <Legend symbol="↑" className="bg-bone text-ink">{t("header.legHigher")}</Legend>
                <Legend symbol="↓" className="bg-bone text-ink">{t("header.legLower")}</Legend>
                <Legend symbol="✗" className="bg-ink text-steel">{t("header.legMiss")}</Legend>
              </ul>
              <p className="mt-3 text-xs text-ash">
                {t("header.howHints")}{" "}
                {t("header.howPools", { pool: t("pool.men") })}
              </p>
            </div>
          </details>
        </div>
      </div>
      <div aria-hidden="true" className="h-[3px] bg-blood" />
    </header>
  );
}

export function Shell({ children }: { children: ReactNode }) {
  return <div className="min-h-screen bg-ink text-bone">{children}</div>;
}
