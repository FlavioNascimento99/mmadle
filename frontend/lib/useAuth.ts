import { useCallback, useEffect, useState } from "react";
import { fetchMe, login as apiLogin, logout as apiLogout, register as apiRegister, type AuthUser } from "./api";

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
      setAuthError(friendlyAuthError(e));
      return false;
    } finally {
      setBusy(false);
    }
  }, []);

  const login = useCallback((email: string, password: string) => run(() => apiLogin(email, password)), [run]);
  const register = useCallback(
    (email: string, password: string, displayName?: string) =>
      run(() => apiRegister(email, password, displayName || undefined)),
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
function friendlyAuthError(e: unknown): string {
  const msg = e instanceof Error ? e.message : "Something went wrong";
  if (/\(401\)/.test(msg)) return "Email or password is incorrect.";
  if (/\(409\)/.test(msg)) return "An account with this email already exists. Try signing in.";
  if (/\(429\)/.test(msg)) return "Too many attempts. Wait a few minutes and try again.";
  if (/weak_password/.test(msg)) return "Password must be at least 10 characters and not a common password.";
  if (/invalid_email/.test(msg)) return "Enter a valid email address.";
  if (/invalid_display_name/.test(msg)) return "Display name must be 1–30 characters.";
  if (/failed/i.test(msg) && /fetch|network|load/i.test(msg)) return "Could not reach the server. Check your connection.";
  return msg.length > 160 ? "Something went wrong. Try again." : msg;
}
