# Relay Context

Relay deploys application lanes and the stateful workloads attached to them.

## Language

**RelayDB**:
A Relay-managed PostgreSQL data workload with stable credentials, connection pooling, resource-aware tuning, readiness checks, and durable storage scoped to one application lane.
_Avoid_: custom database engine, database sidecar

**Companion**:
A stateful or background workload that Relay starts on the same private network as an application lane.
_Avoid_: add-on container, dependency container

**Lane**:
An isolated application deployment target identified by project, environment, and branch.
_Avoid_: instance, stage

**Lane State**:
The durable configuration and rollout snapshot for one Lane, including sticky hostnames, ports, volumes, resource limits, images, and slot state.
_Avoid_: deploy options, app row

**Buildpack**:
A planning adapter that detects an application framework and produces the container build and runtime plan for a Lane.
_Avoid_: framework script, deploy template

**Container Runtime**:
The execution module that builds images, starts workloads, manages networks, and streams workload logs through Docker or Station adapters.
_Avoid_: Docker helper, shell runner

**Control Route**:
An HTTP path that exposes Relay operations through the network or local-socket transport with one attached authorization policy.
_Avoid_: endpoint copy, socket-only API

**SQLite Store**:
The primary and analytics connection pools plus schema migration runner that own Relay's embedded durable storage startup.
_Avoid_: database helper, global DB setup

**Schema Migration**:
An ordered, transactional, replay-safe SQLite schema change recorded in Relay's migration ledger.
_Avoid_: best-effort upgrade, startup ALTER

**Analytics Module**:
The log ingestion, traffic analytics, and administrative operations reporting implementation used by Relay's dashboard.
_Avoid_: analytics helpers, dashboard stats block

**GitHub Delivery Workflow**:
The encrypted GitHub App or legacy token connection, installation-scoped repository mapping, signed webhook deliveries, preview and production deploy triggers, Check Runs, health watch, and rollback controls for one Relay project.
_Avoid_: GitHub deploy helper, webhook automation
