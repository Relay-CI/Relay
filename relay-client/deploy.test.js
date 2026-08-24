const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const os = require("os");
const path = require("path");

const { setServerSession } = require("./config");
const { resolveTransport } = require("./deploy");

function tempDir(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "relay-deploy-test-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  return dir;
}

test("--dir selects that project's server even when cwd has another config", (t) => {
  const root = tempDir(t);
  const projectA = path.join(root, "a");
  const projectB = path.join(root, "b");
  fs.mkdirSync(projectA);
  fs.mkdirSync(projectB);
  fs.writeFileSync(path.join(projectA, ".relay.json"), JSON.stringify({ url: "https://server-a.example", token: "token-a" }));
  fs.writeFileSync(path.join(projectB, ".relay.json"), JSON.stringify({ url: "https://server-b.example", token: "token-b" }));

  const previousCwd = process.cwd();
  process.chdir(projectA);
  let transport;
  try {
    transport = resolveTransport({ dir: projectB });
  } finally {
    process.chdir(previousCwd);
  }
  assert.equal(transport.baseUrl, "https://server-b.example");
  assert.equal(transport.token, "token-b");
});

test("project URL resolves the login token saved for that server", (t) => {
  const root = tempDir(t);
  const project = path.join(root, "project");
  fs.mkdirSync(project);
  fs.writeFileSync(path.join(project, ".relay.json"), JSON.stringify({ url: "https://server-b.example" }));

  const previousStatePath = process.env.RELAY_STATE_PATH;
  const previousURL = process.env.RELAY_URL;
  const previousToken = process.env.RELAY_TOKEN;
  process.env.RELAY_STATE_PATH = path.join(root, "state.json");
  process.env.RELAY_URL = "https://old-server.example";
  process.env.RELAY_TOKEN = "old-global-token";
  t.after(() => {
    for (const [key, value] of [["RELAY_STATE_PATH", previousStatePath], ["RELAY_URL", previousURL], ["RELAY_TOKEN", previousToken]]) {
      if (value === undefined) delete process.env[key];
      else process.env[key] = value;
    }
  });
  setServerSession("https://server-b.example", { token: "token-b" });

  const transport = resolveTransport({ dir: project });
  assert.equal(transport.baseUrl, "https://server-b.example");
  assert.equal(transport.token, "token-b");
});
