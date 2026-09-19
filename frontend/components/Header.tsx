import type { ReactNode } from "react";
import Link from "next/link";
import type { AuthUser } from "@/lib/api";

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
  return (
    <header className="border-b-4 border-blood">
      <div className="mx-auto flex max-w-5xl items-end justify-between gap-4 px-4 pb-4 pt-6">
        <div>
          <h1 className="font-display text-6xl uppercase leading-[0.85] tracking-tight text-bone sm:text-7xl">
            MMA<span className="text-blood">dle</span>
          </h1>
          <p className="mt-2 text-sm text-steel">
            Guess today&apos;s hidden UFC fighter{gameDate ? `, ${gameDate}` : ""}
          </p>
        </div>
        <div className="flex items-start gap-2">
          {user ? (
            <details className="relative">
              <summary
                className="press cursor-pointer list-none whitespace-nowrap border-3 border-bone bg-blood px-3 py-1.5 font-semibold text-bone shadow-blood"
                aria-label={`Account: ${user.username}`}
              >
                {user.username}
              </summary>
              <div className="absolute right-0 z-20 mt-3 w-64 border-3 border-ink bg-bone p-4 text-sm text-ink shadow-blood-lg">
                <p className="break-all font-bold">@{user.username}</p>
                <p className="mt-1 text-steel">Signed in — guesses sync across devices.</p>
                <Link
                  href="/stats"
                  className="press mt-3 block border-3 border-ink bg-bone px-3 py-1.5 text-center font-semibold text-ink"
                >
                  My stats
                </Link>
                {user.role === "admin" && (
                  <Link
                    href="/admin"
                    className="press mt-3 block border-3 border-ink bg-blood px-3 py-1.5 text-center font-semibold text-bone"
                  >
                    Metrics
                  </Link>
                )}
                <button
                  onClick={onSignOut}
                  className="press mt-3 w-full border-3 border-ink bg-ink px-3 py-1.5 font-semibold text-bone"
                >
                  Sign out
                </button>
              </div>
            </details>
          ) : (
            <button
              onClick={onSignIn}
              className="press whitespace-nowrap border-3 border-bone bg-blood px-3 py-1.5 font-semibold text-bone shadow-blood"
            >
              Sign in
            </button>
          )}
          <details className="relative">
          <summary className="press cursor-pointer list-none whitespace-nowrap border-3 border-bone bg-ink px-3 py-1.5 font-semibold text-bone shadow-blood">
            How to play
          </summary>
          <div className="absolute right-0 z-20 mt-3 w-72 border-3 border-ink bg-bone p-4 text-sm text-ink shadow-blood-lg">
            <p className="mb-3 font-bold">Pick a fighter. Each guess shows how close you are.</p>
            <ul className="space-y-2">
              <Legend symbol="✓" className="bg-blood text-bone">Landed: this attribute matches.</Legend>
              <Legend symbol="↑" className="bg-bone text-ink">The hidden fighter is older or taller.</Legend>
              <Legend symbol="↓" className="bg-bone text-ink">The hidden fighter is younger or shorter.</Legend>
              <Legend symbol="✗" className="bg-ink text-steel">Missed: no match.</Legend>
            </ul>
            <p className="mt-3">
              Stuck? Hints unlock as you keep guessing. Pick <strong>Men only</strong> for a separate
              daily fighter from the men&apos;s divisions.
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
