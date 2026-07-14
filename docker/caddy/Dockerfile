# Custom Caddy build with the Cloudflare DNS provider plugin, needed for
# DNS-01 ACME challenges. Stock caddy:alpine has no DNS provider plugins, and
# DNS-01 is required for any hostname fronted by Cloudflare Tunnel (or any
# other setup where Let's Encrypt can't reach the origin directly for
# HTTP-01/TLS-ALPN-01).
#
# Build:
#   docker build -t relay-caddy-cloudflare:latest -f docker/caddy/Dockerfile docker/caddy
#
# Run relayd with:
#   RELAY_CADDY_IMAGE=relay-caddy-cloudflare:latest
#   CLOUDFLARE_API_TOKEN=<token scoped to Zone:DNS:Edit on the target zone>

FROM caddy:builder AS builder

RUN xcaddy build \
    --with github.com/caddy-dns/cloudflare

FROM caddy:alpine

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
