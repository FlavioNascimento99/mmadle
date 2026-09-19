"use client";

import { useEffect, useState } from "react";
import { AuthDialog } from "@/components/AuthDialog";
import { Header, Shell } from "@/components/Header";
import { GameBoard } from "@/components/GameBoard";
import { Splash } from "@/components/Splash";
import { fetchToday } from "@/lib/api";
import { useAuth } from "@/lib/useAuth";

export default function Page() {
  const [gameDate, setGameDate] = useState("");
  const [booted, setBooted] = useState(false);
  const [authOpen, setAuthOpen] = useState(false);
  const { user, authBusy, authError, clearAuthError, login, register, logout } = useAuth();

  useEffect(() => {
    fetchToday()
      .then((t) => setGameDate(t.date))
      .catch(() => setGameDate(""))
      .finally(() => setBooted(true));
  }, []);

  if (!booted) {
    return (
      <Shell>
        <Splash />
      </Shell>
    );
  }

  return (
    <Shell>
      <Header
        gameDate={gameDate}
        user={user}
        onSignIn={() => {
          clearAuthError();
          setAuthOpen(true);
        }}
        onSignOut={() => void logout()}
      />
      <main className="mx-auto max-w-5xl space-y-8 px-4 py-8">
        <section aria-label="Daily game">
          <GameBoard user={user} />
        </section>
        <footer className="border-t-2 border-bone/20 pt-4 text-xs text-steel">
          Fighter stats are approximate. Photos are by Wikimedia Commons contributors; each photo&apos;s author and licence show in its tooltip and under the winner&apos;s portrait.
        </footer>
      </main>
      <AuthDialog
        open={authOpen}
        busy={authBusy}
        error={authError}
        onLogin={login}
        onRegister={register}
        onClose={() => setAuthOpen(false)}
      />
    </Shell>
  );
}
