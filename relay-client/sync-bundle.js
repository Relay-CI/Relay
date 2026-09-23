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

// Default cap on bytes per bundle request. Cloudflare's free/pro edge holds
// a request open for at most 100s waiting on the origin; a big diff sent as
// one request is upload-bandwidth bound, and on a slow link that single PUT
// can blow past 100s and come back as an opaque Cloudflare 524 with no
// origin-side error to debug. Splitting into capped batches means no single
// request's transfer time depends on total diff size — only on the slowest
// individual file, which still gets its own request even if it's larger
// than the cap (can't split one tar entry across requests).
const DEFAULT_MAX_BUNDLE_BYTES = 8 * 1024 * 1024;

// Groups relPaths into batches whose stat'd sizes sum to at most maxBytes
// each (an oversized single file still gets its own solo batch — batching
// can't shrink one file, only avoid piling many onto the same request).
async function planBundleBatches(rootDir, relPaths, maxBytes) {
  const batches = [];
  let current = [];
  let currentBytes = 0;
  for (const rel of relPaths) {
    const abs = path.join(rootDir, rel.split("/").join(path.sep));
    let size = 0;
    try {
      size = (await fsp.stat(abs)).size;
    } catch {
      // File may have been removed/replaced since the diff was planned;
      // let putBundle's own addFile surface that error at upload time.
    }
    if (current.length && currentBytes + size > maxBytes) {
      batches.push(current);
      current = [];
      currentBytes = 0;
    }
    current.push(rel);
    currentBytes += size;
  }
  if (current.length) batches.push(current);
  return batches;
}

// Uploads relPaths as one or more gzip tar bundles, none larger than
// maxBytes, sequentially against the same sync session. Returns combined
// {files, bytes} like putBundle. A failure on any batch (including the
// "old server doesn't support bundles" case callers detect via err.status)
// propagates immediately — files already written by prior batches stay
// staged, since the server extracts each batch into the same session
// directory and a caller falling back to per-file upload for the whole set
// just re-writes the same paths harmlessly.
async function putBundleBatched(transport, apiPath, rootDir, relPaths, opts) {
  const maxBytes = (opts && opts.maxBytes) || DEFAULT_MAX_BUNDLE_BYTES;
  const batches = await planBundleBatches(rootDir, relPaths, maxBytes);
  let files = 0;
  let bytes = 0;
  for (const batch of batches) {
    const result = await putBundle(transport, apiPath, rootDir, batch);
    files += (result && result.files) || 0;
    bytes += (result && result.bytes) || 0;
  }
  return { ok: true, files, bytes, batches: batches.length };
}

module.exports = { putBundle, putBundleBatched, planBundleBatches, DEFAULT_MAX_BUNDLE_BYTES };
