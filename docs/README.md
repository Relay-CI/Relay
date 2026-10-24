# Relay documentation

Relay deploys apps on your own server. The `relay` CLI runs on your computer; the `relayd` agent runs on the machine that builds and hosts your apps.

## Learn and deploy

| I want to… | Read |
| --- | --- |
| Deploy an app for the first time | [First deployment](getting-started.md) |
| Understand the configuration files | [Configuration guide](configuration.md) |
| Fix a connection, build, or routing problem | [Troubleshooting](troubleshooting.md) |
| Find a CLI command or flag | [CLI reference](../relay-client/README.md) |
| Test an alternate build with real visitors | [Variants and feedback](variants.md) |
| Scale replicas, configure readiness, or set resource limits | [Scaling and operations](scaling-and-operations.md) |
| Configure the server, authentication, or GitHub delivery | [Agent reference](../relayd/README.md) |
| Connect an AI editor or assistant | [MCP server](../relay-mcp/README.md) |
| Run or edit the docs website | [Site guide](../site/README.md) |

## Terms used in the guides

- **Lane:** an isolated deployment target identified by project, environment, and branch. `my-app / dev / main` is separate from `my-app / prod / main`.
- **Buildpack:** a planning adapter that detects your framework and produces the container build and runtime plan.
- **Companion:** a stateful or background workload started on the same private network as an application lane.
- **RelayDB:** a Relay-managed PostgreSQL data workload with pooling, stable credentials, readiness checks, and durable storage scoped to one lane.
- **Lane State:** the saved configuration and rollout snapshot for a lane, including hostnames, ports, volumes, resource limits, images, and slot state.

## Operations and maintenance

- [Running on small hosts](small-host-memory.md)
- [Documentation source map](source-map.md) — implementation references and where to update each guide.
- [Release versioning](release-versioning.md)
- [Cloud production checklist](cloud-production-checklist.md) — for operators of the cloud offering; not required for a self-hosted first deployment.
- [Contributing](../CONTRIBUTING.md)

## Project plans

[Implementation plan](plan.md) and [extensibility roadmap](extensibility-roadmap.md) describe planning work. Use the guides and component references above for setup and current behavior.
