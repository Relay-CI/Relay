# Relay Admin UI Redesign

**Date:** 2026-03-19
**Status:** Approved
**Scope:** `relayd/ui-src/src/` — `styles.css` + `index.jsx`

---

## Goal

Redesign the Relay admin dashboard (Control Room) to match the website's warm design language, use a modern dark theme (Vercel/Supabase quality), and restructure key components for clarity. The result should feel cohesive with the `site/` marketing site while being purpose-built for a dashboard context.

---

## Design Decisions

### Color Palette — Warm Dark

Replace the current cold blue-black + teal palette. The new `:root` block for `styles.css` (**author this first — all subsequent rules depend on it**):

```css
:root {
  --bg:              #0e0c0a;
  --bg-deep:         #0a0907;
  --panel:           rgba(20, 18, 15, 0.92);
  --line:            rgba(245, 240, 232, 0.07);
  --line-strong:     rgba(245, 240, 232, 0.12);
  --text:            #f0ece4;
  --muted:           #8a7e72;
  --muted-deep:      #5a5248;
  --accent:          #c65b2f;
  --accent-bright:   #e8764a;
  --accent-gradient: linear-gradient(90deg, #c65b2f, #e07840);
  --success:         #50c88c;
  --warn:            #e09040;
  --danger:          #e05050;
  --shadow:          0 40px 120px rgba(0, 0, 0, 0.5);
}
```

### Typography

| Use | Font | Weight |
|---|---|---|
| All UI text | Space Grotesk (Google Fonts) | 400, 500, 600, 700 |
| Code / monospace | IBM Plex Mono (Google Fonts) | 400, 500 |

Loaded via `<link>` in `index.html`. `body` font-family becomes `'Space Grotesk', sans-serif`.

---

## Component Changes

### `index.html`
- Add `<link>` for Space Grotesk + IBM Plex Mono from Google Fonts before the stylesheet link.
- Update the inline SVG favicon: replace `%230C1118` → `%230e0c0a`, `%23163238` → `%232a1a10`, `%237AF0D4` → `%23e8764a`.

### `styles.css` — full rewrite

**`body` and `body::before`**

`body` uses the new warm radial gradient background:
```css
background:
  radial-gradient(circle at top left, rgba(198, 91, 47, 0.06), transparent 30%),
  radial-gradient(circle at 80% 10%, rgba(230, 140, 70, 0.04), transparent 25%),
  #0e0c0a;
```

Retain `body::before` (the subtle grid texture). The `rgba(255,255,255, 0.035)` grid lines are neutral and work with the warm theme unchanged.

**`select` / `option` background**

Replace the cold hard-coded `background: #0d1118` with `background: var(--bg-deep)`.

**Teal replacement — all occurrences**

All `rgba(122, 240, 212, ...)` / `#7af0d4` / `#c3fff0` values are replaced. Full enumeration:

- `--accent` (orange) replaces teal accent on backgrounds/borders in: `.selector-item--active`, `.nav__item--active`, `.context-card--active`, `.text-input:focus` (border + box-shadow), `.status-chip--ok` (background + border), `.metric-card--accent` background.
- `--accent-bright` (brighter orange) replaces `--accent-strong` (teal text) in: `.context-card__url`, `.row-card__badge`, `.link-teal`, `.code-callout pre`, `.status-chip--ok` text color.
- `.link-teal` class name is **kept** in CSS and JSX — only its `color` rule updates to `var(--accent-bright)`.

**`.sidebar`**

Add explicit `background: var(--bg-deep)`. Place the `.sidebar` rule **after** `.panel` in the stylesheet so it overrides `.panel`'s background (both are single-class selectors — source order determines precedence). The JSX keeps `className="sidebar panel"` unchanged.

**Hero rules — deleted**

Remove all of the following CSS rules entirely: `.hero`, `.panel--hero`, `.hero__title`, `.hero__grid`, `.hero__body`. Note: `.hero__body` is currently grouped with `.login-card__body` in a shared rule:
```css
.hero__body,
.login-card__body { ... }
```
Split this rule — **keep** `.login-card__body` with its `max-width`, `color`, and `line-height` properties; remove only `.hero__body` from the selector list.

**`.topbar__title`**

The `<h1 className="topbar__title">` is replaced by the breadcrumb in JSX. Remove the `.topbar__title` CSS rule (currently `font-size: 2.4rem`) as it becomes dead code.

**New rules**

```css
.nav-section-label {
  font-size: 0.62rem;
  font-weight: 700;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--muted-deep);
  padding: 4px 10px;
  margin-top: 8px;
}

.metric-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.breadcrumb { display: flex; align-items: center; gap: 8px; font-size: 0.85rem; color: var(--muted); }
.breadcrumb__sep { color: var(--muted-deep); }
.breadcrumb__active { color: var(--text); font-weight: 600; text-transform: capitalize; }
```

**`.primary-button`** background: `var(--accent-gradient)`, color: `#fff8f2` (warm white).

**`.status-chip--ok`** uses `--success` (green) for background, border, and text.

**Responsive media queries**

