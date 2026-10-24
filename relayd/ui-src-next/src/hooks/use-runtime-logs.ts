"use client";

import { useEffect, useState } from "react";
import { getRuntimeTargets, type EnvInfo, type RuntimeTarget } from "@/lib/api";
import {
  EMPTY_RUNTIME_LANE_STATE,
  humanizeRuntimeStreamError,
  isRuntimeOfflineError,
  normalizeRuntimeLaneState,
  parseRuntimeLogEntry,
  sinceISO,
  type RuntimeLaneState,
  type RuntimeLogEntry,
} from "@/lib/relay-utils";

interface RuntimeLogsState {
  targetScope: string;
  loading: boolean;
  targets: RuntimeTarget[];
  defaultTarget: string;
  lines: RuntimeLogEntry[];
  status: string;
  targetMeta: RuntimeTarget | null;
  lane: RuntimeLaneState;
  error: string;
}

type SelectedEnv = Pick<EnvInfo, "app" | "env" | "branch" | "stopped" | "engine" | "active_slot" | "standby_slot" | "replicas" | "running_replicas" | "latestDeploy">;

export function useRuntimeLogs(
  selectedEnv: SelectedEnv | null | undefined,
  selectedTarget: string,
  windowFilter: string,
): RuntimeLogsState {
  const [state, setState] = useState<RuntimeLogsState>({
    targetScope: "",
    loading: false,
    targets: [],
    defaultTarget: "",
    lines: [],
    status: "idle",
    targetMeta: null,
    lane: EMPTY_RUNTIME_LANE_STATE,
    error: "",
  });

  const app = selectedEnv?.app;
  const env = selectedEnv?.env;
  const branch = selectedEnv?.branch;
  const stopped = selectedEnv?.stopped;
  // Refresh targets when runtime identity changes, while ignoring identical snapshots.
  const targetScope = JSON.stringify([
    app, env, branch, stopped, selectedEnv?.engine,
    selectedEnv?.active_slot, selectedEnv?.standby_slot,
    selectedEnv?.replicas, selectedEnv?.running_replicas,
    selectedEnv?.latestDeploy?.id, selectedEnv?.latestDeploy?.status,
  ]);

  // Fetch available targets when the lane changes, not on every SSE snapshot.
  useEffect(() => {
    if (!app || !env || !branch) {
      setState({
        targetScope: "",
        loading: false, targets: [], defaultTarget: "", lines: [],
        status: "idle", targetMeta: null, lane: EMPTY_RUNTIME_LANE_STATE, error: "",
      });
      return undefined;
    }

    let cancelled = false;
    setState((prev) => ({ ...prev, targetScope: "", loading: true, error: "" }));

    getRuntimeTargets(app, env, branch)
      .then((data) => {
        if (cancelled) return;
        const lane = normalizeRuntimeLaneState(data?.lane, { app, env, branch, stopped });
        const defaultTarget = data?.default_target ?? "";
        setState((prev) => ({
          ...prev,
          targetScope,
          loading: false,
          targets: data?.targets ?? [],
          defaultTarget,
          lines: [],
          status: defaultTarget ? "idle" : lane.hasRunningTarget ? "idle" : "offline",
          targetMeta: null,
          lane,
          error: "",
        }));
      })
      .catch((err: Error) => {
        if (cancelled) return;
        setState((prev) => ({
          ...prev,
          targetScope,
          loading: false,
          targets: [],
          defaultTarget: "",
          lines: [],
          status: "error",
          targetMeta: null,
          lane: normalizeRuntimeLaneState(null, { app, env, branch, stopped }),
          error: err.message,
        }));
      });

    return () => { cancelled = true; };
  }, [app, env, branch, stopped, targetScope]);

  // Stream logs when target changes
  useEffect(() => {
    if (!app || !env || !branch || state.targetScope !== targetScope) return undefined;
    const selectedTargetMeta = state.targets.find((t) => t.id === selectedTarget) ?? null;
    const effectiveTarget = selectedTargetMeta ? selectedTarget : state.defaultTarget;
    const effectiveTargetMeta = state.targets.find((t) => t.id === effectiveTarget) ?? null;
    const laneState = state.lane;

    if (!effectiveTarget) {
      setState((prev) => ({
        ...prev,
        lines: [],
        status: prev.lane.hasRunningTarget ? "idle" : "offline",
        targetMeta: null,
        error: "",
      }));
      return undefined;
    }

    if (effectiveTargetMeta && !effectiveTargetMeta.available) {
      setState((prev) => ({
        ...prev,
        lines: [],
        status: "offline",
        targetMeta: effectiveTargetMeta,
        error: "",
      }));
      return undefined;
    }

    const controller = new AbortController();
    let cancelled = false;
    setState((prev) => ({ ...prev, lines: [], status: "connecting", targetMeta: effectiveTargetMeta, error: "" }));

    async function run() {
      try {
        const params = new URLSearchParams({
          app: app!,
          env: env!,
          branch: branch!,
          target: effectiveTarget,
          tail: "300",
          since: sinceISO(windowFilter),
        });
        const res = await fetch(`/api/runtime/logs/stream?${params.toString()}`, {
          credentials: "include",
          signal: controller.signal,
        });
        if (!res.ok) {
          const contentType = res.headers.get("content-type") ?? "";
          const payload = contentType.includes("application/json") ? await res.json() : await res.text();
          const message = (typeof payload === "object" && payload?.error)
            || (typeof payload === "string" && payload)
            || `HTTP ${res.status}`;
          throw new Error(humanizeRuntimeStreamError(String(message), laneState, effectiveTargetMeta));
        }
        setState((prev) => ({ ...prev, status: "live" }));

        const reader = res.body!.getReader();
        const decoder = new TextDecoder("utf-8");
        let buffer = "";

        while (!cancelled) {
          const { value, done } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          let idx: number;
          while ((idx = buffer.indexOf("\n\n")) >= 0) {
            const frame = buffer.slice(0, idx);
            buffer = buffer.slice(idx + 2);
            let eventName = "message";
            const data: string[] = [];
            frame.split("\n").forEach((line) => {
              if (line.startsWith("event: ")) eventName = line.slice(7).trim();
              if (line.startsWith("data: ")) data.push(line.slice(6));
            });
            if (!data.length) continue;
            const payload = data.join("\n");
            if (eventName === "runtime-target") {
              try {
                const parsed = JSON.parse(payload) as RuntimeTarget;
                setState((prev) => ({ ...prev, targetMeta: parsed }));
              } catch { /* ignored */ }
              continue;
            }
            if (eventName === "runtime-status") {
              try {
                const parsed = JSON.parse(payload) as { status: string; error?: string };
                setState((prev) => {
                  const nextError = humanizeRuntimeStreamError(parsed.error ?? prev.error, prev.lane, prev.targetMeta ?? effectiveTargetMeta);
                  return {
                    ...prev,
                    status: parsed.status === "error" && isRuntimeOfflineError(nextError, prev.lane)
                      ? "offline"
                      : (parsed.status || "complete"),
                    error: parsed.status === "error" ? nextError : (parsed.error ? nextError : prev.error),
                  };
                });
              } catch {
                setState((prev) => ({ ...prev, status: payload || "complete" }));
              }
              continue;
            }
            setState((prev) => ({
              ...prev,
              lines: [...prev.lines, parseRuntimeLogEntry(payload, prev.targetMeta?.label ?? effectiveTarget)].slice(-500),
            }));
          }
        }

        setState((prev) => ({
          ...prev,
          status: prev.status === "live" ? "complete" : prev.status,
        }));
      } catch (err) {
        if (!cancelled && !controller.signal.aborted) {
          const message = humanizeRuntimeStreamError((err as Error).message, laneState, effectiveTargetMeta);
          setState((prev) => ({
            ...prev,
            status: isRuntimeOfflineError(message, laneState) ? "offline" : "error",
            error: message,
          }));
        }
      }
    }

    run();
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [app, env, branch, selectedTarget, state.defaultTarget, state.lane, state.targets, state.targetScope, targetScope, windowFilter]);

  return state;
}
