"use client";

import { useEffect, useState } from "react";
import { Header, Shell } from "@/components/Header";
import { GameBoard } from "@/components/GameBoard";
import { fetchToday } from "@/lib/api";

export default function Page() {
  const [gameDate, setGameDate] = useState("");

  useEffect(() => {
    fetchToday()
      .then((t) => setGameDate(t.date))
      .catch(() => setGameDate(""));
  }, []);

  return (
    <Shell>
      <Header gameDate={gameDate} />
      <main className="mx-auto max-w-5xl space-y-8 px-4 py-8">
        <section aria-label="Daily game">
          <GameBoard />
        </section>
        <footer className="border-t-2 border-bone/20 pt-4 text-xs text-steel">
          Fighter stats are approximate. Photos are by Wikimedia Commons contributors; each photo&apos;s author and licence show in its tooltip and under the winner&apos;s portrait.
        </footer>
      </main>
    </Shell>
  );
}
