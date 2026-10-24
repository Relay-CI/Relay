# Relay Docs Site

This folder contains the Relay documentation and marketing site built with Next.js (App Router). Open `/docs` for the documentation index or `/get-started` for the first deployment guide.

## Local development

```bash
cd site
npm ci
npm run dev
```

Then open [the docs](http://localhost:3000/docs). Site development requires Node.js 20.9 or later (Next.js 16); the Relay CLI itself supports Node.js 18 or later.

## Build and preview

```bash
npm run build
npm run start
```

## What to update when features change

| Change | Edit |
| --- | --- |
| First deployment instructions | `src/app/get-started/page.tsx` and `../docs/getting-started.md` |
| Task index, concepts, config overview | `src/app/docs/page.tsx` and `../docs/README.md` |
| CLI commands and examples | `src/app/docs/cli/page.tsx` and `../relay-client/README.md` |
| Server settings and APIs | `src/app/docs/agent/page.tsx` and `../relayd/README.md` |
| Variants, visitor assignments, metrics, and feedback | `src/app/docs/variants/page.tsx` and `../docs/variants.md` |
| Replicas, readiness, and CPU/memory limits | `src/app/docs/scaling/page.tsx` and `../docs/scaling-and-operations.md` |
| Runtime logs, analytics, and owner operations | `src/app/docs/operations/page.tsx` |
| Integration or operations guide | Its page under `src/app/docs/` |
| Sidebar labels, search terms, section links | `src/components/docs-config.ts` |
| Search and mobile docs menu | `src/components/docs-sidebar.tsx` |
| Reading layout and code panels | `src/components/site-shell.tsx` |
| Docs typography and responsive styles | `src/app/docs/docs.css` (scoped to `.docs-shell`) |
| Homepage callouts | `src/app/page.tsx` |
| Shared typography, navigation, and static page layouts | `src/app/quiet.css` |
| Graphite palette, homepage composition, and product-preview styles | `src/app/graphite.css` |
| Three.js hero artwork and its motion lifecycle | `src/components/hero-lattice.tsx` |
| Interactive example workspace tabs | `src/components/lane-preview.tsx` |
| Interactive MCP configuration examples | `src/components/mcp-setup.tsx` |
| MCP tool table and count | Generate `src/lib/mcp-tools.json` with `npm run docs:sync` in `../relay-mcp`; do not edit it directly |

The docs site uses a compact `DocsHero`; marketing pages use `PageHero`. The docs sidebar searches guide titles, summaries, keywords, and section labels, rather than full article bodies. Use canonical `/docs/...` URLs for new guide links; older short URLs are kept as redirects.

The local `DESIGN.md` defines the Graphite visual direction and is intentionally ignored by Git. The shared palette uses near-black surfaces, fine borders, off-white buttons, and orange text links. Relay Form is the brand and heading font; locally bundled Inter Variable is used for body text.

The homepage has one lazily imported Three.js polygon graphic. It stops offscreen and when the tab is hidden, supports pause/play and reduced motion, and falls back to static SVG geometry without WebGL. GPU resources and listeners are released on unmount. The workspace preview is a labeled example with keyboard-accessible tabs; it does not call a live agent.

`home-effects.tsx` retains small static compatibility wrappers for existing pages. Keep display effects out of the documentation and use spacing, typography, and section dividers for hierarchy.

## Writing a clear guide

1. Name the task and state what the reader will achieve.
2. List prerequisites before the first command. Label where each command runs: server, laptop, or app folder.
3. Define product terms before relying on them. A lane is identified by project, environment, and branch.
4. Keep runnable commands separate from optional setup. Label example hostnames, paths, and placeholders.
5. Explain the expected result, how to verify it, and what to check if it fails.
6. End with relevant next steps. Keep planning and roadmap statements separate from shipped behavior.

Address the reader directly. Describe what a command does and what happens next. Avoid slogans, release-note narration, and abstract labels such as “control surface” where “dashboard” or “settings” is clearer.

Keep command examples aligned with the shipped CLI and agent behavior. When adding or renaming section IDs, update `docs-config.ts` so search results and “On this page” links stay valid.

See `../docs/source-map.md` for the handlers behind each guide. After changing MCP tools, run `npm run docs:sync` and `npm run docs:check` from `../relay-mcp` to keep its README and the site reference aligned with actual stdio discovery.

## Check changes

```sh
npm run lint
npx tsc --noEmit --incremental false
npm run build
```

Check `/docs`, `/get-started`, and the changed guide on desktop and mobile. Verify search, mobile navigation, section anchors, keyboard focus, and code copying. Long commands should scroll inside code panels rather than widen the page.

This folder is tracked in the main Relay repository alongside the Markdown documentation. Commit site changes and generated MCP references with the corresponding Relay updates. Local design notes, credentials, dependencies, and build output stay ignored.
