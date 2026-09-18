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
      <main className="mx-auto max-w-5xl space-y-6 px-4 py-6">
        <section aria-label="Daily game">
          <GameBoard />
        </section>
        <footer className="pt-4 text-center text-xs text-zinc-500">
          MMAdle MVP · fighter data is demo seed data · new fighter every day
        </footer>
      </main>
    </Shell>
  );
}
