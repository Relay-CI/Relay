// ── API client ─────────────────────────────────────────────────────────────
// All requests are relative so they proxy through the relayd HTTP server.
// The dashboard is always served at /dashboard/* so the API lives at /api/*.

const API_BASE = "";

export interface ApiError extends Error {
  status: number;
}

export async function apiFetch<T = unknown>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      "X-Requested-With": "relay-dashboard",
      ...(options.headers as Record<string, string> | undefined),
    },
    ...options,
  });

  const contentType = res.headers.get("content-type") ?? "";
  const data = contentType.includes("application/json")
    ? await res.json()
    : await res.text();

  if (!res.ok) {
    const message =
      typeof data === "object" && data?.error
        ? data.error
        : `HTTP ${res.status}`;
    const err = new Error(message) as ApiError;
    err.status = res.status;
    throw err;
  }

  return data as T;
}

// ── Auth ──────────────────────────────────────────────────────────────────

export interface SessionInfo {
  authed: boolean;
  user?: { username: string; role: string };
  setup_available?: boolean;
  legacy_mode?: boolean;
  cli_mode?: boolean;
  cli_token?: string;
}

interface RawSessionInfo {
  authenticated?: boolean;
  username?: string;
  role?: string;
  setup_required?: boolean;
  legacy_mode?: boolean;
  cli_mode?: boolean;
  cli_token?: string;
}

export interface LoginRequest {
  username: string;
  password: string;
  cliPort?: number;
}

export interface LoginResponse {
  authenticated?: boolean;
  username?: string;
  role?: string;
  setup_required?: boolean;
  cli_code?: string;
  cli_redirect?: string;
}

export async function getSession(): Promise<SessionInfo> {
  const session = await apiFetch<RawSessionInfo>("/api/auth/session");
  return {
    authed: session.authenticated ?? false,
    user:
      session.username && session.role
        ? { username: session.username, role: session.role }
        : undefined,
    setup_available: session.setup_required ?? false,
    legacy_mode: session.legacy_mode ?? false,
    cli_mode: session.cli_mode ?? false,
    cli_token: session.cli_token,
  };
}

export async function login(
  credentials: LoginRequest | string,
): Promise<LoginResponse> {
  if (typeof credentials === "string") {
    return apiFetch<LoginResponse>("/api/auth/session", {
      method: "POST",
      body: JSON.stringify({ token: credentials }),
    });
  }

  return apiFetch<LoginResponse>("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({
      username: credentials.username,
      password: credentials.password,
      ...(credentials.cliPort ? { cli_port: credentials.cliPort } : {}),
    }),
  });
}

export async function cliStart(cliPort: number): Promise<{ cli_redirect?: string }> {
  return apiFetch<{ cli_redirect?: string }>("/api/auth/cli/start", {
    method: "POST",
    body: JSON.stringify({ cli_port: cliPort }),
  });
}

export async function logout(): Promise<void> {
  await apiFetch("/api/auth/session", { method: "DELETE" });
}

export async function setup(credentials: {
  username: string;
  password: string;
}): Promise<void> {
  await apiFetch("/api/auth/setup", {
    method: "POST",
    body: JSON.stringify(credentials),
  });
}

export async function getMe(): Promise<{ username: string; role: string }> {
  const me = await apiFetch<{
    authenticated?: boolean;
    username?: string;
    role?: string;
  }>("/api/auth/me");
  return {
    username: me.username ?? "",
    role: me.role ?? "",
  };
}

// ── Projects + Deploys ────────────────────────────────────────────────────

export interface EnvInfo {
  app: string;
  env: string;
  branch: string;
  engine?: string;
  mode?: string;
  traffic_mode?: string;
  access_policy?: string;
  ip_allowlist?: string;
  expires_at?: number;
  host_port?: number;
  service_port?: number;
  public_host?: string;
  repo_url?: string;
  webhook_secret?: string;
  stopped?: boolean;
  active_slot?: string;
  standby_slot?: string;
  drain_until?: number;
  latestDeploy?: Deploy;
  previewURL?: string;
}

export interface Project {
  name: string;
  envs: EnvInfo[];
  services: Service[];
  latestDeployAt?: string;
}

export interface Deploy {
  id: string;
  app: string;
  env: string;
  branch: string;
  status: string;
  source?: string;
  created_at: string;
  started_at?: string;
  ended_at?: string;
  commit_sha?: string;
  commit_message?: string;
  image_tag?: string;
  imageTag?: string;
  previous_image_tag?: string;
  prevImage?: string;
  build_number?: number;
  preview_url?: string;
  deployed_by?: string;
  log?: string;
}

export interface Service {
  name: string;
  type: string;
  container: string;
  network: string;
  env_key: string;
  env_val: string;
  env?: string;
  branch?: string;
  version?: string;
  running?: boolean;
  port?: number;
}

export async function getProjects(): Promise<Project[]> {
  return apiFetch<Project[]>("/api/projects");
}

export async function getDeploys(): Promise<Deploy[]> {
  return apiFetch<Deploy[]>("/api/deploys");
}

