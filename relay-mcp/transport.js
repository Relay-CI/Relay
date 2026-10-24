"use strict";

const http = require("node:http");
const https = require("node:https");

function readConfig(env = process.env) {
  const socketPath = env.RELAY_SOCKET?.trim() || null;
  const timeoutMs = Number(env.RELAY_REQUEST_TIMEOUT_MS || 30000);
  if (!Number.isSafeInteger(timeoutMs) || timeoutMs < 1 || timeoutMs > 600000) {
    throw new Error("RELAY_REQUEST_TIMEOUT_MS must be an integer between 1 and 600000.");
  }
  let baseUrl;
  if (!socketPath) {
    try {
      baseUrl = new URL(env.RELAY_URL || "http://127.0.0.1:8080");
    } catch {
      throw new Error("RELAY_URL must be an absolute HTTP or HTTPS URL.");
    }
    if (!["http:", "https:"].includes(baseUrl.protocol) || baseUrl.username || baseUrl.password || baseUrl.search || baseUrl.hash) {
      throw new Error("RELAY_URL must use HTTP or HTTPS without credentials, a query, or a fragment. Use RELAY_TOKEN for authentication.");
    }
  }
  return { socketPath, baseUrl, token: env.RELAY_TOKEN || "", timeoutMs };
}

function createRelayRequest(config) {
  return function relayRequest(method, apiPath, body) {
    return new Promise((resolve, reject) => {
      const bodyStr = body === undefined ? null : JSON.stringify(body);
      const headers = { Accept: "application/json", "Content-Type": "application/json" };
      if (config.token && !config.socketPath) headers["X-Relay-Token"] = config.token;
      if (bodyStr !== null) headers["Content-Length"] = Buffer.byteLength(bodyStr);
      let deadline;
      let req;
      let settled = false;
      const finish = (error, value) => {
        if (settled) return;
        settled = true;
        clearTimeout(deadline);
        if (error) reject(error);
        else resolve(value);
      };
      const onResponse = (res) => {
        const chunks = [];
        let bytes = 0;
        res.on("data", (chunk) => {
          bytes += chunk.length;
          if (bytes > 16 * 1024 * 1024) {
            const error = new Error("Relay response exceeded 16 MiB. Request fewer records or inspect the logs in the dashboard.");
            finish(error);
            req.destroy(error);
            return;
          }
          chunks.push(chunk);
        });
        res.on("aborted", () => finish(new Error("The Relay agent closed the response early. Check the agent and proxy logs.")));
        res.on("error", (error) => finish(error));
        res.on("end", () => {
          const raw = Buffer.concat(chunks).toString("utf8");
          const status = res.statusCode;
          if (status < 200 || status >= 300) {
            let detail = raw.trim().slice(0, 1000);
            try { detail = String(JSON.parse(raw).error || detail).slice(0, 1000); } catch { /* A proxy can return plain text. */ }
            const hint = status === 401 ? " Check RELAY_TOKEN for this server."
              : status === 403 ? " Check the account's role and permissions."
              : status >= 300 && status < 400 ? " Set RELAY_URL to the final agent URL; redirects are not followed." : "";
            finish(new Error(`Relay HTTP ${status}${detail ? `: ${detail}` : ""}.${hint}`));
            return;
          }
          if (!raw.trim()) return finish(null, {});
          try { finish(null, JSON.parse(raw)); }
          catch { finish(null, { text: raw }); }
        });
      };
      if (config.socketPath) {
        req = http.request({ socketPath: config.socketPath, method, path: apiPath, headers }, onResponse);
      } else {
        const target = new URL(config.baseUrl);
        target.pathname = target.pathname.replace(/\/$/, "") + apiPath.split("?")[0];
        target.search = apiPath.includes("?") ? apiPath.slice(apiPath.indexOf("?")) : "";
        const client = target.protocol === "https:" ? https : http;
        req = client.request(target, { method, headers }, onResponse);
      }
      req.on("error", (error) => {
        const hint = config.socketPath ? "Check RELAY_SOCKET and filesystem permissions."
          : "Check RELAY_URL and whether relayd is running.";
        finish(new Error(`Cannot reach the Relay agent (${error.code || error.message}). ${hint}`));
      });
      deadline = setTimeout(() => {
        const error = new Error(`Relay request timed out after ${config.timeoutMs} ms. Check the agent or increase RELAY_REQUEST_TIMEOUT_MS.`);
        finish(error);
        req.destroy(error);
      }, config.timeoutMs);
      if (bodyStr !== null) req.write(bodyStr);
      req.end();
    });
  };
}

module.exports = { readConfig, createRelayRequest };
