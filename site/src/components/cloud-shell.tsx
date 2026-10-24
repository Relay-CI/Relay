import type { ReactNode } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  ArrowRight,
  Fingerprint,
  Layers3,
  Orbit,
  Shield,
} from "lucide-react";
import { GlassSurface } from "@/components/home-effects";
import { Badge } from "@/components/ui/badge";

const cloudNavigation = [
  { href: "/cloud", label: "Overview" },
  { href: "/cloud#tiers", label: "Products" },
  { href: "/cloud#interest", label: "Contact" },
];

export function CloudPageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="site-shell cloud-shell">
      <CloudHeader />
      <main className="relative z-10 mx-auto flex w-full max-w-[1500px] flex-col gap-14 px-5 py-8 sm:px-6 lg:px-10 lg:py-10">
        {children}
      </main>
      <CloudFooter />
    </div>
  );
}

function CloudHeader() {
  return (
    <header className="sticky top-0 z-50 border-b border-white/8 bg-[rgba(5,17,22,0.74)] backdrop-blur-2xl">
      <div className="mx-auto flex w-full max-w-[1500px] items-center justify-between gap-5 px-5 py-4 sm:px-6 lg:px-10">
        <div className="flex items-center gap-4">
          <Link
            href="/cloud"
            className="inline-flex items-center gap-3 text-sm font-semibold tracking-[-0.02em] text-[#f1fff9]"
          >
            <span className="inline-flex h-11 w-11 items-center justify-center rounded-2xl border border-[rgba(98,255,222,0.18)] bg-[rgba(98,255,222,0.08)] shadow-[0_12px_30px_rgba(0,0,0,0.28)]">
              <Orbit className="h-5 w-5 text-[#62ffde]" />
            </span>
            <span className="flex items-center gap-2">
              Relay Cloud
              <Badge className="rounded-full border border-[rgba(98,255,222,0.18)] bg-[rgba(98,255,222,0.1)] px-2 text-[0.62rem] uppercase tracking-[0.22em] text-[#b9fff2]">
                waitlist
              </Badge>
            </span>
          </Link>
          <div className="hidden items-center gap-2 text-xs uppercase tracking-[0.2em] text-white/32 md:flex">
            <span className="h-1 w-1 rounded-full bg-[#62ffde]" />
            operated with relay
          </div>
        </div>

        <nav className="hidden items-center gap-2 rounded-full border border-white/8 bg-white/[0.03] p-1.5 md:flex">
          {cloudNavigation.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="rounded-full px-4 py-2 text-sm font-medium text-white/64 transition-colors duration-200 hover:bg-white/[0.06] hover:text-white"
            >
              {item.label}
            </Link>
          ))}
          <Link
            href="/cloud#interest"
            className="rounded-full border border-[rgba(98,255,222,0.18)] bg-[rgba(98,255,222,0.08)] px-4 py-2 text-sm font-medium text-[#cbfff5] transition-colors duration-200 hover:bg-[rgba(98,255,222,0.14)]"
          >
            Join waitlist
          </Link>
        </nav>

        <div className="flex items-center gap-2">
          <Link
            href="/"
            className="hidden rounded-full border border-white/8 bg-white/[0.03] px-4 py-2 text-sm font-medium text-white/70 transition-colors duration-200 hover:bg-white/[0.06] hover:text-white md:inline-flex"
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            Relay main site
          </Link>
          <Link
            href="/cloud#interest"
            className="inline-flex items-center rounded-full border border-[rgba(98,255,222,0.18)] bg-[rgba(98,255,222,0.08)] px-4 py-2 text-sm font-medium text-[#cbfff5] transition-colors duration-200 hover:bg-[rgba(98,255,222,0.14)]"
          >
            Join waitlist
          </Link>
        </div>
      </div>
    </header>
  );
}

function CloudFooter() {
  return (
    <footer className="relative z-10 border-t border-white/8">
      <div className="mx-auto flex w-full max-w-[1500px] flex-col gap-6 px-5 py-8 text-sm text-white/46 sm:px-6 lg:flex-row lg:items-end lg:justify-between lg:px-10">
        <div className="space-y-2">
          <div className="inline-flex items-center gap-2 text-white/78">
            <Layers3 className="h-4 w-4 text-[#62ffde]" />
            Relay Cloud
          </div>
          <p className="max-w-xl leading-6">
            Relay Cloud is currently available by waitlist for teams evaluating
            Cloud Connect and Managed Nodes.
          </p>
        </div>
        <div className="flex flex-wrap gap-3">
          <Link href="/cloud" className="hover:text-white">
            Overview
          </Link>
          <Link href="/cloud#interest" className="hover:text-white">
            Contact
          </Link>
          <Link href="/" className="hover:text-white">
            Relay main site
          </Link>
        </div>
      </div>
    </footer>
  );
}

