import { useCallback, useEffect, useState } from "react";
import { fetchMe, login as apiLogin, logout as apiLogout, register as apiRegister, type AuthUser } from "./api";
import { DICTS, useLang, type Lang } from "./i18n";

export type AuthState =
  | { status: "loading"; user: null }
  | { status: "guest"; user: null }
  | { status: "ready"; user: AuthUser };

/**
 * Session state for player accounts. Guests are the default: the session is
 * only checked, never required. Components refresh server data when the user
 * changes (sign in/out).
 */
export function useAuth() {
  const { lang } = useLang();
  const [state, setState] = useState<AuthState>({ status: "loading", user: null });
  const [authError, setAuthError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    fetchMe()
      .then((user) => setState(user ? { status: "ready", user } : { status: "guest", user: null }))
      .catch(() => setState({ status: "guest", user: null }));
  }, []);

  const run = useCallback(async (fn: () => Promise<AuthUser | void>): Promise<boolean> => {
    setBusy(true);
    setAuthError(null);
    try {
      const result = await fn();
      if (result) setState({ status: "ready", user: result });
      else setState({ status: "guest", user: null });
      return true;
    } catch (e) {
      setAuthError(friendlyAuthError(e, lang));
      return false;
    } finally {
      setBusy(false);
    }
  }, [lang]);

  const login = useCallback((username: string, password: string) => run(() => apiLogin(username, password)), [run]);
  const register = useCallback(
    (username: string, password: string) => run(() => apiRegister(username, password)),
    [run],
  );
  const logout = useCallback(() => run(async () => { await apiLogout(); }), [run]);

  return {
    user: state.user,
    authLoading: state.status === "loading",
    authError,
    authBusy: busy,
    clearAuthError: () => setAuthError(null),
    login,
    register,
    logout,
  };
}

/** Backend messages are terse codes; map the common ones to UI copy. */
function friendlyAuthError(e: unknown, lang: Lang): string {
  // t() without a provider: tiny local lookup so this stays callable in tests.
  const t = (key: "auth.unauthorized" | "auth.taken" | "auth.rateLimited" | "auth.weakPassword" | "auth.badUsername" | "auth.offline" | "auth.failed"): string =>
    DICTS[lang][key];
  const msg = e instanceof Error ? e.message : t("auth.failed");
  if (/\(401\)/.test(msg)) return t("auth.unauthorized");
  if (/\(409\)/.test(msg) || /username_taken/.test(msg)) return t("auth.taken");
  if (/\(429\)/.test(msg)) return t("auth.rateLimited");
  if (/weak_password/.test(msg)) return t("auth.weakPassword");
  if (/invalid_username/.test(msg)) return t("auth.badUsername");
  if (/failed/i.test(msg) && /fetch|network|load/i.test(msg)) return t("auth.offline");
  return msg.length > 160 ? t("auth.failed") : msg;
}
