"use strict";

const { test } = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const { readConfig, createRelayRequest } = require("./transport.js");

async function agent(t, handler) {
  const server = http.createServer(handler);
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  t.after(() => new Promise((resolve) => {
    server.close(resolve);
    server.closeAllConnections();
  }));
  return `http://127.0.0.1:${server.address().port}`;
}

test("HTTP transport preserves proxy prefixes, filters, authentication, and JSON writes", async (t) => {
  const calls = [];
  const url = await agent(t, (req, res) => {
    let body = "";
    req.on("data", (chunk) => { body += chunk; });
    req.on("end", () => {
      calls.push({ path: req.url, token: req.headers["x-relay-token"], body });
      res.end(JSON.stringify({ ok: true }));
    });
  });
  const request = createRelayRequest(readConfig({ RELAY_URL: `${url}/relay/`, RELAY_TOKEN: "test-token" }));
  assert.deepEqual(await request("GET", "/api/deploys?branch=feature%2Fdocs"), { ok: true });
  await request("POST", "/api/apps/config", { app: "demo", env: "dev" });
  assert.equal(calls[0].path, "/relay/api/deploys?branch=feature%2Fdocs");
  assert.equal(calls[0].token, "test-token");
  assert.deepEqual(JSON.parse(calls[1].body), { app: "demo", env: "dev" });
});

test("stored logs and empty successful responses remain usable", async (t) => {
  const url = await agent(t, (req, res) => {
    if (req.url === "/empty") { res.writeHead(204); res.end(); }
    else res.end("build output\nsecond line");
  });
  const request = createRelayRequest(readConfig({ RELAY_URL: url }));
  assert.deepEqual(await request("GET", "/logs"), { text: "build output\nsecond line" });
  assert.deepEqual(await request("DELETE", "/empty"), {});
});

test("authentication and redirect failures explain the next action", async (t) => {
  const url = await agent(t, (req, res) => {
    if (req.url === "/redirect") { res.writeHead(302, { Location: "/login" }); res.end(); }
    else { res.writeHead(401); res.end(JSON.stringify({ error: "unauthorized" })); }
  });
  const request = createRelayRequest(readConfig({ RELAY_URL: url }));
  await assert.rejects(request("GET", "/private"), /HTTP 401.*RELAY_TOKEN/);
  await assert.rejects(request("GET", "/redirect"), /redirects are not followed/);
});

test("the deadline covers a response that starts but never finishes", async (t) => {
  const url = await agent(t, (_req, res) => { res.writeHead(200); res.write("{"); });
  const request = createRelayRequest(readConfig({ RELAY_URL: url, RELAY_REQUEST_TIMEOUT_MS: "40" }));
  await assert.rejects(request("GET", "/stalled"), /timed out after 40 ms/);
});

test("invalid configuration fails early; socket mode takes precedence over URL", () => {
  for (const RELAY_URL of ["not a url", "ftp://example.com", "https://user:pass@example.com", "https://example.com/?token=secret"]) {
    assert.throws(() => readConfig({ RELAY_URL }), /RELAY_URL/);
  }
  assert.throws(() => readConfig({ RELAY_REQUEST_TIMEOUT_MS: "0" }), /TIMEOUT/);
  assert.throws(() => readConfig({ RELAY_REQUEST_TIMEOUT_MS: "NaN" }), /TIMEOUT/);
  const config = readConfig({ RELAY_SOCKET: "/tmp/relay.sock", RELAY_URL: "invalid" });
  assert.equal(config.socketPath, "/tmp/relay.sock");
  assert.equal(config.baseUrl, undefined);
});

test("a missing socket reports a connection error with a recovery hint", async () => {
  const socket = process.platform === "win32" ? "\\\\.\\pipe\\relay-mcp-nonexistent-test" : "/tmp/relay-mcp-nonexistent-test.sock";
  const request = createRelayRequest(readConfig({ RELAY_SOCKET: socket }));
  await assert.rejects(request("GET", "/api/version"), /Check RELAY_SOCKET/);
});
