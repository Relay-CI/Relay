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
  access_role?: string;
  project_root?: string;
  build_context?: string;
  dockerfile?: string;
  engine?: string;
  mode?: string;
  traffic_mode?: string;
  access_policy?: string;
  ip_allowlist?: string;
  expires_at?: number;
  host_port?: number;
  service_port?: number;
  public_host?: string;
  public_hosts?: string[];
  repo_url?: string;
  webhook_secret?: string;
  notification_webhooks?: string;
  traffic_split_percent?: number;
  rollout_min_requests?: number;
  rollout_error_percent?: number;
  rollout_assess_seconds?: number;
  rollout_started_at?: number;
  rollout_deploy_id?: string;
  rollout_status?: string;
  stopped?: boolean;
  cpu_limit?: string;
  mem_limit?: string;
  resource_mode?: "manual" | "auto";
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

export interface CreateDeployRequest {
  app: string;
  env: string;
  branch: string;
  repo_url?: string;
  source?: string;
  workspace_env?: string;
  workspace_branch?: string;
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

export async function createDeploy(payload: CreateDeployRequest): Promise<Deploy> {
  return apiFetch<Deploy>("/api/deploys", {
    method: "POST",
    body: JSON.stringify(payload),
  });
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

export async function deleteLane(target: AppTarget): Promise<void> {
  await apiFetch("/api/apps/delete-lane", {
    method: "POST",
    body: JSON.stringify({ ...target, confirm: "DELETE" }),
  });
}

export async function restartApp(target: AppTarget): Promise<void> {
  await apiFetch("/api/apps/restart", {
    method: "POST",
    body: JSON.stringify(target),
  });
}

export async function rollback(
  target: AppTarget,
  source?: { env: string; branch?: string },
): Promise<void> {
  const payload: Record<string, string> = {
    app: target.app,
    env: target.env,
    branch: target.branch,
  };
  if (source?.env) {
    payload.source_env = source.env;
    if (source.branch) payload.source_branch = source.branch;
  }
  await apiFetch("/api/deploys/rollback", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function cancelDeploy(deployId: string): Promise<void> {
  await apiFetch(`/api/deploys/cancel/${deployId}`, { method: "POST" });
}

// ── App config ────────────────────────────────────────────────────────────

export interface AppConfig {
  repo_url?: string;
  project_root?: string;
  build_context?: string;
  dockerfile?: string;
  engine?: string;
  mode?: string;
  traffic_mode?: string;
  access_policy?: string;
  ip_allowlist?: string;
  expires_at?: number;
  host_port?: number;
  service_port?: number;
  public_host?: string;
  public_hosts?: string[];
  webhook_secret?: string;
  notification_webhooks?: string;
  traffic_split_percent?: number;
  rollout_min_requests?: number;
  rollout_error_percent?: number;
  rollout_assess_seconds?: number;
  rollout_started_at?: number;
  rollout_deploy_id?: string;
  rollout_status?: string;
  cpu_limit?: string;
  mem_limit?: string;
  resource_mode?: "manual" | "auto";
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

export async function setSecretsRaw(
  target: AppTarget,
  raw: string,
  replace = false,
): Promise<{ ok: boolean; count?: number; replace?: boolean }> {
  return apiFetch<{ ok: boolean; count?: number; replace?: boolean }>(
    "/api/apps/secrets",
    {
      method: "POST",
      body: JSON.stringify({ ...target, raw, replace }),
    },
  );
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
  acme_disabled?: string;
  custom_host_rules?: CustomHostRule[];
  theme_name?: string;
  theme_css?: string;
  plugin_mutations_enabled?: boolean;
  doctor?: DoctorReport;
  [key: string]: unknown;
}

export interface CustomHostRule {
  id?: string;
  host: string;
  action: "redirect" | "reverse_proxy" | "static_response" | "relay_dashboard";
  redirect_url?: string;
  redirect_code?: number;
  preserve_path?: boolean;
  upstream_url?: string;
  response_status?: number;
  response_body?: string;
  response_content_type?: string;
}

export interface AdminOpsContainerUsage {
  id: string;
  label: string;
  kind: string;
  container: string;
  running: boolean;
  cpu_percent: number;
  mem_usage_bytes: number;
  mem_limit_bytes: number;
  mem_percent: number;
  storage_bytes: number;
  net_rx_bytes: number;
  net_tx_bytes: number;
  block_read_bytes: number;
  block_write_bytes: number;
}

export interface AdminOpsLaneUsage {
  cpu_percent: number;
  mem_usage_bytes: number;
  mem_limit_bytes: number;
  mem_percent: number;
  storage_bytes: number;
  net_rx_bytes: number;
  net_tx_bytes: number;
  block_read_bytes: number;
  block_write_bytes: number;
  running_containers: number;
  container_count: number;
  measured: boolean;
  note?: string;
  targets: AdminOpsContainerUsage[];
}

export interface AdminOpsTrafficWindow {
  requests: number;
  server_errors: number;
  client_errors: number;
  bandwidth_bytes: number;
  server_error_rate: number;
}

export interface AdminOpsDeploySummary {
  id: string;
  status: string;
  source?: string;
  created_at: string;
  started_at?: string;
  ended_at?: string;
  build_number?: number;
  build_duration_ms?: number;
  commit_sha?: string;
  commit_message?: string;
  deployed_by?: string;
  image_tag?: string;
  previous_image_tag?: string;
}

export interface AdminOpsDeployDelta {
  current: AdminOpsDeploySummary;
  previous?: AdminOpsDeploySummary;
  window: { seconds: number };
  current_traffic?: AdminOpsTrafficWindow;
  previous_traffic?: AdminOpsTrafficWindow;
  build_duration_delta_ms?: number;
  server_error_rate_delta?: number;
  request_delta?: number;
  bandwidth_delta_bytes?: number;
  analytics_available: boolean;
  analytics_note?: string;
}

export interface AdminOpsLane {
  app: string;
  env: string;
  branch: string;
  engine: string;
  public_host?: string;
  host_port?: number;
  repo_url?: string;
  stopped: boolean;
  current_image?: string;
  previous_image?: string;
  usage: AdminOpsLaneUsage;
  latest?: AdminOpsDeployDelta;
}

export interface AdminOpsApp {
  app: string;
  lane_count: number;
  online_lanes: number;
  usage: AdminOpsLaneUsage;
  lanes: AdminOpsLane[];
}

export interface AdminOpsResponse {
  generated_at: number;
  summary: {
    app_count: number;
    lane_count: number;
    online_lanes: number;
    cpu_percent: number;
    mem_usage_bytes: number;
    mem_limit_bytes: number;
    mem_percent: number;
    storage_bytes: number;
  };
  apps: AdminOpsApp[];
}

export interface DoctorCheck {
  status: "ok" | "warn" | "error" | "info";
  summary: string;
  detail?: string;
  hint?: string;
}

export interface DoctorReport {
  generated_at: number;
  http_addr: string;
  socket_path?: string;
  base_domain?: string;
  dashboard_host?: string;
  managed_example_url?: string;
  webhook_url?: string;
  checks: Record<string, DoctorCheck>;
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

export interface PublicTheme {
  theme_name?: string;
  theme_css?: string;
}

export async function getPublicTheme(): Promise<PublicTheme> {
  return apiFetch<PublicTheme>("/api/public/theme");
}

export async function getDoctor(): Promise<DoctorReport> {
  return apiFetch<DoctorReport>("/api/doctor");
}

export async function getAdminOps(): Promise<AdminOpsResponse> {
  return apiFetch<AdminOpsResponse>("/api/admin/ops");
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
  env?: string,
  branch?: string,
  period?: string,
): Promise<unknown> {
  const params = new URLSearchParams();
  if (app) params.set("app", app);
  if (env) params.set("env", env);
  if (branch) params.set("branch", branch);
  if (period) params.set("period", period);
  return apiFetch(`/api/analytics?${params}`);
}

// ── Users ─────────────────────────────────────────────────────────────────

export interface User {
  id: string;
  username: string;
  role: string;
  created_at: string;
  permissions?: UserPermission[];
}

export interface UserPermission {
  app: string;
  env: string;
  role: string;
}

export async function getUsers(): Promise<User[]> {
  return apiFetch<User[]>("/api/users");
}

export async function createUser(data: {
  username: string;
  password: string;
  role: string;
  permissions?: UserPermission[];
}): Promise<void> {
  await apiFetch("/api/users", { method: "POST", body: JSON.stringify(data) });
}

export async function updateUser(
  id: string,
  patch: { role?: string; permissions?: UserPermission[] },
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

export interface BuildpackPlugin {
  name: string;
  description?: string;
  priority?: number;
  detect: {
    kind?: string;
    kinds?: string[];
    files_any?: string[];
    files_all?: string[];
    dirs_any?: string[];
    dirs_all?: string[];
    package_deps_any?: string[];
    package_deps_all?: string[];
    file_extensions_any?: string[];
    file_extensions_all?: string[];
  };
  plan: {
    kind?: string;
    service_port?: number;
    build_image?: string;
    run_image?: string;
    install_cmd?: string;
    build_cmd?: string;
    start_cmd?: string;
    dockerfile_template: string;
    write_default_conf?: boolean;
    cleanup_paths?: string[];
  };
}

export interface PluginCatalogEntry {
  name: string;
  description?: string;
  tags?: string[];
  homepage?: string;
  source_url?: string;
  installed: boolean;
}

export async function getBuildpackPlugins(): Promise<BuildpackPlugin[]> {
  return apiFetch<BuildpackPlugin[]>("/api/plugins/buildpacks");
}

export async function getPluginCatalog(): Promise<PluginCatalogEntry[]> {
  return apiFetch<PluginCatalogEntry[]>("/api/plugins/catalog");
}

export async function installCatalogPlugin(
  name: string,
): Promise<BuildpackPlugin> {
  return apiFetch<BuildpackPlugin>("/api/plugins/catalog", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export async function installBuildpackPlugin(
  plugin: BuildpackPlugin,
): Promise<BuildpackPlugin> {
  return apiFetch<BuildpackPlugin>("/api/plugins/buildpacks", {
    method: "POST",
    body: JSON.stringify(plugin),
  });
}

export async function installBuildpackPluginFromURL(
  url: string,
  sha256?: string,
): Promise<BuildpackPlugin> {
  return apiFetch<BuildpackPlugin>("/api/plugins/buildpacks/install-url", {
    method: "POST",
    body: JSON.stringify({ url, sha256 }),
  });
}

export async function removeBuildpackPlugin(name: string): Promise<void> {
  await apiFetch(`/api/plugins/buildpacks/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
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
