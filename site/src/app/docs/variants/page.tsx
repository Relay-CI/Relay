import type { Metadata } from "next";
import Link from "next/link";
import { CodePanel, DocsHero, DocsPageLayout } from "@/components/site-shell";

export const metadata: Metadata = {
  title: "Variants and visitor feedback",
  description: "Deploy a variant beside a running release, assign traffic, compare metrics and feedback, and promote or remove it.",
  alternates: { canonical: "/docs/variants" },
};

export default function VariantsPage() {
  return (
    <DocsPageLayout currentPath="/docs/variants" hero={<DocsHero title="Test a variant with real visitors" summary="A variant runs another build beside the normal release in the same lane. Give it a share of new visitors, compare the results, then promote it or remove it." />}>
      <section id="requirements">
        <h2>Before you start</h2>
        <ul className="docs-requirements">
          <li>Deploy the lane at least once. Its normal release, called <strong>control</strong>, must be running.</li>
          <li>Use Docker and <code>traffic_mode: edge</code>. Variants do not support Station or session traffic mode.</li>
          <li>Use an account with deployer access to that lane to change variants. Viewer access is enough to read the report.</li>
          <li>Allow room on the server for extra containers. Each variant starts with the lane&apos;s replica count; databases remain shared within the lane.</li>
        </ul>
        <p>Use a separate <Link className="docs-text-link" href="/docs/previews">lane</Link> when you need an isolated environment. A variant shares the lane&apos;s URL, services, and configuration, so its code must remain compatible with those services and any shared database schema.</p>
      </section>
      <section id="deploy">
        <h2>Deploy a variant</h2>
        <p>Run these commands in your app folder, using your project&apos;s connection settings. The example names the variant <code>checkout</code> and gives it 10% of new visitors.</p>
        <CodePanel label="Inside your app folder" code={`relay deploy --app my-app --env prod --branch main --variant checkout --weight 10
relay variants list --app my-app --env prod --branch main --period 24h`} />
        <p>Change the source first, then deploy with the same variant name to update that variant. Relay waits for the replacement containers before switching their route. A normal deployment without <code>--variant</code> updates control and leaves variants in place.</p>
        <p>Names are 1–24 lowercase letters, digits, or hyphens and must start with a letter or digit. Names such as <code>control</code>, <code>blue</code>, and <code>green</code> are reserved. A lane can have at most eight variants.</p>
      </section>
      <section id="traffic">
        <h2>Choose who sees it</h2>
        <p>Weights apply to <strong>new assignments</strong>. A visitor&apos;s first requests are assigned consistently using their IP address and user agent, then a cookie keeps them on the same version for up to 30 days.</p>
        <CodePanel label="Use the same app, environment, and branch flags as above" code={`relay variants weight checkout 25 --app my-app --env prod --branch main
relay variants reshuffle --app my-app --env prod --branch main`} />
        <p>All running variant weights together must be at most 100%. Control gets the remaining share. Setting a weight to zero stops new automatic assignments; visitors already assigned to that variant can keep seeing it. Reshuffle resets assignments for everyone under the current weights.</p>
        <p>To preview a version yourself, add <code>?__relay_variant=checkout</code> to the app URL. Use <code>?__relay_variant=control</code> to preview control. This also updates your assignment cookie. The response header <code>X-Relay-Variant</code> shows the selected version.</p>
      </section>
      <section id="metrics">
        <h2>Compare the results</h2>
        <p>Open <strong>Variants</strong> in the dashboard after selecting the project and lane. Choose 24 hours, 7 days, or 30 days to compare visitors, page views, requests, server errors, p50/p95 latency, and feedback. The same report is available through <code>relay variants list --period 7d</code> and the MCP tool <code>list_variants</code>.</p>
        <p>Metrics come from traffic through Relay&apos;s edge proxy. Requests sent directly to an app container are not included. Reports describe observed traffic and feedback; Relay does not calculate statistical significance or choose a winner for you.</p>
      </section>
      <section id="feedback">
        <h2>Collect visitor feedback</h2>
        <p>Your app can submit feedback to its own <code>/__relay/feedback</code> endpoint. Relay&apos;s edge proxy supplies the lane and assigned variant. You do not put an agent token in the browser or send a variant name yourself.</p>
        <CodePanel label="Example browser code in your app" code={`const response = await fetch("/__relay/feedback", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    vote: "up",
    rating: 5,
    comment: "Checkout was easy to use",
    page: location.pathname
  })
});
if (!response.ok) throw new Error("Feedback could not be saved");`} />
        <p>Send at least a vote (<code>up</code> or <code>down</code>), a rating from 1 to 5, or a comment. Comments are limited to 2,000 characters; page values to 256. The endpoint limits submissions to ten per minute for a lane and client IP.</p>
        <p>Read the latest 50 entries for the selected period in the dashboard, in <code>list_variants</code>, or with <code>relay variants feedback --period 7d</code>. Avoid collecting secrets or personal details in free-text feedback.</p>
      </section>
      <section id="finish">
        <h2>Promote or remove a variant</h2>
        <CodePanel label="Use one action after reviewing the results" code={`# Make this image the lane's normal release
relay variants promote checkout --app my-app --env prod --branch main

# Or stop the experiment
relay variants rm checkout --app my-app --env prod --branch main`} />
        <p>Promote queues a deployment of the existing image and returns its deployment ID. The variant stays in place until that deployment succeeds, then Relay removes it. If the deployment fails, the variant remains available.</p>
        <p>Removing a variant removes its route before stopping its containers. Its visitors fall back to control. Variant promotion changes control within one lane; it is separate from <Link className="docs-text-link" href="/docs/previews#lane-lifecycle">promotion between staging and production</Link>.</p>
        <p>MCP provides <code>set_variant_weight</code>, <code>promote_variant</code>, <code>remove_variant</code>, and <code>reshuffle_variants</code>. Creating or rebuilding a variant from local source still uses <code>relay deploy</code>.</p>
      </section>
      <section id="api">
        <h2>HTTP API</h2>
        <p>Read <code>GET /api/variants?app=my-app&amp;env=prod&amp;branch=main&amp;period=7d</code>. The response includes <code>control</code>, <code>variants</code>, <code>feedback</code>, <code>supported</code>, and an <code>unsupported</code> reason when the lane cannot run variants.</p>
        <CodePanel label="POST /api/variants — example weight change" code={`{
  "app": "my-app",
  "env": "prod",
  "branch": "main",
  "action": "weight",
  "name": "checkout",
  "weight": 25
}`} />
        <p>Other actions are <code>promote</code> and <code>remove</code>, which take a name, and <code>reshuffle</code>, which does not. These routes are available through both HTTP and the local agent socket.</p>
      </section>
    </DocsPageLayout>
  );
}