export async function getDeployById(id: string): Promise<Deploy> {
  return apiFetch<Deploy>(`/api/deploys/${id}`);
}

// ── Deployment operations ─────────────────────────────────────────────────

export interface AppTarget {
  app: string;
  env: string;
  branch: string;
}

export async function startApp(target: AppTarget): Promise<void> {
  await apiFetch("/api/apps/start", {
    method: "POST",
    body: JSON.stringify(target),
  });
}

export async function stopApp(target: AppTarget): Promise<void> {
  await apiFetch("/api/apps/stop", {
    method: "POST",
    body: JSON.stringify(target),
  });
}

export async function restartApp(target: AppTarget): Promise<void> {
  await apiFetch("/api/apps/restart", {
    method: "POST",
    body: JSON.stringify(target),
  });
}

export async function rollback(deployId: string): Promise<void> {
  await apiFetch("/api/deploys/rollback", {
    method: "POST",
    body: JSON.stringify({ deploy_id: deployId }),
  });
}

export async function cancelDeploy(deployId: string): Promise<void> {
  await apiFetch(`/api/deploys/cancel/${deployId}`, { method: "POST" });
}

// ── App config ────────────────────────────────────────────────────────────

export interface AppConfig {
  repo_url?: string;
  engine?: string;
  mode?: string;
  traffic_mode?: string;
  access_policy?: string;
  ip_allowlist?: string;
  expires_at?: number;
  host_port?: number;
  service_port?: number;
  public_host?: string;
  webhook_secret?: string;
}

export async function getAppConfig(target: AppTarget): Promise<AppConfig> {
  const params = new URLSearchParams(
    target as unknown as Record<string, string>,
  );
  return apiFetch<AppConfig>(`/api/apps/config?${params}`);
}

export async function saveAppConfig(
  target: AppTarget,
  config: AppConfig,
): Promise<void> {
  await apiFetch("/api/apps/config", {
    method: "POST",
    body: JSON.stringify({ ...target, ...config }),
  });
}

export interface SignedLinkResponse {
  url: string;
  base_url: string;
  access_policy: string;
  expires_at: number;
}

export async function generateSignedLink(
  target: AppTarget,
  expiresInMinutes = 24 * 60,
): Promise<SignedLinkResponse> {
  const params = new URLSearchParams({
    ...target,
    expires_in_minutes: String(expiresInMinutes),
  });
  return apiFetch<SignedLinkResponse>(`/api/apps/signed-link?${params}`);
}

export interface PromotionRecord {
  id: string;
  app: string;
  source_env: string;
  source_branch: string;
  source_deploy_id?: string;
  source_image?: string;
  target_env: string;
  target_branch: string;
  status: string;
  approval_required: boolean;
  requested_by?: string;
  requested_at: number;
  approved_by?: string;
  approved_at?: number;
  target_deploy_id?: string;
  rollback_deploy_id?: string;
  health_status?: string;
  health_detail?: string;
}

export async function getPromotions(
  app: string,
  sourceEnv?: string,
  branch?: string,
): Promise<PromotionRecord[]> {
  const params = new URLSearchParams({ app });
  if (sourceEnv) params.set("source_env", sourceEnv);
  if (branch) params.set("branch", branch);
  return apiFetch<PromotionRecord[]>(`/api/promotions?${params}`);
}

