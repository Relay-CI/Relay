# Cloud Production Checklist

This is the shortest practical checklist for bringing `cloud-site` and the Relay cloud hub into a real production setup.

## Cloud Site

- Set `DATABASE_URL`
- Set `NEXT_PUBLIC_SITE_URL` to the public HTTPS origin of the cloud site
- Set `RELAY_CLOUD_USER_SESSION_SECRET`
- Set `RELAY_CLOUD_ADMIN_EMAIL`
- Set `RELAY_CLOUD_ADMIN_PASSWORD`
- Set `RELAY_CLOUD_ADMIN_SESSION_SECRET`

## Email

- Set `RESEND_API_KEY`
- Set `RELAY_EMAIL_FROM`
- Optionally set `RELAY_SUPPORT_EMAIL`

Without these, rollout email stays in stubbed log-only mode.

## Billing

- Set `LEMON_SQUEEZY_API_KEY`
- Set `LEMON_SQUEEZY_STORE_ID`
- Set `LEMON_SQUEEZY_WEBHOOK_SECRET`

For managed-node checkout, also set:

- `LEMON_SQUEEZY_MANAGED_NODE_STARTER_VARIANT_ID`
- `LEMON_SQUEEZY_MANAGED_NODE_PRO_VARIANT_ID`
- `LEMON_SQUEEZY_MANAGED_NODE_BUSINESS_VARIANT_ID`

## Managed Nodes

For automatic provisioning:

- Set `HETZNER_CLOUD_API_TOKEN`

Optional local/testing fallback:

- Set `RELAY_MANAGED_NODE_DRY_RUN=1`

Optional DNS automation:

- Set `CLOUDFLARE_API_TOKEN`
- Set `CLOUDFLARE_ZONE_ID`
- Set `RELAY_MANAGED_NODE_BASE_DOMAIN`

Current gap:

- Managed-node TLS issuance is still not implemented in the provisioning flow.

## Cloud Hub

Run a `relayd` instance with cloud hub enabled.

Required on the hub process:

- `RELAY_CLOUD_HUB=true` or `relayd --cloud-hub`
- `RELAY_CLOUD_PG_URL=<postgres connection for agent token + connected server tables>`

The hub process is what allows cloud-issued agent tokens to register and report heartbeats back into the control plane.

## Public Infra

- Put the cloud site behind HTTPS
- Deliver Lemon Squeezy webhooks to the public HTTPS origin
- Point installer traffic (`/install.sh`) at the same public origin
- Back up the production database
- Monitor the cloud site and hub process separately

## Verification

Before opening this to real users:

- `cloud-site` should pass `npm run lint`
- `cloud-site` should pass `npm run build`
- `cloud-site` should pass `npm test`
- A generated Cloud Connect token should register a real `relayd` agent
- A paid managed-node checkout should progress through `payment_pending -> provisioning -> bootstrapping -> active`
