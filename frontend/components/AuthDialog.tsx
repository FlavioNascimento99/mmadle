"use client";

import { useEffect, useRef, useState } from "react";
import { useClickOutside } from "@/lib/useClickOutside";
import { useLang } from "@/lib/i18n";

type Props = {
  open: boolean;
  busy: boolean;
  error: string | null;
  onLogin: (username: string, password: string) => Promise<boolean>;
  onRegister: (username: string, password: string) => Promise<boolean>;
  onClose: () => void;
};

/**
 * Sign-in / register dialog in the neo-brutalist style. Guest mode stays the
 * default: this dialog only ever opens from the header's Sign in button.
 */
export function AuthDialog({ open, busy, error, onLogin, onRegister, onClose }: Props) {
  const { t } = useLang();
  const [tab, setTab] = useState<"login" | "register">("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const ref = useRef<HTMLDivElement>(null);
  useClickOutside(ref, () => {
    if (open) onClose();
  });

  useEffect(() => {
    if (open) {
      setPassword("");
    }
  }, [open, tab]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const ok =
      tab === "login"
        ? await onLogin(username.trim(), password)
        : await onRegister(username.trim(), password);
    if (ok) onClose();
  };

  return (
    <div className="fixed inset-0 z-30 flex items-center justify-center bg-ink/80 p-4" role="dialog" aria-modal="true" aria-label={t(tab === "login" ? "auth.signIn" : "auth.create")}>
      <div ref={ref} className="w-full max-w-sm border-3 border-bone bg-ink p-5 shadow-blood-lg">
        <div className="mb-4 flex border-3 border-bone" role="tablist" aria-label={t("auth.tabs")}>
          {(["login", "register"] as const).map((mode) => (
            <button
              key={mode}
              role="tab"
              aria-selected={tab === mode}
              onClick={() => setTab(mode)}
              className={`flex-1 px-3 py-2 font-display text-lg uppercase tracking-wide ${
                tab === mode ? "bg-blood text-bone" : "bg-ink text-steel"
              }`}
            >
              {mode === "login" ? t("auth.signIn") : t("auth.join")}
            </button>
          ))}
        </div>

        <form onSubmit={submit} className="space-y-3">
          <label className="block text-sm font-semibold text-bone">
            {t("auth.username")}
            <input
              type="text"
              required
              minLength={3}
              maxLength={20}
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="mt-1 w-full border-3 border-bone bg-ink px-3 py-2 text-bone placeholder:text-steel/60"
              placeholder="octagon_fan"
            />
          </label>
          <label className="block text-sm font-semibold text-bone">
            {t("auth.password")}
            <input
              type="password"
              required
              minLength={10}
              autoComplete={tab === "login" ? "current-password" : "new-password"}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 w-full border-3 border-bone bg-ink px-3 py-2 text-bone placeholder:text-steel/60"
              placeholder={t("auth.passwordPh")}
            />
          </label>
          {error && (
            <p className="border-3 border-blood bg-bruise px-3 py-2 text-sm font-semibold text-bone" role="alert">
              {error}
            </p>
          )}
          <div className="flex gap-2 pt-1">
            <button
              type="submit"
              disabled={busy}
              className="press flex-1 border-3 border-bone bg-blood px-3 py-2 font-display text-lg uppercase text-bone shadow-blood disabled:opacity-60"
            >
              {busy ? t("auth.working") : tab === "login" ? t("auth.signIn") : t("auth.create")}
            </button>
            <button
              type="button"
              onClick={onClose}
              className="press border-3 border-bone bg-ink px-3 py-2 font-semibold text-steel"
            >
              {t("auth.later")}
            </button>
          </div>
        </form>
        <p className="mt-3 text-xs text-steel">
          {t("auth.note")}
        </p>
      </div>
    </div>
  );
}