Inside the existing `@media (max-width: 720px)` block:
- Remove `.hero` from the `flex-direction: column` selector list.
- Remove the `.hero__title, .topbar__title { font-size: 2rem }` rule entirely (both classes are gone from JSX).
- Add inside the same block: `.metric-row { grid-template-columns: 1fr; }`

---

### `App` component — shell layout
No structural change to `grid-template-columns: 320px 1fr`.

---

### Sidebar JSX changes — `<aside className="sidebar panel">`

**1. Brand block** — the current JSX at lines 987–993 has eyebrow "Relay Admin" / title "Control Room". Replace the entire inner content with:
```jsx
<div className="brand">
  <div className="brand__glyph">R</div>
  <div>
    <div className="brand__title">Relay</div>
    <div className="eyebrow">Control Room</div>
  </div>
</div>
```
(Product name "Relay" in the title slot; "Control Room" in the smaller eyebrow slot.)

**2. Nav section label** — insert immediately before `<nav className="nav">`:
```jsx
<div className="nav-section-label">Navigation</div>
```

**3. Nav icons** — use [Lucide](https://lucide.dev) SVG paths (MIT license, no runtime dependency needed — inline the SVGs). Each `nav__item` button wraps an icon + label:
```jsx
<button className={cx("nav__item", activeTab === id && "nav__item--active")} ...>
  {icon}  {/* 16×16 inline SVG, stroke="currentColor", strokeWidth={1.5}, fill="none" */}
  {label}
</button>
```
Icon assignments (use Lucide outline variants):
- `overview` → `LayoutGrid` (four squares)
- `deployments` → `GitBranch`
- `settings` → `Settings2` (sliders)

The nav array in `App` (currently `[["overview","Overview"], ...]`) becomes an array of `[id, label, iconJSX]` tuples, or the icons are defined inline with a small map — implementer's choice of approach as long as icons render inside the button.

---

### Topbar JSX — `<header className="topbar">`

Remove:
```jsx
<div>
  <div className="eyebrow">Latest Changes Reflected</div>
  <h1 className="topbar__title">{selectedProject ? selectedProject.name : "No project deployed"}</h1>
</div>
```

Replace with:
```jsx
<div className="breadcrumb">
  <span>{selectedProject ? selectedProject.name : "No project"}</span>
  <span className="breadcrumb__sep">/</span>
  <span className="breadcrumb__active">{activeTab}</span>
</div>
```

---

### Hero panel → Metric row — JSX in `App`

**Remove** the entire `<section className="hero panel panel--hero">...</section>` block (lines 1054–1072 in current `index.jsx`).

**Replace** with (placed immediately above `<ContextBar>`):
```jsx
<div className="metric-row">
  <MetricCard label="Environments" value={String(selectedProject.envs.length)} meta="project contexts" />
  <MetricCard label="Services" value={String(selectedProject.services.length)} meta="live companions" accent />
  <MetricCard
    label="Latest Deploy"
    value={selectedEnv?.latestDeploy ? selectedEnv.latestDeploy.status.toUpperCase() : "IDLE"}
    meta={selectedEnv?.latestDeploy ? `${timeAgo(selectedEnv.latestDeploy.created_at)} ago` : "waiting for first deploy"}
  />
</div>
```

---

### `LoginScreen`

Add `.brand__glyph` div at the top of the form card, above the eyebrow:
```jsx
<div className="brand__glyph">R</div>
```
Input/button restyling is CSS-only.

---

### `LogViewer` modal
CSS-only: overlay background → `rgba(10, 9, 7, 0.8)`; log output background → `rgba(8, 6, 4, 0.92)`.

### `ContextBar` / `OverviewTab` / `DeploymentsTab` / `SettingsTab`
CSS-only updates. Table `th` color → `var(--muted-deep)`.

---

## What Is NOT Changing

- All API calls, data fetching, polling, and auth logic — untouched.
- Component hierarchy and prop interfaces — unchanged.
- `build.mjs` build pipeline — unchanged.
- `SplashScreen`, `EmptyState`, `Detail`, `ServiceCard` — CSS updates only.

---

## Files Changed

| File | Change type |
|---|---|
| `relayd/ui-src/src/index.html` | Add Google Fonts link; update favicon SVG fill values |
| `relayd/ui-src/src/styles.css` | Full rewrite — new `:root`; all teal replaced; `select/option` bg fixed; `.sidebar` bg override; hero rules deleted; `.topbar__title` deleted; `.login-card__body` split from `.hero__body`; new `.nav-section-label`, `.metric-row`, `.breadcrumb` rules; responsive block updated |
| `relayd/ui-src/src/index.jsx` | Sidebar: brand block text corrected, nav-section-label added, nav icons added. Topbar: breadcrumb replaces h1+eyebrow. Hero panel removed, `.metric-row` div added. LoginScreen: brand glyph added. |

---

## Success Criteria

- Admin dashboard visually matches the site's warm editorial tone in dark mode.
- All existing functionality (login, project switching, env selection, deploy logs, settings, secrets) works identically.
- Sidebar nav has icons on all items.
- No hero panel — metric row replaces it above the fold.
- Topbar shows breadcrumb instead of large project title.
- Login screen has the brand logo glyph.
- No cold blue/teal colour values remain anywhere in `styles.css` or `index.html`.
