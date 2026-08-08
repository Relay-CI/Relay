"use client";

import { useEffect, useRef, useState } from "react";
import { getSession, logout, type SessionInfo } from "@/lib/api";

interface AuthState {
  loading: boolean;
  authed: boolean;
  user: { username: string; role: string } | null;
  setupAvailable: boolean;
  legacyMode: boolean;
  cliMode: boolean;
}

const INITIAL: AuthState = {
  loading: false,
  authed: false,
  user: null,
  setupAvailable: false,
  legacyMode: false,
  cliMode: false,
};

export function useAuth() {
  const [state, setState] = useState<AuthState>(INITIAL);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Fetch session without showing the loading spinner (silent background check).
  // Call withLoading=true only on the very first mount check.
  const fetchSession = (withLoading: boolean) => {
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    if (withLoading) {
      setState((prev) => ({ ...prev, loading: true }));
      // Failsafe only for the initial blocking check: never leave the app stuck
      // on the spinner if the very first session fetch hangs. A silent refresh
      // must NOT arm this — a slow background fetch would otherwise flip an
      // already-authenticated operator back to the login screen.
      timeoutRef.current = setTimeout(() => {
        setState({ loading: false, authed: false, user: null, setupAvailable: false, legacyMode: false, cliMode: false });
      }, 5_000);
    }
    getSession()
      .then((session) => {
        if (timeoutRef.current) clearTimeout(timeoutRef.current);
        setState({
          loading: false,
          authed: session.authed,
          user: session.user ?? null,
          setupAvailable: session.setup_available ?? false,
          legacyMode: session.legacy_mode ?? false,
          cliMode: session.cli_mode ?? false,
        });
      })
      .catch(() => {
        if (timeoutRef.current) clearTimeout(timeoutRef.current);
        setState({ loading: false, authed: false, user: null, setupAvailable: false, legacyMode: false, cliMode: false });
      });
  };

  // After login/setup success, refresh silently — no spinner.
  const refresh = () => fetchSession(false);

  useEffect(() => {
    fetchSession(true);
    return () => { if (timeoutRef.current) clearTimeout(timeoutRef.current); };
  }, []);

  const signOut = async () => {
    try {
      await logout();
    } catch {
      // clear local state regardless of whether the server call succeeded
    }
    setState({ loading: false, authed: false, user: null, setupAvailable: false, legacyMode: false, cliMode: false });
  };

  return { ...state, refresh, signOut };
}