export function CloudHeroPanel({
  eyebrow,
  title,
  body,
  actions,
  aside,
}: {
  eyebrow: string;
  title: string;
  body: string;
  actions?: ReactNode;
  aside?: ReactNode;
}) {
  return (
    <section className="noise-surface relative overflow-hidden rounded-[2.5rem] border border-white/8 bg-[linear-gradient(145deg,rgba(7,20,25,0.96),rgba(4,18,27,0.92))] p-6 shadow-[0_32px_130px_rgba(0,0,0,0.34)] sm:p-8 lg:p-10">
      <div className="relative z-10 grid gap-8 lg:grid-cols-[minmax(0,1.1fr)_23rem] lg:items-end">
        <div className="space-y-6">
          <div className="inline-flex items-center gap-2 rounded-full border border-[rgba(98,255,222,0.18)] bg-[rgba(98,255,222,0.08)] px-3.5 py-1.5 text-[0.68rem] uppercase tracking-[0.24em] text-[#c8fff6]">
            <Fingerprint className="h-3.5 w-3.5" />
            {eyebrow}
          </div>
          <div className="max-w-4xl space-y-4">
            <h1 className="text-balance text-4xl font-semibold tracking-[0.02em] text-white sm:text-5xl lg:text-6xl">
              {title}
            </h1>
            <p className="max-w-2xl text-lg leading-8 text-white/62">{body}</p>
          </div>
          {actions ? <div className="flex flex-wrap gap-3">{actions}</div> : null}
        </div>
        {aside ? (
          <GlassSurface className="rounded-[2rem] border border-white/8 p-5">
            {aside}
          </GlassSurface>
        ) : null}
      </div>
    </section>
  );
}

export function CloudSectionHeading({
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
      <div className="inline-flex rounded-full border border-[rgba(98,255,222,0.16)] bg-[rgba(98,255,222,0.08)] px-3.5 py-1.5 text-[0.68rem] uppercase tracking-[0.24em] text-[#c8fff6]">
        {label}
      </div>
      <h2 className="text-balance text-3xl font-semibold tracking-[-0.01em] text-white sm:text-4xl">
        {title}
      </h2>
      {body ? <p className="text-base leading-7 text-white/58 sm:text-lg">{body}</p> : null}
    </div>
  );
}

export function CloudCtaLink({
  href,
  children,
}: {
  href: string;
  children: ReactNode;
}) {
  return (
    <Link
      href={href}
      className="inline-flex items-center gap-2 rounded-full border border-[rgba(98,255,222,0.18)] bg-[rgba(98,255,222,0.08)] px-4 py-2 text-sm font-medium text-[#cbfff5] transition-colors duration-200 hover:bg-[rgba(98,255,222,0.14)]"
    >
      {children}
      <ArrowRight className="h-4 w-4" />
    </Link>
  );
}

export function CloudMiniStat({
  label,
  value,
  body,
}: {
  label: string;
  value: string;
  body: string;
}) {
  return (
    <div className="rounded-[1.5rem] border border-white/8 bg-white/[0.03] p-4">
      <div className="text-[0.68rem] uppercase tracking-[0.22em] text-white/34">
        {label}
      </div>
      <div className="mt-3 text-3xl font-semibold tracking-[-0.02em] text-white">
        {value}
      </div>
      <div className="mt-2 text-sm leading-6 text-white/46">{body}</div>
    </div>
  );
}

export function CloudCallout({
  icon: Icon,
  title,
  body,
}: {
  icon: typeof Shield;
  title: string;
  body: string;
}) {
  return (
    <GlassSurface className="rounded-[1.7rem] border border-white/8 p-5">
      <div className="flex items-start gap-4">
        <div className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-[rgba(98,255,222,0.14)] bg-[rgba(98,255,222,0.08)] text-[#c8fff6]">
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <div className="text-lg font-semibold tracking-[0.02em] text-white">
            {title}
          </div>
          <p className="mt-2 text-sm leading-6 text-white/56">{body}</p>
        </div>
      </div>
    </GlassSurface>
  );
}
