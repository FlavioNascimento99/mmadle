"use client";

/**
 * Full-screen splash shown while the app boots (first game load). Purely
 * presentational: no data fetching, so it can render before anything else.
 */
export function Splash() {
  return (
    <div
      className="fixed inset-0 z-50 flex flex-col items-center justify-center gap-6 bg-ink"
      role="status"
      aria-label="Loading MMAdle"
    >
      <h1 className="animate-pulse font-display text-7xl uppercase leading-[0.85] tracking-tight text-bone sm:text-8xl">
        MMA<span className="text-blood">dle</span>
      </h1>
      <div className="flex items-end gap-2" aria-hidden="true">
        {[0, 1, 2].map((i) => (
          <span
            key={i}
            className="w-4 animate-bounce bg-blood"
            style={{ height: `${16 + i * 8}px`, animationDelay: `${i * 150}ms` }}
          />
        ))}
      </div>
      <p className="text-sm text-steel">Warming up the octagon…</p>
    </div>
  );
}
