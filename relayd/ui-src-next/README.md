# Relay dashboard

The dashboard is a Next.js static export served by `relayd` at `/dashboard/`.

## Development

```sh
npm ci
npm run dev
```

The dashboard needs an authenticated agent API. Static files alone do not provide projects, logs, or server settings.

## Build

```sh
npm run build
```

This exports the app, copies it into `../ui`, and creates compressed assets. Rebuild and restart `relayd` to include the updated embedded dashboard.

## Design and behavior

Follow the Graphite direction in the local `../../site/DESIGN.md`: near-black backgrounds, fine borders, white primary buttons, and orange text links. Keep Relay Form for branding and headings. Use a readable system font for controls and tables, and monospace for logs and code.

The overview loads first. Other pages, deployment details, and owner settings load when opened. Avoid adding imports that pull those pages back into the initial bundle.

Runtime-log requests depend on the selected app, environment, and branch. An unchanged dashboard event must not refetch log targets or reconnect the stream. Switching a lane or log target must still update the stream.

## Checks

```sh
npx tsc --noEmit --incremental false
npx eslint src
npm run build
```

Check the overview, lane selection, logs, deployment details, owner settings, and mobile navigation against an agent or a controlled API fixture. Keep preview data clearly separate from a running server.
