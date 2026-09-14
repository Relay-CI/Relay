"use strict";

const fs = require("fs");
const fsp = require("fs/promises");
const path = require("path");
const http = require("http");
const https = require("https");
const zlib = require("zlib");
const tar = require("tar-stream");

function authHeaders(token) {
  return token
    ? { "X-Relay-Token": token, Authorization: `Bearer ${token}` }
    : {};
}

function addFile(pack, rootDir, rel) {
  return new Promise(async (resolve, reject) => {
    try {
      const abs = path.join(rootDir, rel.split("/").join(path.sep));
      const stat = await fsp.stat(abs);
      const entry = pack.entry(
        {
          name: rel,
          size: stat.size,
          mode: stat.mode & 0o777,
          mtime: stat.mtime,
          type: "file",
        },
        (err) => (err ? reject(err) : resolve()),
      );
      const input = fs.createReadStream(abs);
      input.on("error", reject);
      entry.on("error", reject);
      input.pipe(entry);
    } catch (err) {
      reject(err);
    }
  });
}

function bundleRequest(transport, apiPath) {
  const headers = {
    ...authHeaders(transport.token),
    "Content-Type": "application/x-tar",
    "Content-Encoding": "gzip",
  };
  if (transport.kind === "socket") {
    return {
      client: http,
      target: {
        socketPath: transport.socketPath,
        method: "PUT",
        path: apiPath,
        headers,
      },
    };
  }
  const url = new URL(`${transport.baseUrl.replace(/\/$/, "")}${apiPath}`);
  return {
    client: url.protocol === "https:" ? https : http,
    target: { method: "PUT", headers, protocol: url.protocol, hostname: url.hostname, port: url.port, path: `${url.pathname}${url.search}` },
  };
}

// Streams all changed files as one gzip-compressed tar request. Compression
// level 1 is deliberate: source uploads are network/RTT bound, and low CPU
// overhead matters on the same 4 GB hosts that are about to run a build.
function putBundle(transport, apiPath, rootDir, relPaths) {
  return new Promise((resolve, reject) => {
    const { client, target } = bundleRequest(transport, apiPath);
    const pack = tar.pack();
    const gzip = zlib.createGzip({ level: 1 });
    let settled = false;
    const finish = (err, value) => {
      if (settled) return;
      settled = true;
      if (err) reject(err);
      else resolve(value);
    };
    const req = client.request(target, (res) => {
      const chunks = [];
      res.on("data", (chunk) => chunks.push(chunk));
      res.on("end", () => {
        const text = Buffer.concat(chunks).toString("utf8");
        if ((res.statusCode || 500) >= 400) {
          const err = new Error(`bundle upload HTTP ${res.statusCode}: ${text}`);
          err.status = res.statusCode;
          finish(err);
          return;
        }
        let body = {};
        try { body = JSON.parse(text); } catch {}
        finish(null, body);
      });
    });
    req.on("error", finish);
    pack.on("error", finish);
    gzip.on("error", finish);
    pack.pipe(gzip).pipe(req);

    (async () => {
      for (const rel of relPaths) await addFile(pack, rootDir, rel);
      pack.finalize();
    })().catch((err) => {
      pack.destroy(err);
      req.destroy(err);
    });
  });
}

module.exports = { putBundle };
