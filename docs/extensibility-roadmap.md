# Relay Extensibility Roadmap

This repo now supports more build targets directly in the agent, but there are still two separate product layers:

1. Container deploys
2. Artifact and CI pipelines

Only the first one is implemented today.

## Implemented Today

Buildpacks in the current agent:

- `node-next`
- `node-vite`
- `expo-web`
- `node-generic`
- `go`
- `dotnet`
- `python`
- `java`
- `rust`
- `c-cpp`
- `wasm-static`
- `static`

Framework-aware support already included:

- Flask via the Python buildpack
- FastAPI via the Python buildpack
- C and C++ via the native `c-cpp` buildpack
- WebAssembly static sites via `wasm-static`

## What Belongs In The Next Layer

These requests do not fit cleanly into the current “build image, run container” flow and should become a plugin or artifact pipeline:

- Electron desktop builds
- Native Expo outputs for iOS and Android
- Cross-compiling binaries for macOS, Linux, and Windows
- CI-only steps like linting, testing, notifications, signing, or publishing

## Proposed Plugin Types

### Buildpack plugins

Status:
- Implemented in basic server-side JSON form

Purpose:
- Add new project detection and Dockerfile generation without editing `relayd/main.go`

Shape:
- Detect project
- Produce a build plan
- Optionally expose extra config schema

Current form:
- server-installed JSON plugin with detect rules and Dockerfile template

Likely future forms:
- local executable plugin
- OCI image plugin
- WASM plugin for sandboxed plan generation

### CI step plugins

Purpose:
- Run non-runtime pipeline stages such as `test`, `lint`, `notify`, `package`, or `publish`

Typical chain:

1. `relay ci test`
2. `relay ci lint`
3. `relay deploy`
4. `relay ci notify`

### Registry plugins

Purpose:
- Install reusable plugins from a shared catalog

Proposed CLI:

```bash
relay plugin search wasm
relay plugin install org/plugin-name
relay plugin list
relay plugin upgrade
```

## Cross-Compile Direction

Current Relay builds one Docker image for one runtime target. A future artifact mode should support:

- `relay build --target linux-amd64`
- `relay build --target windows-amd64`
- `relay build --target darwin-arm64`

Outputs should be stored as build artifacts rather than deployed as containers.

That is the right place for:

- `.exe` delivery
- signed desktop packages
- `.dll` and `.so` companion binaries
- release bundles uploaded to object storage or a CDN

## Why Electron Is Different

Electron is not a server workload. Relay can still help build Electron artifacts later, but Electron should land in the artifact pipeline, not the container deploy pipeline.

The same applies to:

- native mobile Expo builds
- desktop installers
- platform-specific executable distribution
