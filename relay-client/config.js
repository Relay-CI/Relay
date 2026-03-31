const fs = require("fs");
const path = require("path");

function findRelayConfig(startDir = process.cwd()) {
  let dir = startDir;

  while (true) {
    const p1 = path.join(dir, ".relay", "config.json");
    const p2 = path.join(dir, ".relay.json");
    const p3 = path.join(dir, "relay.config.json");

    if (fs.existsSync(p1)) return p1;
    if (fs.existsSync(p2)) return p2;
    if (fs.existsSync(p3)) return p3;

    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  return null;
}

function loadRelayConfig(startDir) {
  const p = findRelayConfig(startDir);
  if (!p) return { path: null, data: {} };

  const raw = fs.readFileSync(p, "utf8");
  let data = {};
  try {
    data = JSON.parse(raw);
  } catch (e) {
    throw new Error(`Invalid JSON in ${p}`);
  }
  return { path: p, data };
}

module.exports = { findRelayConfig, loadRelayConfig };
