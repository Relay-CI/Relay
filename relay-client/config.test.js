const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const os = require("os");
const path = require("path");

const {
  findRelayConfig,
  saveRelayConfig,
  getServerSession,
  setServerSession,
} = require("./config");

function tempDir(t) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "relay-config-test-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  return dir;
}

test("repository config lookup does not inherit a home-level Relay server", (t) => {
  const root = tempDir(t);
  const home = path.join(root, "home");
  const repo = path.join(home, "code", "app");
  const nested = path.join(repo, "src");
  fs.mkdirSync(path.join(repo, ".git"), { recursive: true });
  fs.mkdirSync(nested, { recursive: true });
  fs.writeFileSync(
    path.join(home, ".relay.json"),
    JSON.stringify({ url: "https://old-server.example" }),
  );

  assert.equal(findRelayConfig(nested), null);

  fs.writeFileSync(
    path.join(repo, ".relay.json"),
    JSON.stringify({ url: "https://repo-server.example" }),
  );
  assert.equal(findRelayConfig(nested), path.join(repo, ".relay.json"));
});

test("saving project config never overwrites an ancestor config", (t) => {
  const root = tempDir(t);
  const parentConfig = path.join(root, ".relay.json");
  const project = path.join(root, "project");
  fs.mkdirSync(project);
  fs.writeFileSync(parentConfig, JSON.stringify({ url: "https://server-a.example" }));

  const savedPath = saveRelayConfig({ url: "https://server-b.example" }, project);

  assert.equal(savedPath, path.join(project, ".relay.json"));
  assert.equal(JSON.parse(fs.readFileSync(parentConfig, "utf8")).url, "https://server-a.example");
  assert.equal(JSON.parse(fs.readFileSync(savedPath, "utf8")).url, "https://server-b.example");
});

test("login sessions are stored independently per server URL", (t) => {
  const root = tempDir(t);
  const previousStatePath = process.env.RELAY_STATE_PATH;
  process.env.RELAY_STATE_PATH = path.join(root, "state.json");
  t.after(() => {
    if (previousStatePath === undefined) delete process.env.RELAY_STATE_PATH;
    else process.env.RELAY_STATE_PATH = previousStatePath;
  });

  setServerSession("https://server-a.example/", { token: "token-a", username: "a" });
  setServerSession("https://server-b.example", { token: "token-b", username: "b" });

  assert.equal(getServerSession("https://server-a.example").token, "token-a");
  assert.equal(getServerSession("https://server-b.example/").token, "token-b");
});

