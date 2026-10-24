"use client";

import { useId, useRef, useState } from "react";
import { ArrowUpRight, Box, Check, ChevronRight, Database, GitBranch, Layers3, LockKeyhole, Terminal } from "lucide-react";
import Link from "next/link";

const tabs = [
  { name: "Deployments", icon: Terminal, href: "/docs/cli" },
  { name: "Companions", icon: Database, href: "/docs/services" },
  { name: "Lane access", icon: LockKeyhole, href: "/docs/previews" },
];

export function LanePreview() {
  const [active, setActive] = useState(0);
  const id = useId();
  const buttons = useRef<(HTMLButtonElement | null)[]>([]);

  return (
    <div className="lane-preview">
      <div className="preview-toolbar"><div><Layers3 size={15} aria-hidden="true" /><span>Relay workspace</span><ChevronRight size={13} aria-hidden="true" /><strong>my-app</strong></div><span className="preview-example">Interactive example</span></div>
      <div className="preview-body">
        <aside className="preview-project" aria-label="Example lane context">
          <div className="preview-project-name"><span>R</span> my-app</div>
          <span className="preview-sidebar-label">Environments</span>
          <div className="preview-environment">prod <span>main</span></div>
          <div className="preview-environment is-selected">dev <span>main</span></div>
          <div className="preview-environment">preview <span>feature/ui</span></div>
          <div className="preview-runtime"><Box size={14} aria-hidden="true" /> Docker runtime</div>
        </aside>
        <div className="preview-panel" role="tabpanel" id={`${id}-panel`} aria-labelledby={`${id}-tab-${active}`} tabIndex={0}>
          <div className="preview-panel-heading"><div><div className="preview-lane-name">dev <span>/</span> main</div><h3>{tabs[active].name}</h3></div><span className="preview-status"><Check size={12} aria-hidden="true" /> {active === 2 ? "Protected" : "Ready"}</span></div>
          {active === 0 && <>
            <div className="preview-deployment"><div className="preview-build-mark"><Box size={18} aria-hidden="true" /></div><div><strong>Update checkout flow</strong><span><GitBranch size={12} aria-hidden="true" /> main <span>·</span> Build #12</span></div><span className="preview-build-state">Active</span></div>
            <div className="preview-log" aria-label="Example deployment output"><div><span>01</span> Sync changed files <Check size={13} aria-hidden="true" /></div><div><span>02</span> Build container image <Check size={13} aria-hidden="true" /></div><div><span>03</span> Check readiness <Check size={13} aria-hidden="true" /></div><div><span>04</span> Switch traffic <Check size={13} aria-hidden="true" /></div></div>
            <div className="preview-history"><span>Previous image</span><strong>Build #11</strong><span>Available for rollback</span></div>
          </>}
          {active === 1 && <>
            <div className="preview-service"><Database size={22} aria-hidden="true" /><div><strong>RelayDB</strong><span>PostgreSQL + PgBouncer</span></div><span className="preview-build-state">Ready</span></div>
            <dl className="preview-properties"><div><dt>Network</dt><dd>Private to this lane</dd></div><div><dt>Application connection</dt><dd><code>DATABASE_URL</code></dd></div><div><dt>Data</dt><dd>Persistent volume</dd></div><div><dt>Credentials</dt><dd>Stable across deployments</dd></div></dl>
          </>}
          {active === 2 && <>
            <div className="preview-access"><LockKeyhole size={24} aria-hidden="true" /><div><strong>Relay Login</strong><p>This dev lane requires an authenticated Relay session.</p></div></div>
            <dl className="preview-properties"><div><dt>Environment</dt><dd>dev</dd></div><div><dt>Branch</dt><dd>main</dd></div><div><dt>Access policy</dt><dd><code>relay-login</code></dd></div><div><dt>Production</dt><dd>Separate lane and policy</dd></div></dl>
          </>}
          <Link className="preview-guide-link" href={tabs[active].href}>Explore {tabs[active].name.toLowerCase()} <ArrowUpRight size={14} aria-hidden="true" /></Link>
        </div>
      </div>
      <div className="preview-tabs" role="tablist" aria-label="Explore the example workspace">
        {tabs.map((tab, index) => <button type="button" key={tab.name} role="tab" id={`${id}-tab-${index}`} aria-selected={active === index} aria-controls={`${id}-panel`} tabIndex={active === index ? 0 : -1}
          ref={(element) => { buttons.current[index] = element; }}
          onClick={() => setActive(index)} onKeyDown={(event) => {
            let next = index;
            if (event.key === "ArrowRight") next = (index + 1) % tabs.length;
            else if (event.key === "ArrowLeft") next = (index + tabs.length - 1) % tabs.length;
            else if (event.key === "Home") next = 0;
            else if (event.key === "End") next = tabs.length - 1;
            else return;
            event.preventDefault(); setActive(next); buttons.current[next]?.focus();
          }}><tab.icon size={14} aria-hidden="true" />{tab.name}</button>)}
      </div>
    </div>
  );
}
