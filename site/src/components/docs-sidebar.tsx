"use client";

import Link from "next/link";
import { Search, ChevronDown, X } from "lucide-react";
import { useDeferredValue, useMemo, useState } from "react";
import type { DocsPage } from "@/components/docs-config";

export function DocsSidebar({ pages, currentPath }: { pages: DocsPage[]; currentPath: string }) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const deferredQuery = useDeferredValue(query);
  const currentPage = pages.find((page) => page.href === currentPath) ?? pages[0];
  const groups = ["Start here", "Reference", "Integrations", "Operations"];

  const results = useMemo(() => {
    const terms = deferredQuery.toLowerCase().trim().split(/\s+/).filter(Boolean);
    if (!terms.length) return [];
    const matches = (text: string) => terms.every((term) => text.toLowerCase().includes(term));
    return pages.flatMap((page) => {
      const pageText = [page.label, page.summary, ...(page.keywords ?? [])].join(" ");
      const pageResult = matches(pageText) ? [{ href: page.href, title: page.label, description: page.summary }] : [];
      // Only search section-specific text; a page match should not duplicate every section.
      const sections = page.sections.filter((section) => matches([section.label, ...(section.keywords ?? [])].join(" ")))
        .map((section) => ({ href: `${page.href}#${section.id}`, title: section.label, description: page.label }));
      return [...pageResult, ...sections];
    }).slice(0, 12);
  }, [deferredQuery, pages]);

  return (
    <aside className="docs-sidebar" aria-label="Documentation navigation">
      <div className="docs-search">
        <label htmlFor="docs-search">Find a guide</label>
        <div className="docs-search-input">
          <Search size={16} aria-hidden="true" />
          <input id="docs-search" type="search" value={query} onChange={(event) => setQuery(event.target.value)}
            placeholder="Try secrets or Docker" aria-describedby="docs-search-hint" aria-controls="docs-search-results" />
          {query && <button type="button" aria-label="Clear search" onClick={() => setQuery("")}><X size={16} aria-hidden="true" /></button>}
        </div>
        <p id="docs-search-hint">Search guide titles and topics.</p>
      </div>
      <div className="sr-only" role="status">{deferredQuery.trim() ? `${results.length} results` : ""}</div>
      <div id="docs-search-results">
        {deferredQuery.trim() && <nav aria-label="Search results" className="docs-search-results">
          {results.length ? results.map((result) => <Link key={result.href} href={result.href} onClick={() => { setQuery(""); setOpen(false); }}>
            <strong>{result.title}</strong><span>{result.description}</span>
          </Link>) : <p>No guides found. Try “deploy”, “database”, or a config file name.</p>}
        </nav>}
      </div>
      <button type="button" className="docs-menu-toggle" aria-expanded={open} aria-controls="docs-menu" onClick={() => setOpen(!open)}>
        Browse documentation <ChevronDown size={16} aria-hidden="true" />
      </button>
      <div id="docs-menu" className="docs-menu" data-open={open}>
        <nav aria-label="Documentation guides">
          {groups.map((group) => <div key={group} className="docs-nav-group">
            <h2>{group}</h2>
            {pages.filter((page) => page.category === group).map((page) => <Link key={page.href} href={page.href}
              aria-current={page.href === currentPath ? "page" : undefined} onClick={() => setOpen(false)}>{page.label}</Link>)}
          </div>)}
        </nav>
        <nav aria-label="On this page" className="docs-toc">
          <h2>On this page</h2>
          {currentPage.sections.map((section) => <a key={section.id} href={`#${section.id}`} onClick={() => setOpen(false)}>{section.label}</a>)}
        </nav>
      </div>
    </aside>
  );
}
