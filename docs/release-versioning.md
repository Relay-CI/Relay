# Release And Versioning Plan

## Versioning

Relay should use semantic versioning:

- `MAJOR`: breaking API, config, or deploy behavior changes
- `MINOR`: new buildpacks, new APIs, backward-compatible features
- `PATCH`: fixes, docs corrections, security hardening without breaking contracts

## Release Units

The repo contains two operator-facing deliverables:

- `relayd`
- `relay` CLI
- `station` runtime for the experimental backend

Releases should publish these together unless there is a strong reason to split them.

## Tag Format

Use annotated Git tags:

```text
v0.1.0
v0.1.1
v0.2.0
```

## Release Checklist

1. Ensure docs match the current agent and CLI behavior.
2. Run `go build` in `relayd/`.
3. Run `go build` in `station/`.
4. Build a release folder with `relayd` and a platform-matched `station` side by side, for example via `./scripts/build-relayd-with-station.ps1`.
5. Verify the CLI command paths touched by the release.
6. Run framework smoke checks for affected buildpacks.
7. Confirm no secrets, logs, databases, or local binaries are staged.
8. Create a changelog summary.
9. Tag and publish the release.

## Changelog Structure

Each release should summarize:

- Added
- Changed
- Fixed
- Security

## Production Gates

Do not cut a production-marked release unless:

- CORS is configured explicitly
- secrets handling is documented
- plugin mutation policy is documented
- rollback behavior is tested
- image retention behavior is verified
- station limitations are documented if the release exposes the experimental engine
