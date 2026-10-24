import type { ReactNode } from "react";
import "@/app/docs/docs.css";
import Link from "next/link";
import {
  ArrowRight,
  ArrowUpRight,
  BookOpenText,
  Command,
  Github,
  Layers3,
  MenuIcon,
} from "lucide-react";
import { GlassSurface, HoloCard, Reveal, SpotlightCard } from "@/components/home-effects";
import { docsPages, getDocsPage } from "@/components/docs-config";
import { DocsSidebar } from "@/components/docs-sidebar";
import { CopyCodeButton } from "@/components/copy-code-button";
import {
  Sheet,
  SheetContent,
  SheetClose,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";

export const navigation = [
  { href: "/features", label: "Features" },
  { href: "/cloud", label: "Cloud" },
  { href: "/docs", label: "Docs" },
  { href: "/about", label: "About" },
  { href: "/contact", label: "Contact" },
  { href: "/get-started", label: "Get started" },
];

export function SiteHeader() {
  return (
    <header className="site-header">
      <div className="site-header-inner">
        <Link href="/" className="site-brand" aria-label="Relay home">
          <svg width="28" height="28" viewBox="0 0 100 100" fill="none" aria-hidden="true">
            <path d="M32 24H65Q80 24 80 38Q80 54 65 58L52 62M52 62L72 86M32 24Q18 30 22 46Q26 60 52 62" stroke="currentColor" strokeWidth="10" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          <span>Relay</span>
        </Link>
        <nav className="site-desktop-nav" aria-label="Main navigation">
          {navigation.filter((item) => item.href !== "/get-started").map((item) => <Link key={item.href} href={item.href}>{item.label}</Link>)}
        </nav>
        <div className="site-header-actions">
          <a className="site-github-link" href="https://github.com/Relay-CI/Relay" aria-label="Relay on GitHub"><Github size={19} aria-hidden="true" /></a>
          <Link className="site-button site-header-cta" href="/get-started">Get started <ArrowRight size={15} aria-hidden="true" /></Link>
          <Sheet>
            <SheetTrigger asChild><button type="button" className="site-mobile-trigger" aria-label="Open navigation"><MenuIcon size={22} aria-hidden="true" /></button></SheetTrigger>
            <SheetContent className="site-mobile-panel">
              <SheetHeader><SheetTitle>Relay</SheetTitle><SheetDescription>Deploy and operate apps on your own server.</SheetDescription></SheetHeader>
              <nav aria-label="Mobile navigation">
                {navigation.map((item) => <SheetClose key={item.href} asChild><Link href={item.href}>{item.label}</Link></SheetClose>)}
              </nav>
              <a href="https://github.com/Relay-CI/Relay" className="site-mobile-github"><Github size={18} aria-hidden="true" /> View on GitHub</a>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  );
}

export function SiteFooter() {
  return (
    <footer className="site-footer">
      <div><Link href="/" className="site-brand">Relay</Link><p>Open-source deployments for your own server.</p></div>
      <nav aria-label="Footer navigation">
        <Link href="/docs">Documentation</Link><Link href="/docs/mcp">MCP</Link><Link href="/about">About</Link><Link href="/contact">Contact</Link><a href="https://github.com/Relay-CI/Relay">GitHub</a>
      </nav>
    </footer>
  );
}

export function PageHero({ title, summary, actions, aside }: {
  eyebrow?: string;
  title: string;
  summary: string;
  actions?: ReactNode;
  aside?: ReactNode;
  align?: "start" | "end";
}) {
  return (
    <section className="page-intro">
      <div><h1>{title}</h1><p>{summary}</p>{actions && <div className="page-intro-actions">{actions}</div>}</div>
      {aside && <div className="page-intro-aside">{aside}</div>}
    </section>
  );
}

export function SitePage({ children }: { children: ReactNode }) {
  return (
    <div className="site-shell">
      <div className="site-announcement"><Link href="/docs/mcp">Relay + MCP <span>Connect Relay to your editor.</span><ArrowRight size={13} aria-hidden="true" /></Link></div>
      <SiteHeader />
      <main className="site-layout">
        {children}
      </main>
      <SiteFooter />
    </div>
  );
}

export function DocsPageLayout({
  currentPath,
  hero,
  children,
}: {
  currentPath: string;
  hero: ReactNode;
  children: ReactNode;
}) {
  const page = getDocsPage(currentPath);
  return (
    <div className="site-shell docs-shell min-h-screen">
      <a className="docs-skip-link" href="#docs-content">Skip to content</a>
      <SiteHeader />
      <main className="docs-layout">
        <DocsSidebar key={currentPath} pages={docsPages} currentPath={currentPath} />
        <article id="docs-content" tabIndex={-1} className="docs-content">
          <nav aria-label="Breadcrumb" className="docs-breadcrumb">
            <Link href="/docs">Documentation</Link>
            {currentPath !== "/docs" && <><span aria-hidden="true">/</span><span aria-current="page">{page.label}</span></>}
          </nav>
          {hero}
          <div className="docs-sections">{children}</div>
          {currentPath !== "/docs" && <footer className="docs-guide-footer">
            <Link href="/docs">All documentation</Link>
            <Link href="/get-started">First deployment guide <ArrowRight size={16} aria-hidden="true" /></Link>
          </footer>}
        </article>
      </main>
      <SiteFooter />
    </div>
  );
}

/** A compact reading header for documentation; marketing pages keep PageHero. */
export function DocsHero({ title, summary, actions, aside }: {
  eyebrow?: string;
  title: string;
  summary: string;
  actions?: ReactNode;
  aside?: ReactNode;
  align?: "start" | "end";
}) {
  return (
    <header className="docs-hero">
      <h1>{title}</h1>
      <p>{summary}</p>
      {actions && <div className="docs-hero-actions">{actions}</div>}
      {aside && <details className="docs-overview"><summary>At a glance</summary><div>{aside}</div></details>}
    </header>
  );
}

export function SectionHeading({
  label,
  title,
  body,
}: {
  label: string;
  title: string;
  body?: string;
}) {
  return (
    <div className="max-w-3xl space-y-3">
      <div className="text-xs font-medium uppercase tracking-widest text-[color:var(--muted-strong)]">
        {label}
      </div>
      <h2 className="text-balance text-3xl font-semibold tracking-[-0.01em] text-[color:var(--ink)] sm:text-4xl">
        {title}
      </h2>
      {body ? (
        <p className="text-base leading-7 text-[color:var(--muted)] sm:text-lg">
          {body}
        </p>
      ) : null}
    </div>
  );
}

export function CodePanel({
  label,
  code,
  footer,
}: {
  label: string;
  code: string;
  footer?: string;
}) {
  return (
    <GlassSurface className="code-panel scan-line-card rounded-xl p-5">
      <div className="mb-4 flex items-center justify-between">
        <div className="site-code-label inline-flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.24em]">
          <Command className="h-3.5 w-3.5" />
          {label}
        </div>
        <CopyCodeButton code={code} />
      </div>
      <pre tabIndex={0} aria-label={label} className="site-code-copy overflow-x-auto whitespace-pre font-mono text-[0.8rem] leading-7">
        <code>{code}</code>
      </pre>
      {footer ? (
        <div className="site-code-footer mt-4 border-t border-[color:var(--line)] pt-4 text-xs leading-6">
          {footer}
        </div>
      ) : null}
    </GlassSurface>
  );
}

export function InlineLink({
  href,
  children,
}: {
  href: string;
  children: ReactNode;
}) {
  return (
    <Link
      href={href}
      className="inline-flex items-center gap-2 text-sm font-semibold text-[color:var(--ink)] transition-colors duration-200 hover:text-[color:var(--accent)]"
    >
      {children}
      <ArrowUpRight className="h-4 w-4" />
    </Link>
  );
}

export function ResourceStrip() {
  return (
    <div className="grid gap-4 md:grid-cols-3">
      {[
        {
          icon: <BookOpenText className="h-4 w-4" />,
          title: "Readable docs",
          body: "Guides stay focused on setup, deploys, integrations, and runtime behavior.",
        },
        {
          icon: <Command className="h-4 w-4" />,
          title: "Command-first",
          body: "Every page still prioritizes actual commands and deploy behavior.",
        },
        {
          icon: <Layers3 className="h-4 w-4" />,
          title: "Self-hosted clarity",
          body: "Relay keeps deploy lanes and rollouts fast without hiding Docker or host-level behavior.",
        },
      ].map((item, index) => (
        <Reveal key={item.title} delay={index * 80}>
          <SpotlightCard className="h-full rounded-[1.6rem]">
            <HoloCard className="h-full rounded-[1.6rem]">
              <GlassSurface className="h-full rounded-[1.6rem] p-5">
                <div className="space-y-3">
                  <div className="inline-flex h-10 w-10 items-center justify-center rounded-2xl border border-[color:var(--line)] bg-[rgba(255,255,255,0.04)] text-[color:var(--accent)]">
                    {item.icon}
                  </div>
                  <div className="space-y-1">
                    <h3 className="text-lg font-semibold tracking-[0.02em] text-[color:var(--ink)]">
                      {item.title}
                    </h3>
                    <p className="text-sm leading-6 text-[color:var(--muted)]">{item.body}</p>
                  </div>
                </div>
              </GlassSurface>
            </HoloCard>
          </SpotlightCard>
        </Reveal>
      ))}
    </div>
  );
}
