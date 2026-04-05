"use client";

import { useEffect, useState } from "react";
import { getRuntimeTargets, type RuntimeTarget } from "@/lib/api";
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
  loading: boolean;
  targets: RuntimeTarget[];
  defaultTarget: string;
  lines: RuntimeLogEntry[];
  status: string;
  targetMeta: RuntimeTarget | null;
  lane: RuntimeLaneState;
  error: string;
}

interface SelectedEnv {
  app: string;
  env: string;
  branch: string;
  stopped?: boolean;
}

export function useRuntimeLogs(
  selectedEnv: SelectedEnv | null | undefined,
  selectedTarget: string,
  windowFilter: string,
): RuntimeLogsState {
  const [state, setState] = useState<RuntimeLogsState>({
    loading: false,
    targets: [],
    defaultTarget: "",
    lines: [],
    status: "idle",
    targetMeta: null,
    lane: EMPTY_RUNTIME_LANE_STATE,
    error: "",
  });

  // Fetch available targets when env changes
  useEffect(() => {
    if (!selectedEnv?.app || !selectedEnv?.env || !selectedEnv?.branch) {
      setState({
        loading: false, targets: [], defaultTarget: "", lines: [],
        status: "idle", targetMeta: null, lane: EMPTY_RUNTIME_LANE_STATE, error: "",
      });
      return undefined;
    }

    let cancelled = false;
    setState((prev) => ({ ...prev, loading: true, error: "" }));

    getRuntimeTargets(selectedEnv.app, selectedEnv.env, selectedEnv.branch)
      .then((data) => {
        if (cancelled) return;
        const lane = normalizeRuntimeLaneState(data?.lane, selectedEnv);
        const defaultTarget = data?.default_target ?? "";
        setState((prev) => ({
          ...prev,
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
          loading: false,
          targets: [],
          defaultTarget: "",
          lines: [],
          status: "error",
          targetMeta: null,
          lane: normalizeRuntimeLaneState(null, selectedEnv),
          error: err.message,
        }));
      });

    return () => { cancelled = true; };
  }, [selectedEnv]);

  // Stream logs when target changes
  useEffect(() => {
    if (!selectedEnv?.app || !selectedEnv?.env || !selectedEnv?.branch) return undefined;
    const effectiveTarget = selectedTarget || state.defaultTarget;
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

    if (effectiveTargetMeta && !effectiveTargetMeta.running) {
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
          app: selectedEnv!.app,
          env: selectedEnv!.env,
          branch: selectedEnv!.branch,
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
  }, [selectedEnv, selectedTarget, state.defaultTarget, state.lane, state.targets, windowFilter]);

  return state;
}
