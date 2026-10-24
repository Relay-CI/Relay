"use strict";

// Discover the actual stdio tool list; no agent requests are made.
const fs = require("node:fs");
const path = require("node:path");
const { Client } = require("@modelcontextprotocol/sdk/client/index.js");
const { StdioClientTransport } = require("@modelcontextprotocol/sdk/client/stdio.js");

async function main() {
  const root = path.resolve(__dirname, "..");
  const check = process.argv.includes("--check");
  const client = new Client({ name: "relay-docs", version: "1.0.0" });
  const transport = new StdioClientTransport({
    command: process.execPath, args: [path.join(root, "index.js")], stderr: "pipe",
    env: { ...process.env, RELAY_SOCKET: "", RELAY_URL: "http://127.0.0.1:1", RELAY_TOKEN: "docs-discovery-only" },
  });
  try {
    await client.connect(transport);
    const { tools } = await client.listTools();
    const catalog = tools.map((tool) => ({
      name: tool.name, description: tool.description,
      readOnly: Boolean(tool.annotations?.readOnlyHint),
    }));
    const table = [
      "<!-- mcp-tools:start -->",
      `There are **${catalog.length} tools**. Lane branches default to \`main\` unless a tool states otherwise. The agent enforces account and lane permissions.`,
      "", "| Tool | Action | Description |", "| --- | --- | --- |",
      ...catalog.map((tool) => `| \`${tool.name}\` | ${tool.readOnly ? "Inspect" : "Change"} | ${tool.description.replaceAll("|", "\\|")} |`),
      "<!-- mcp-tools:end -->",
    ].join("\n");
    const readmePath = path.join(root, "README.md");
    const readme = fs.readFileSync(readmePath, "utf8").replaceAll("\r\n", "\n");
    const nextReadme = readme.includes("<!-- mcp-tools:start -->")
      ? readme.replace(/<!-- mcp-tools:start -->[\s\S]*?<!-- mcp-tools:end -->/, table)
      : readme.replace(/## Tools\n[\s\S]*?\n## Troubleshooting/, `## Tools\n\n${table}\n\n## Troubleshooting`);
    const targets = [
      [readmePath, nextReadme],
      [path.resolve(root, "../site/src/lib/mcp-tools.json"), JSON.stringify(catalog, null, 2) + "\n"],
    ];
    const stale = [];
    for (const [file, content] of targets) {
      const existing = fs.existsSync(file) ? fs.readFileSync(file, "utf8").replaceAll("\r\n", "\n") : "";
      if (existing === content) continue;
      if (check) stale.push(path.relative(root, file));
      else { fs.mkdirSync(path.dirname(file), { recursive: true }); fs.writeFileSync(file, content); }
    }
    if (stale.length) throw new Error(`MCP docs are stale: ${stale.join(", ")}. Run npm run docs:sync in relay-mcp.`);
    process.stdout.write(`${check ? "Verified" : "Synced"} documentation for ${catalog.length} MCP tools.\n`);
  } finally { await client.close(); }
}

main().catch((error) => { process.stderr.write(error.message + "\n"); process.exitCode = 1; });
