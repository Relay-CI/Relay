# Variants and visitor feedback

[All documentation](README.md) · [Replicas and resources](scaling-and-operations.md)

A variant runs an alternate build beside a lane's normal release, called control. It shares the lane's URL, configuration, secrets, and services. Use another lane if you need a separate database or environment.

## Requirements

- A deployed Docker lane with a running control release.
- `traffic_mode: edge`. Station and session traffic mode do not support variants.
- Deployer access to make changes; viewer access to read reports.
- Capacity for the extra containers. Each variant starts with the lane's replica count.

## Deploy and inspect

Run in your app folder, using its Relay connection settings:

```sh
relay deploy --app my-app --env prod --branch main --variant checkout --weight 10
relay variants list --app my-app --env prod --branch main --period 24h
```

Change the source and deploy with the same name to update that variant. Ordinary deployments without `--variant` update control without replacing variants. Use names of 1–24 lowercase letters, digits, or hyphens, starting with a letter or digit. Names such as `control`, `blue`, and `green` are reserved. A lane can have at most eight variants.

Open **Variants** after selecting the same lane in the dashboard. Compare visitors, page views, requests, server errors, p50/p95 latency, and feedback over 24h, 7d, or 30d. Metrics describe observed proxy traffic; they do not establish statistical significance.

## Change traffic

```sh
relay variants weight checkout 25 --app my-app --env prod --branch main
relay variants reshuffle --app my-app --env prod --branch main
```

Weights apply to new visitor assignments. A cookie keeps a visitor on their assigned version for up to 30 days. All running variant weights together must be at most 100%; control gets the remainder. Zero weight stops new automatic assignments but does not evict existing visitors. Reshuffle resets assignments for everyone.

Preview the app with `?__relay_variant=checkout`, or `?__relay_variant=control`. This sets your assignment cookie too. Check the `X-Relay-Variant` response header to see the selected version.

## Collect feedback

From browser code in your app, post to the app's own edge URL:

```js
await fetch("/__relay/feedback", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ vote: "up", rating: 5, comment: "Easy to use", page: location.pathname })
});
```

Send at least a vote (`up`/`down`), a rating (1–5), or a comment. Comments are limited to 2,000 characters, page values to 256. The edge supplies the lane and variant; no agent token belongs in browser code. Submissions are limited to ten per minute per lane/client IP.

Use `relay variants feedback --app my-app --env prod --branch main --period 7d`, the dashboard, or MCP `list_variants` for the latest 50 entries in the selected period.

## Finish the experiment

```sh
relay variants promote checkout --app my-app --env prod --branch main
# Or remove it:
relay variants rm checkout --app my-app --env prod --branch main
```

Promote queues the variant's existing image as the lane's control release and returns a deployment ID. Relay removes the variant after that deployment succeeds; failure leaves the variant available. Removing a variant removes its route and containers, and its visitors fall back to control.

This is separate from cross-lane promotion approvals. MCP exposes `list_variants`, `set_variant_weight`, `promote_variant`, `remove_variant`, and `reshuffle_variants`. Creating or rebuilding a variant from source still uses the CLI.

The API is `GET /api/variants` with lane fields and `period`, or `POST /api/variants` with lane fields and an `action` of `weight`, `promote`, `remove`, or `reshuffle`. Both HTTP and the local socket expose these routes.
