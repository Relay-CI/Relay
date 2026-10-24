import type { Metadata } from "next";
import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { PageHero, SitePage } from "@/components/site-shell";
import { siteEnv } from "@/lib/env";

export const metadata: Metadata = { title: "Contact", description: "Report a Relay bug, ask a question, or contribute on GitHub.", alternates: { canonical: "/contact" } };
export default function ContactPage() {
  return (
    <SitePage>
      <PageHero title="Get in touch." summary="Use GitHub to report a bug, ask a question, or suggest a feature." />
      <section className="contact-links" aria-label="Contact options">
        <a href="https://github.com/Relay-CI/Relay/issues"><div><h2>Report a bug or request a feature</h2><p>Check existing issues first, then open a new one.</p></div><ArrowUpRight size={19} aria-hidden="true" /></a>
        <a href="https://github.com/Relay-CI/Relay"><div><h2>Contribute to Relay</h2><p>Read the source and contribution guide on GitHub.</p></div><ArrowUpRight size={19} aria-hidden="true" /></a>
        {siteEnv.contactEmail && <a href={`mailto:${siteEnv.contactEmail}`}><div><h2>Email</h2><p>{siteEnv.contactEmail}</p></div><ArrowUpRight size={19} aria-hidden="true" /></a>}
      </section>
      <section className="editorial-section"><h2>Reporting a deployment problem</h2><div><p>Include the Relay version, operating system, failed command, and relevant build log. The app, environment, and branch help identify the affected lane.</p><p>Remove tokens, passwords, and connection strings before posting logs.</p><p>If you are still setting up Relay, try <code>relay doctor</code> and the <Link className="docs-text-link" href="/get-started">setup guide</Link> first.</p></div></section>
    </SitePage>
  );
}
