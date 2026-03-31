# Relay Smoke Test Results

Agent under test:

- URL: `http://127.0.0.1:8080`
- Token source: `relayd/data/token.txt`

## Verified Buildpacks

| Framework | Fixture | Buildpack | Result | Preview |
| --- | --- | --- | --- | --- |
| Static HTML | `static-basic` | `static` | Pass | `200` |
| Node app | `node-basic` | `node-generic` | Pass | `200` |
| Vite app | `vite-basic` | `node-vite` | Pass | `200` |
| Go app | `go-basic` | `go` | Pass | `200` |
| Flask app | `python-basic` | `python` | Pass | `200` |
| FastAPI app | `fastapi-basic` | `python` | Pass | `200` |
| .NET app | `dotnet-basic` | `dotnet` | Pass | `200` |
| Java app | `java-basic` | `java` | Pass | `200` |
| Rust app | `rust-basic` | `rust` | Pass | `200` |
| C app | `c-basic` | `c-cpp` | Pass | `200` |
| C++ app | `cpp-basic` | `c-cpp` | Pass | `200` |
| WASM static app | `wasm-basic` | `wasm-static` | Pass | `200` |
| Astro app via plugin | `astro-plugin-basic` | `astro-static` | Pass | `200` |

## Notes

- Flask and FastAPI are both handled by the Python buildpack now.
- Rust now runs successfully with the updated default runtime image.
- `wasm-static` is a dedicated static path that adds an nginx config suitable for `.wasm` assets.
- `expo-web` buildpack code exists in the agent, but it was not included in this smoke matrix.
- `astro-static` was installed as a server-side plugin using `relay plugin install plugins/astro-static.json`.
- Electron and desktop cross-compilation are not part of the current container deploy path.

## Most Recent Targeted Runs

| Fixture | Time | Result |
| --- | --- | --- |
| `c-basic` | `26.3s` | `OK deploy ready in 25.4s` |
| `cpp-basic` | `3.8s` | `OK deploy ready in 2.7s` |
| `python-basic` | `4.3s` | `OK deploy ready in 3.2s` |
| `fastapi-basic` | `6.5s` | `OK deploy ready in 5.4s` |
| `wasm-basic` | `4.5s` | `OK deploy ready in 3.4s` |
| `rust-basic` | `5.0s` | `OK deploy ready in 4.1s` |
| `astro-plugin-basic` | `3.8s` | `OK deploy ready in 2.7s` |

## Preview URLs

```text
http://127.0.0.1:3120
http://127.0.0.1:3121
http://127.0.0.1:3122
http://127.0.0.1:3123
http://127.0.0.1:3124
http://127.0.0.1:3125
http://127.0.0.1:3126
```
