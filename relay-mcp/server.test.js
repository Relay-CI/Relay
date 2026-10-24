"use strict";

const { test } = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const path = require("node:path");
const { Client } = require("@modelcontextprotocol/sdk/client/index.js");
const { StdioClientTransport } = require("@modelcontextprotocol/sdk/client/stdio.js");
const { version } = require("./package.json");

test("stdio handshake, tool discovery, diagnostics, defaults, errors, and recovery", async (t) => {
  const calls = [];
  const payloads = [];
  const requests = [];
  const agent = http.createServer((req, res) => {
    calls.push(req.url);
    let body = "";
    req.on("data", (chunk) => { body += chunk; });
    req.on("end", () => {
      requests.push({ method: req.method, path: req.url, body: body ? JSON.parse(body) : undefined });
      if (body) payloads.push({ path: req.url, body: JSON.parse(body) });
      if (req.url === "/api/doctor") res.end(JSON.stringify({ checks: { docker: { status: "ok" } } }));
      else if (req.url === "/api/deploys/missing") { res.writeHead(404); res.end(JSON.stringify({ error: "deploy not found" })); }
      else if (req.url === "/api/variants" && JSON.parse(body || "{}").name === "restricted") { res.writeHead(403); res.end(JSON.stringify({ error: "insufficient lane access" })); }
      else res.end(JSON.stringify({ ok: true }));
    });
  });
  await new Promise((resolve) => agent.listen(0, "127.0.0.1", resolve));
  t.after(() => new Promise((resolve) => { agent.close(resolve); agent.closeAllConnections(); }));
  const transport = new StdioClientTransport({
    command: process.execPath,
    args: [path.join(__dirname, "index.js")],
    env: { ...process.env, RELAY_SOCKET: "", RELAY_URL: `http://127.0.0.1:${agent.address().port}`, RELAY_TOKEN: "test-token" },
    stderr: "pipe",
  });
  const client = new Client({ name: "relay-test", version: "1.0.0" });
  t.after(() => client.close());
  await client.connect(transport);
  assert.equal(client.getServerVersion().version, version);
  const { tools } = await client.listTools();
  assert.equal(tools.length, 43);
  assert.equal(tools.find((tool) => tool.name === "get_doctor").annotations.readOnlyHint, true);
  assert.equal(tools.find((tool) => tool.name === "delete_lane").annotations.destructiveHint, true);
  const doctor = await client.callTool({ name: "get_doctor", arguments: {} });
  assert.equal(JSON.parse(doctor.content[0].text).checks.docker.status, "ok");
  const failed = await client.callTool({ name: "get_deploy", arguments: { id: "missing" } });
  assert.equal(failed.isError, true);
  assert.match(failed.content[0].text, /HTTP 404/);
  await client.callTool({ name: "list_deploys", arguments: { app: "demo", branch: "feature/docs" } });
  assert.ok(calls.includes("/api/deploys?app=demo&branch=feature%2Fdocs&limit=20"));
  const recovered = await client.callTool({ name: "get_version", arguments: {} });
  assert.equal(recovered.isError, undefined);
  assert.deepEqual(JSON.parse(recovered.content[0].text), { ok: true });
  await client.callTool({ name: "create_signed_link", arguments: { app: "demo", env: "dev", expires_in: 61 } });
  assert.deepEqual(payloads.find((entry) => entry.path === "/api/apps/signed-link").body, {
    app: "demo", env: "dev", branch: "main", expires_in_minutes: 2,
  });

  const lane = { app: "demo", env: "prod", branch: "feature/docs" };
  const cases = [
    ["list_variants", { ...lane, period: "24h" }, "GET", "/api/variants?app=demo&env=prod&branch=feature%2Fdocs&period=24h"],
    ["set_variant_weight", { ...lane, name: "checkout", weight: 0 }, "POST", "/api/variants", { ...lane, name: "checkout", weight: 0, action: "weight" }],
    ["promote_variant", { ...lane, name: "checkout" }, "POST", "/api/variants", { ...lane, name: "checkout", action: "promote" }],
    ["remove_variant", { ...lane, name: "checkout" }, "POST", "/api/variants", { ...lane, name: "checkout", action: "remove" }],
    ["reshuffle_variants", lane, "POST", "/api/variants", { ...lane, action: "reshuffle" }],
    ["scale_app", { ...lane, replicas: 3, apply: false }, "POST", "/api/apps/scale", { ...lane, replicas: 3, apply: false }],
    ["get_analytics", { ...lane, variant: "control", period: "30d" }, "GET", "/api/analytics?app=demo&env=prod&branch=feature%2Fdocs&period=30d&variant=control"],
    ["abort_rollout", lane, "POST", "/api/deploys/rollout-abort", lane],
    ["request_promotion", { app: "demo", source_env: "staging", target_env: "prod" }, "POST", "/api/promotions", { app: "demo", source_env: "staging", source_branch: "main", target_env: "prod", target_branch: "main" }],
    ["get_runtime_targets", lane, "GET", "/api/runtime/logs/targets?app=demo&env=prod&branch=feature%2Fdocs"],
    ["get_admin_ops", {}, "GET", "/api/admin/ops"],
    ["get_status", {}, "GET", "/api/status"],
    ["list_buildpack_catalog", {}, "GET", "/api/plugins/catalog"],
    ["install_buildpack_from_catalog", { name: "astro-static" }, "POST", "/api/plugins/catalog", { name: "astro-static" }],
    ["install_buildpack_from_url", { url: "https://example.test/plugin.json", sha256: "a".repeat(64) }, "POST", "/api/plugins/buildpacks/install-url", { url: "https://example.test/plugin.json", sha256: "a".repeat(64) }],
    ["set_app_config", { ...lane, cpu_limit: "0.5", mem_limit: "512m", resource_mode: "manual", traffic_mode: "edge", public_hosts: [] }, "POST", "/api/apps/config", { ...lane, cpu_limit: "0.5", mem_limit: "512m", resource_mode: "manual", traffic_mode: "edge", public_hosts: [] }],
  ];
  for (const [name, args, method, route, body] of cases) {
    const result = await client.callTool({ name, arguments: args });
    assert.notEqual(result.isError, true, name);
    assert.deepEqual(requests.at(-1), { method, path: route, body }, name);
    const annotations = tools.find((tool) => tool.name === name).annotations;
    assert.equal(annotations.readOnlyHint, method === "GET", name);
    assert.equal(annotations.destructiveHint, method !== "GET", name);
  }
  await client.callTool({ name: "scale_app", arguments: { app: "demo", env: "prod", replicas: 1 } });
  assert.deepEqual(requests.at(-1).body, { app: "demo", env: "prod", branch: "main", replicas: 1, apply: true });
  await client.callTool({ name: "list_variants", arguments: { app: "demo", env: "prod" } });
  assert.equal(requests.at(-1).path, "/api/variants?app=demo&env=prod&branch=main&period=7d");
  await client.callTool({ name: "request_promotion", arguments: { app: "demo", source_env: "staging", source_branch: "release/v2", target_env: "prod" } });
  assert.equal(requests.at(-1).body.target_branch, "release/v2");

  for (const [name, args] of [
    ["scale_app", { ...lane, replicas: 0 }],
    ["scale_app", { ...lane, replicas: 17 }],
    ["scale_app", { ...lane, replicas: 1.5 }],
    ["set_variant_weight", { ...lane, name: "checkout", weight: 101 }],
    ["set_variant_weight", { ...lane, name: "checkout", weight: -1 }],
    ["set_app_config", { ...lane, traffic_split_percent: 0 }],
    ["set_app_config", { ...lane, rollout_assess_seconds: 29 }],
    ["remove_variant", { ...lane, name: "control" }],
    ["list_variants", { ...lane, period: "forever" }],
    ["install_buildpack_from_url", { url: "http://example.test/plugin.json" }],
    ["install_buildpack_from_url", { url: "https://example.test/plugin.json", sha256: "invalid" }],
  ]) {
    const before = requests.length;
    const result = await client.callTool({ name, arguments: args });
    assert.equal(result.isError, true, name);
    assert.equal(requests.length, before, `${name} must reject invalid input before contacting the agent`);
  }
  const denied = await client.callTool({ name: "set_variant_weight", arguments: { ...lane, name: "restricted", weight: 10 } });
  assert.equal(denied.isError, true);
  assert.match(denied.content[0].text, /HTTP 403/);
  const afterDenied = await client.callTool({ name: "get_status", arguments: {} });
  assert.notEqual(afterDenied.isError, true);
});

test("network-only tools explain the socket limitation without connecting", async (t) => {
  const transport = new StdioClientTransport({
    command: process.execPath, args: [path.join(__dirname, "index.js")],
    env: { ...process.env, RELAY_SOCKET: path.join(__dirname, "missing-test.sock") }, stderr: "pipe",
  });
  const client = new Client({ name: "socket-test", version: "1.0.0" });
  t.after(() => client.close());
  await client.connect(transport);
  for (const name of ["get_analytics", "abort_rollout"]) {
    const result = await client.callTool({ name, arguments: { app: "demo", env: "prod" } });
    assert.equal(result.isError, true);
    assert.match(result.content[0].text, /requires HTTP\/HTTPS.*unset RELAY_SOCKET/);
  }
  const { tools } = await client.listTools();
  assert.equal(tools.length, 43);
});
