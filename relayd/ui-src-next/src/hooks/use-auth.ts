"use client";

import { useEffect, useState } from "react";
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
  loading: true,
  authed: false,
  user: null,
  setupAvailable: false,
  legacyMode: false,
  cliMode: false,
};

export function useAuth() {
  const [state, setState] = useState<AuthState>(INITIAL);

  const refresh = () => {
    setState((prev) => ({ ...prev, loading: true }));
    getSession()
      .then((session) => {
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
        setState({ loading: false, authed: false, user: null, setupAvailable: false, legacyMode: false, cliMode: false });
      });
  };

  useEffect(() => { refresh(); }, []);

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
