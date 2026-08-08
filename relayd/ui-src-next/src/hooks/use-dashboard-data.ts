"use client";

import { useCallback, useEffect, useState } from "react";
import { getDeploys, getProjects, getSession, type Deploy, type Project } from "@/lib/api";
import { hasActiveDeploysIn, normalizeProjects } from "@/lib/relay-utils";

interface DashboardState {
  loading: boolean;
  refreshing: boolean;
  projects: Project[];
  deploys: Deploy[];
  error: string;
  isLive: boolean;
}

export function useDashboardData(enabled: boolean): [DashboardState, () => Promise<void>] {
  const [state, setState] = useState<DashboardState>({
    loading: true,
    refreshing: false,
    projects: [],
    deploys: [],
    error: "",
    isLive: false,
  });

  function applyData(projects: Project[], deploys: Deploy[]) {
    const sorted = (deploys ?? []).slice().sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
    );
    setState({
      loading: false,
      refreshing: false,
      projects: normalizeProjects(projects),
      deploys: sorted,
      error: "",
      isLive: sorted.length > 0 && hasActiveDeploysIn(sorted),
    });
  }

  useEffect(() => {
    if (!enabled) return undefined;
    const es = new EventSource("/api/events", { withCredentials: true });
    let closed = false;
    let checkingAuth = false;

    es.addEventListener("snapshot", (e: MessageEvent) => {
      try {
        const { projects, deploys } = JSON.parse(e.data);
        applyData(projects, deploys);
      } catch { /* ignored */ }
    });

    es.addEventListener("update", (e: MessageEvent) => {
      try {
        const { projects, deploys } = JSON.parse(e.data);
        applyData(projects, deploys);
      } catch { /* ignored */ }
    });

    es.onerror = () => {
      // EventSource can't expose the HTTP status, so a session that expired
      // mid-stream looks identical to a transient network blip — the browser
      // just keeps auto-reconnecting and 401ing forever ("reconnecting…" with
      // no recovery). Probe the session out-of-band: if it's actually gone,
      // stop reconnecting and reload so the app falls back to the login screen.
      if (!checkingAuth && !closed) {
        checkingAuth = true;
        getSession()
          .then((session) => {
            if (closed) return;
            if (!session.authed) {
              closed = true;
              es.close();
              window.location.reload();
            }
          })
          .catch(() => {/* network still down — let EventSource keep retrying */})
          .finally(() => { checkingAuth = false; });
      }
      setState((prev) => ({
        ...prev,
        loading: false,
        isLive: prev.isLive,
        error: prev.loading
          ? "Cannot connect to agent"
          : "Connection lost — reconnecting…",
      }));
    };

    return () => { closed = true; es.close(); };
  }, [enabled]);

  const manualRefresh = useCallback(async () => {
    setState((prev) => ({ ...prev, refreshing: true }));
    try {
      const [projects, deploys] = await Promise.all([getProjects(), getDeploys()]);
      applyData(projects, deploys);
    } catch (err) {
      setState((prev) => ({
        ...prev,
        refreshing: false,
        error: (err as Error).message,
      }));
    }
  }, []);

  return [state, manualRefresh];
}
