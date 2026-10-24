# Keeping the docs aligned with Relay

Use executable behavior as the reference. Check handlers, validation, and CLI commands before documenting a new feature. Comments and old README prose can lag behind the implementation.

| Feature | Source to check | Documentation to update |
| --- | --- | --- |
| Variant lifecycle, weight budget, assignments, feedback | `relayd/variants.go`: `handleVariants`, `variantsSupported`, `variantsReport`, `retireVariantAfterDeploy`, `handleEdgeFeedback` | `docs/variants.md`, `/site/src/app/docs/variants/page.tsx` |
| Replica counts and resize deployment | `relayd/replicas.go`: `handleAppScale`; `relayd/main.go` deployment path | `docs/scaling-and-operations.md`, `/site/src/app/docs/scaling/page.tsx` |
| Build config and readiness path | `relayd/buildpack_config.go`; `main.go` readiness validation; `readiness_path_test.go` | `docs/configuration.md`, CLI and scaling guides |
| CPU/memory defaults and manual/auto mode | `relayd/main.go`: `handleAppConfig`, `buildAppContainerSpec`; `relayd/memlimit.go` | Scaling guide and `docs/small-host-memory.md` |
| Runtime-log discovery and SSE | `relayd/main.go`: `handleRuntimeLogTargets`, `handleRuntimeLogStream` | `/site/src/app/docs/operations/page.tsx` |
| Analytics and owner operations report | `relayd/analytics.go`: `handleAnalytics`, `handleAdminOps` | Operations guide and agent API reference |
| HTTP versus socket availability and roles | `relayd/routes.go` plus each handler's lane checks | API, MCP, and operations guides |
| CLI flags and commands | `relay-client/relay.js` help and command handlers | `relay-client/README.md`, site CLI guide |
| MCP tools and input validation | `relay-mcp/index.js`, `relay-mcp/server.test.js` | Generated tool reference plus task guides |

## MCP reference generation

From `relay-mcp/`:

```sh
npm test
npm run docs:sync
npm run docs:check
```

The sync command starts the local stdio MCP server and discovers its actual tools without making agent requests. It updates the README table and `site/src/lib/mcp-tools.json`, which the site uses for its MCP reference and count. Do not edit the generated tool table or JSON directly.

The site is a separate Git repository ignored by the parent repository. Review its generated catalog and content changes there too. Run `npm run docs:check` from this checkout before release; it fails if the generated references are stale.

For each new guide, update `site/src/components/docs-config.ts`, the `/docs` index, and the sitemap. Check that guide search terms and section links reach the actual headings. Add a feature-page or homepage link when the capability should be discoverable before opening the docs.
