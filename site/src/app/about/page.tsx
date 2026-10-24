import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { PageHero, SitePage } from "@/components/site-shell";

export const metadata: Metadata = {
  title: "About Relay",
  description: "Relay is an open-source tool for deploying apps on your own server.",
  alternates: { canonical: "/about" },
};

export default function AboutPage() {
  return (
    <SitePage>
      <PageHero title="Deploy on your own server." summary="Relay is an open-source deployment tool. It builds and runs your apps on a machine you control, with a CLI for deployments and a dashboard for everything that’s running." actions={<Link className="site-button" href="/get-started">Get started <ArrowRight size={15} aria-hidden="true" /></Link>} />
      <section className="editorial-section"><h2>What Relay does</h2><div><p>Install the agent on your server and the CLI on your computer. Run <code>relay deploy</code> from your app folder. Relay uploads the changed files, builds a container image, and deploys it.</p><p>The dashboard shows deployment history, logs, app settings, and services. You can restart an app, inspect a failed build, or roll back to a previous image.</p><p>When your app needs a database, declare it in <code>relay.json</code>. Relay starts the service on the same private network and supplies its connection details to the app.</p></div></section>
      <section className="editorial-section"><h2>Where your apps run</h2><div><p>Apps run on your server. Docker is the default runtime, and the agent keeps deployment state and history in its data directory.</p><p>Development, staging, preview, and production can use separate lanes. Each lane has its own route, access policy, and deployment history.</p><p>You manage the host, storage, and backups. Relay handles builds and deployments; it does not replace server maintenance.</p></div></section>
      <section className="editorial-section"><h2>Work on Relay</h2><div><p>The source is available on GitHub. You can report a bug, suggest a change, or contribute code and documentation.</p><Link className="graphite-text-link" href="https://github.com/Relay-CI/Relay">View the repository <ArrowRight size={15} aria-hidden="true" /></Link></div></section>
    </SitePage>
  );
}
