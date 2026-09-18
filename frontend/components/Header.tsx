import type { ReactNode } from "react";

export function Header({ gameDate }: { gameDate: string }) {
  return (
    <header className="border-b border-zinc-800">
      <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-4">
        <div>
          <h1 className="text-xl font-extrabold tracking-tight text-zinc-50">
            MMAdle <span className="text-red-500">🥊</span>
          </h1>
          <p className="text-xs text-zinc-400">
            The daily MMA fighter guessing game
            {gameDate ? ` · ${gameDate}` : ""}
          </p>
        </div>
        <details className="relative">
          <summary className="cursor-pointer list-none rounded-lg border border-zinc-700 px-3 py-1.5 text-sm text-zinc-200 hover:border-zinc-500">
            How to play
          </summary>
          <div className="absolute right-0 z-20 mt-2 w-72 rounded-xl border border-zinc-700 bg-zinc-900 p-4 text-sm text-zinc-300 shadow-xl">
            <p className="mb-2 font-semibold text-zinc-100">
              Guess the hidden fighter of the day.
            </p>
            <ul className="list-disc space-y-1 pl-4">
              <li>Search and submit a fighter.</li>
              <li>
                <span className="text-emerald-300">✓ correct</span> — attribute
                matches.
              </li>
              <li>
                <span className="text-amber-300">↑ higher</span> — the target
                is older / taller. Guess higher!
              </li>
              <li>
                <span className="text-sky-300">↓ lower</span> — the target is
                younger / shorter. Guess lower!
              </li>
              <li>
                <span className="text-zinc-300">✗ incorrect</span> — no match.
              </li>
            </ul>
          </div>
        </details>
      </div>
    </header>
  );
}

export function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">{children}</div>
  );
}
