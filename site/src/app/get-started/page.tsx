import type { Metadata } from "next";
import Link from "next/link";
import { CodePanel, DocsHero, DocsPageLayout } from "@/components/site-shell";

export const metadata: Metadata = {
  title: "Deploy your first app",
  description: "Install the Relay CLI and agent, connect an app folder to your server, deploy, and verify the result.",
  alternates: { canonical: "/get-started" },
};

export default function GetStartedPage() {
  return (
    <DocsPageLayout currentPath="/get-started" hero={<DocsHero title="Deploy your first app" summary="Install the agent on your server, connect the CLI from your app folder, and deploy. This guide uses the published CLI and agent binaries; you do not need to build Relay from source." />}>
      <section id="requirements">
        <h2>Before you begin</h2>
        <ul className="docs-requirements">
          <li><strong>Server:</strong> a machine with Docker installed and running. Your laptop can also be the server for a local trial.</li>
          <li><strong>Node.js 18 or later:</strong> required wherever you run the CLI. Installing the agent with the CLI also requires Node.js on the server.</li>
          <li><strong>An app:</strong> use an existing project with working build and start commands. Run project commands from its root folder.</li>
        </ul>
        <p>A <strong>lane</strong> is one deployment target: project + environment + branch. This guide uses <code>my-app / dev / main</code>.</p>
      </section>
      <section id="path" className="space-y-8">
        <h2>Install and deploy</h2>
        <div className="docs-step">
          <h3>1. Install and start the agent on your server</h3>
          <p>Run these commands on the machine that will host the app. The installer prints how to add the downloaded binaries to your shell&apos;s PATH.</p>
          <CodePanel label="On your server" code={`npm install -g @relay-org/relay\nrelay agent install`} />
          <p>Follow the printed PATH instructions, then start the agent:</p>
          <CodePanel label="On your server" code="relayd" footer="Leave this process running. By default, relayd listens on port 8080 and stores state in a data folder relative to its working directory." />
          <p>Open the dashboard at your server&apos;s address on port 8080. On a local trial, use <code>http://127.0.0.1:8080</code>. Complete the first-owner setup when prompted. If the server uses a configured static token, use that token to authenticate.</p>
          <details className="docs-overview"><summary>Keep the agent running on a Linux server</summary><div><p>For a persistent installation that survives SSH disconnects and reboots, use the service installer. Choose a persistent data directory before deploying apps.</p><CodePanel label="Linux server" code="sudo relayd service install --user relay --group relay --data-dir /var/lib/relayd" /><p>See <Link className="docs-text-link" href="/docs/agent#operational-shape">agent storage and operations</Link> before changing the data directory of an existing server.</p></div></details>
        </div>
        <div className="docs-step">
          <h3>2. Connect the CLI from your app folder</h3>
          <p>On your laptop, install the CLI. If you are using the same machine as the server, the CLI is already installed; open a second terminal.</p>
          <CodePanel label="On your laptop" code="npm install -g @relay-org/relay" />
          <p>Change to your app&apos;s root folder, then run:</p>
          <CodePanel label="Inside your app folder" code="relay init" />
          <ol>
            <li>Enter the server URL. Use <code>http://127.0.0.1:8080</code> for a local trial; use your server&apos;s HTTPS URL for a remote connection.</li>
            <li>Follow the authentication prompts for your server.</li>
            <li>Choose your app name, <code>dev</code> environment, and <code>main</code> branch. Choose <code>docker</code> as the engine.</li>
          </ol>
          <p>The wizard writes <code>.relay.json</code> into your project folder. Later commands reuse these connection and lane settings.</p>
          <details className="docs-overview"><summary>Connect with a token or local Unix socket</summary><div><p>For a token-based connection, replace the example URL and token with your server&apos;s values. Keep the token private.</p><CodePanel label="HTTP with a token (one line, any shell)" code="relay init --url https://relay.example.com --token YOUR_TOKEN --app my-app --env dev --branch main" /><p>On a machine where the agent exposes a Unix socket, use its absolute path. The path belongs to the agent&apos;s data directory, not your app folder. Socket access uses filesystem permissions and requires no token.</p><CodePanel label="Example: local Unix socket" code="relay init --socket /var/lib/relayd/relay.sock --app my-app --env dev --branch main" /><p>If your system does not expose a Unix socket, use HTTP.</p></div></details>
        </div>
        <div className="docs-step">
          <h3>3. Check the connection and deploy</h3>
          <p>Stay in the same app folder. First check the server connection, then upload and deploy your app.</p>
          <CodePanel label="Inside your app folder" code={`relay doctor\nrelay deploy --stream`} />
          <p>Relay detects the framework, uploads changed files, builds an image, and rolls out the container. The command streams build logs and reports the deployment result.</p>
        </div>
      </section>
      <section id="verify">
        <h2>Check your deployment</h2>
        <p className="mt-4">Inspect the lane you just deployed:</p>
        <div className="mt-4"><CodePanel label="Inside your app folder" code={`relay status\nrelay projects\n# If the deployment failed, use its ID to inspect the logs:\nrelay logs <deploy-id>`} /></div>
        <p className="mt-4">Open the app URL shown by Relay or use its dashboard link. The URL depends on the server&apos;s routing setup; port mode and managed host mode produce different addresses.</p>
        <div className="docs-callout mt-6"><h3>Dev and staging apps can require login</h3><p>These lanes default to Relay Login. A login screen does not mean the build failed. Check the lane&apos;s access policy in the dashboard and read <Link className="docs-text-link" href="/docs/previews">routing and app access</Link> before changing how the app is shared.</p></div>
        <p className="mt-4">If a check fails, start with the reported <code>relay doctor</code> issue or the build log. The <Link className="docs-text-link" href="/docs#troubleshooting">troubleshooting guide</Link> explains the next checks.</p>
      </section>
      <section id="after-first-deploy">
        <h2>Next steps</h2>
        <div className="docs-guide-list">
          <Link href="/docs#config-model"><strong>Configure your app</strong><span>Learn which file controls connections, builds, services, and uploads.</span></Link>
          <Link href="/docs/services"><strong>Add a database</strong><span>Declare companion services when your app needs persistent data.</span></Link>
          <Link href="/docs/agent"><strong>Prepare a shared server</strong><span>Review authentication, CORS, persistent state, and server settings.</span></Link>
        </div>
      </section>
    </DocsPageLayout>
  );
}
