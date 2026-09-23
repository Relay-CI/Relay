const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const os = require("os");
const path = require("path");
const http = require("http");
const zlib = require("zlib");
const tar = require("tar-stream");
const { putBundle, putBundleBatched, planBundleBatches } = require("./sync-bundle");

test("putBundle sends many changed files in one streaming request", async (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "relay-bundle-test-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const paths = [];
  for (let i = 0; i < 100; i++) {
    const rel = `src/file-${i}.txt`;
    const abs = path.join(root, rel);
    fs.mkdirSync(path.dirname(abs), { recursive: true });
    fs.writeFileSync(abs, `payload-${i}`);
    paths.push(rel);
  }

  let requests = 0;
  const received = new Map();
  const server = http.createServer((req, res) => {
    requests++;
    assert.equal(req.headers["content-encoding"], "gzip");
    const extract = tar.extract();
    extract.on("entry", (header, stream, next) => {
      const chunks = [];
      stream.on("data", (chunk) => chunks.push(chunk));
      stream.on("end", () => {
        received.set(header.name, Buffer.concat(chunks).toString("utf8"));
        next();
      });
      stream.resume();
    });
    extract.on("finish", () => {
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify({ ok: true, files: received.size }));
    });
    req.pipe(zlib.createGunzip()).pipe(extract);
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  t.after(() => server.close());
  const { port } = server.address();

  const result = await putBundle(
    { kind: "http", baseUrl: `http://127.0.0.1:${port}`, token: "test" },
    "/api/sync/bundle/session",
    root,
    paths,
  );

  assert.equal(requests, 1);
  assert.equal(result.files, 100);
  assert.equal(received.get("src/file-73.txt"), "payload-73");
});

test("planBundleBatches splits a diff once cumulative size exceeds the cap", async (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "relay-bundle-plan-test-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const paths = [];
  for (let i = 0; i < 5; i++) {
    const rel = `file-${i}.bin`;
    fs.writeFileSync(path.join(root, rel), Buffer.alloc(40));
    paths.push(rel);
  }

  // Cap of 100 bytes with 5 x 40-byte files: 2 fit per batch (80 <= 100,
  // + a 3rd would be 120 > 100), so batches should be [2, 2, 1].
  const batches = await planBundleBatches(root, paths, 100);
  assert.deepEqual(
    batches.map((b) => b.length),
    [2, 2, 1],
  );
  assert.deepEqual(batches.flat(), paths);
});

test("planBundleBatches gives an oversized single file its own batch", async (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "relay-bundle-plan-oversized-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  fs.writeFileSync(path.join(root, "small.txt"), Buffer.alloc(10));
  fs.writeFileSync(path.join(root, "huge.bin"), Buffer.alloc(200));
  fs.writeFileSync(path.join(root, "small2.txt"), Buffer.alloc(10));

  const batches = await planBundleBatches(root, ["small.txt", "huge.bin", "small2.txt"], 100);
  // huge.bin alone exceeds the cap but must still get a batch rather than
  // being dropped or blocking the others from batching together.
  assert.deepEqual(batches, [["small.txt"], ["huge.bin"], ["small2.txt"]]);
});

test("putBundleBatched sends a large diff as multiple capped requests to the same session", async (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "relay-bundle-batched-test-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const paths = [];
  for (let i = 0; i < 10; i++) {
    const rel = `asset-${i}.bin`;
    fs.writeFileSync(path.join(root, rel), Buffer.alloc(1024, i));
    paths.push(rel);
  }

  let requests = 0;
  const receivedPaths = [];
  const server = http.createServer((req, res) => {
    requests++;
    const extract = tar.extract();
    let count = 0;
    extract.on("entry", (header, stream, next) => {
      receivedPaths.push(header.name);
      count++;
      stream.on("data", () => {});
      stream.on("end", next);
      stream.resume();
    });
    extract.on("finish", () => {
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify({ ok: true, files: count, bytes: count * 1024 }));
    });
    req.pipe(zlib.createGunzip()).pipe(extract);
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  t.after(() => server.close());
  const { port } = server.address();

  // 10 x 1024-byte files with a 3000-byte cap: at most 2 files per request
  // (3072 > 3000), so this must take more than one request.
  const result = await putBundleBatched(
    { kind: "http", baseUrl: `http://127.0.0.1:${port}`, token: "test" },
    "/api/sync/bundle/session",
    root,
    paths,
    { maxBytes: 3000 },
  );

  assert.ok(requests > 1, `expected multiple requests, got ${requests}`);
  assert.equal(result.batches, requests);
  assert.equal(result.files, 10);
  assert.deepEqual(receivedPaths.sort(), [...paths].sort());
});
