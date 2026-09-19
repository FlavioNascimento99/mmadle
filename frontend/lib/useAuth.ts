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
function friendlyAuthError(e: unknown): string {
  const msg = e instanceof Error ? e.message : "Something went wrong";
  if (/\(401\)/.test(msg)) return "Username or password is incorrect.";
  if (/\(409\)/.test(msg)) return "That username is taken. Try signing in.";
  if (/\(429\)/.test(msg)) return "Too many attempts. Wait a few minutes and try again.";
  if (/weak_password/.test(msg)) return "Password must be at least 10 characters and not a common password.";
  if (/invalid_username/.test(msg)) return "Username must be 3–20 letters, digits or underscores.";
  if (/username_taken/.test(msg)) return "That username is taken. Try signing in.";
  if (/failed/i.test(msg) && /fetch|network|load/i.test(msg)) return "Could not reach the server. Check your connection.";
  return msg.length > 160 ? "Something went wrong. Try again." : msg;
}
