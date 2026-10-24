"use client";

import { useState } from "react";
import { CodePanel } from "@/components/site-shell";

export function McpSetup() {
  const [client, setClient] = useState("cursor");
  const [connection, setConnection] = useState("http");
  const [platform, setPlatform] = useState("unix");
  const windows = platform === "windows";
  const server = {
    ...(client === "vscode" ? { type: "stdio" } : {}),
    command: windows ? "cmd" : "npx",
    args: windows ? ["/c", "npx", "-y", "@relay-org/relay-mcp"] : ["-y", "@relay-org/relay-mcp"],
    env: connection === "socket"
      ? { RELAY_SOCKET: "/absolute/path/to/relay.sock" }
      : { RELAY_URL: "https://relay.example.com", RELAY_TOKEN: "YOUR_RELAY_TOKEN" },
  };
  const config = { [client === "vscode" ? "servers" : "mcpServers"]: { relay: server } };
  const file = client === "vscode" ? ".vscode/mcp.json" : client === "cursor" ? "~/.cursor/mcp.json" : "claude_desktop_config.json";

  return (
    <div className="mcp-setup">
      <div className="mcp-setup-options">
        <label>AI client<select value={client} onChange={(event) => setClient(event.target.value)}>
          <option value="cursor">Cursor</option><option value="claude">Claude Desktop</option><option value="vscode">VS Code</option>
        </select></label>
        <label>Client computer<select value={platform} onChange={(event) => setPlatform(event.target.value)}>
          <option value="unix">macOS or Linux</option><option value="windows">Windows</option>
        </select></label>
        <label>Agent connection<select value={connection} onChange={(event) => setConnection(event.target.value)}>
          <option value="http">HTTP / HTTPS with token</option><option value="socket">Local Unix socket</option>
        </select></label>
      </div>
      <p>Merge this entry into <code>{file}</code>, keeping any existing servers.</p>
      <CodePanel label={file} code={JSON.stringify(config, null, 2)} />
      <p>{connection === "socket" ? "Replace the socket path with the agent’s absolute socket path. The MCP process must run on the same machine with permission to access it. Use HTTP if your system does not expose a Unix socket." : "Replace the example URL and token with your agent’s values. For a local HTTP connection, use http://127.0.0.1:8080. Keep tokens in your local client configuration and out of source control."}</p>
      {windows && <p>The Windows launcher uses <code>cmd /c npx</code>. Node.js must be available to the desktop app, not only in a terminal.</p>}
      <p>In Claude Desktop, open the configuration file through its developer settings. In VS Code, use the workspace file above. Restart or reconnect the MCP server after saving.</p>
    </div>
  );
}