export async function requestPromotion(payload: {
  app: string;
  source_env: string;
  source_branch: string;
  target_env?: string;
  target_branch?: string;
}): Promise<PromotionRecord> {
  return apiFetch<PromotionRecord>("/api/promotions", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function approvePromotion(id: string): Promise<PromotionRecord> {
  return apiFetch<PromotionRecord>("/api/promotions/approve", {
    method: "POST",
    body: JSON.stringify({ id }),
  });
}

// ── Secrets ───────────────────────────────────────────────────────────────

export interface Secret {
  key: string;
}

export async function getSecrets(target: AppTarget): Promise<Secret[]> {
  const params = new URLSearchParams(
    target as unknown as Record<string, string>,
  );
  return apiFetch<Secret[]>(`/api/apps/secrets?${params}`);
}

export async function setSecret(
  target: AppTarget,
  key: string,
  value: string,
): Promise<void> {
  await apiFetch("/api/apps/secrets", {
    method: "POST",
    body: JSON.stringify({ ...target, key, value }),
  });
}

export async function deleteSecret(
  target: AppTarget,
  key: string,
): Promise<void> {
  const params = new URLSearchParams({ ...target, key });
  await apiFetch(`/api/apps/secrets?${params}`, { method: "DELETE" });
}

// ── Companions ────────────────────────────────────────────────────────────

export interface CompanionConfig {
  name?: string;
  type?: string;
  version?: string;
  image?: string;
  command?: string;
  stopped?: boolean;
  disabled?: boolean;
  port?: number;
  host_port?: number;
  env?: Record<string, string>;
  volumes?: string[];
  health?: {
    test: string;
    interval_seconds?: number;
    timeout_seconds?: number;
    retries?: number;
    start_period_seconds?: number;
  };
}

export interface Companion {
  config: CompanionConfig;
  managed: boolean;
  source?: string;
  updated_at?: number;
  running?: Record<string, unknown>;
}

// Alias for convenience
export type { CompanionConfig as ServiceConfig };

export async function getCompanions(target: AppTarget): Promise<Companion[]> {
  const params = new URLSearchParams(
    target as unknown as Record<string, string>,
  );
  return apiFetch<Companion[]>(`/api/apps/companions?${params}`);
}

export async function saveCompanion(
  target: AppTarget,
  config: CompanionConfig,
): Promise<void> {
  await apiFetch("/api/apps/companions", {
    method: "POST",
    body: JSON.stringify({ ...target, config }),
  });
}

export async function deleteCompanion(
  target: AppTarget,
  name: string,
): Promise<void> {
  const params = new URLSearchParams({ ...target, name });
  await apiFetch(`/api/apps/companions?${params}`, { method: "DELETE" });
}

export async function restartCompanion(
  target: AppTarget,
  name: string,
): Promise<void> {
  await apiFetch("/api/apps/companions/restart", {
    method: "POST",
    body: JSON.stringify({ ...target, name }),
  });
}

// ── Server version ────────────────────────────────────────────────────────

export interface VersionInfo {
  version: string;
  commit: string;
  build_date: string;
  os: string;
  arch: string;
}

export async function getVersion(): Promise<VersionInfo> {
  return apiFetch("/api/version");
}

// ── Server config ─────────────────────────────────────────────────────────

export interface ServerConfig {
  base_domain?: string;
  dashboard_host?: string;
  [key: string]: unknown;
}

export async function getServerConfig(): Promise<ServerConfig> {
  return apiFetch("/api/server/config");
}

export async function saveServerConfig(
  config: ServerConfig,
): Promise<ServerConfig> {
  return apiFetch("/api/server/config", {
    method: "POST",
    body: JSON.stringify(config),
  });
}

// ── Build logs ────────────────────────────────────────────────────────────

export async function getDeployLogs(deployId: string): Promise<string[]> {
  return apiFetch<string[]>(`/api/logs/${deployId}`);
}

// ── Runtime logs ──────────────────────────────────────────────────────────

export interface RuntimeTarget {
  id: string;
  label: string;
  running: boolean;
}

export interface RuntimeTargetsResponse {
  targets: RuntimeTarget[];
  default_target?: string;
  lane?: {
    app_stopped?: boolean;
    app_running?: boolean;
    has_running_target?: boolean;
    active_slot?: string;
    standby_slot?: string;
    offline_reason?: string;
  };
}

export async function getRuntimeTargets(
  app: string,
  env: string,
  branch: string,
): Promise<RuntimeTargetsResponse> {
  return apiFetch<RuntimeTargetsResponse>(
    `/api/runtime/logs/targets?app=${encodeURIComponent(app)}&env=${encodeURIComponent(env)}&branch=${encodeURIComponent(branch)}`,
  );
}

// ── Analytics ─────────────────────────────────────────────────────────────

export async function getAnalytics(
  app?: string,
  period?: string,
): Promise<unknown> {
  const params = new URLSearchParams();
  if (app) params.set("app", app);
  if (period) params.set("period", period);
  return apiFetch(`/api/analytics?${params}`);
}

// ── Users ─────────────────────────────────────────────────────────────────

export interface User {
  id: string;
  username: string;
  role: string;
  created_at: string;
}

export async function getUsers(): Promise<User[]> {
  return apiFetch<User[]>("/api/users");
}

export async function createUser(data: {
  username: string;
  password: string;
  role: string;
}): Promise<void> {
  await apiFetch("/api/users", { method: "POST", body: JSON.stringify(data) });
}

export async function updateUser(
  id: string,
  patch: { role: string },
): Promise<void> {
  await apiFetch(`/api/users/${id}`, {
    method: "PATCH",
    body: JSON.stringify(patch),
  });
}

export async function deleteUser(id: string): Promise<void> {
  await apiFetch(`/api/users/${id}`, { method: "DELETE" });
}

// ── Audit log ─────────────────────────────────────────────────────────────

export interface AuditEntry {
  id: string;
  actor: string;
  action: string;
  target?: string;
  detail?: string;
  ts?: string;
  created_at: string;
}

export async function getAuditLog(limit?: number): Promise<AuditEntry[]> {
  const params = limit ? `?limit=${limit}` : "";
  return apiFetch<AuditEntry[]>(`/api/audit${params}`);
}

// ── Plugins ───────────────────────────────────────────────────────────────

export async function getBuildpackPlugins(): Promise<unknown[]> {
  return apiFetch<unknown[]>("/api/plugins/buildpacks");
}

// ── Project delete ────────────────────────────────────────────────────────

export async function deleteProject(name: string): Promise<void> {
  await apiFetch("/api/projects/delete", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

// ── CLI auth ──────────────────────────────────────────────────────────────

export async function cliExchange(code: string): Promise<void> {
  await apiFetch("/api/auth/cli/exchange", {
    method: "POST",
    body: JSON.stringify({ code }),
  });
}
