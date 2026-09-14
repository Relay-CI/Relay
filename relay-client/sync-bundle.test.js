const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const os = require("os");
const path = require("path");
const http = require("http");
const zlib = require("zlib");
const tar = require("tar-stream");
const { putBundle } = require("./sync-bundle");

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
