"use client";

import { useEffect, useState } from "react";
import { AuthDialog } from "@/components/AuthDialog";
import { Header, Shell } from "@/components/Header";
import { GameBoard } from "@/components/GameBoard";
import { InfiniteBoard } from "@/components/InfiniteBoard";
import { Splash } from "@/components/Splash";
import { LangProvider, useLang } from "@/lib/i18n";
import { fetchToday } from "@/lib/api";
import { useAuth } from "@/lib/useAuth";

type GameMode = "daily" | "infinite";

const MODE_KEY = "mmadle-mode";

function parseMode(raw: string | null): GameMode {
  return raw === "infinite" ? "infinite" : "daily";
}

export default function Page() {
  return (
    <LangProvider>
      <Home />
    </LangProvider>
  );
}

function Home() {
  const { t } = useLang();
  const [gameDate, setGameDate] = useState("");
  const [booted, setBooted] = useState(false);
  const [authOpen, setAuthOpen] = useState(false);
  const [mode, setModeState] = useState<GameMode>("daily");
  const { user, authBusy, authError, clearAuthError, login, register, logout } = useAuth();

  useEffect(() => {
    try {
      setModeState(parseMode(localStorage.getItem(MODE_KEY)));
    } catch {
      // Storage blocked: daily stays the default.
    }
  }, []);

  const setMode = (next: GameMode) => {
    setModeState(next);
    try {
      localStorage.setItem(MODE_KEY, next);
    } catch {
      // Storage blocked: mode still works in-memory.
    }
  };

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
      <main className="mx-auto w-full max-w-3xl space-y-6 px-4 py-6 sm:py-8">
        <div className="flex items-center justify-between gap-3">
          <div className="flex border-2 border-bone/25" role="group" aria-label={t("mode.group")}>
            {(["daily", "infinite"] as const).map((m) => (
              <button
                key={m}
                onClick={() => setMode(m)}
                aria-pressed={mode === m}
                className={`px-4 py-1.5 font-mono text-xs font-bold uppercase tracking-[0.18em] ${
                  mode === m ? "bg-blood text-bone" : "bg-transparent text-steel hover:text-bone"
                }`}
              >
                {t(m === "daily" ? "mode.daily" : "mode.infinite")}
              </button>
            ))}
          </div>
        </div>
        <section aria-label={mode === "daily" ? t("board.dailyGame") : t("mode.infinite")}>
          {mode === "daily" ? <GameBoard user={user} /> : <InfiniteBoard user={user} />}
        </section>
        <footer className="border-t border-bone/15 pt-4 font-mono text-[11px] leading-relaxed tracking-wide text-ash">
          {t("board.footer")}
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
