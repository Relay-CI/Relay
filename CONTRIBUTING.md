# Contributing

## Ground Rules

- Open an issue before large changes.
- Keep changes scoped and reviewable.
- Do not commit secrets, local tokens, databases, logs, or built binaries.
- Update docs when behavior changes.

## Local Setup

### Agent

```bash
cd relayd
go build .
./relayd
```

### CLI

```bash
cd relay-client
npm install
npm link
```

## Development Expectations

- For Go changes, run `gofmt` and `go build`.
- For CLI changes, run the command paths you touched.
- For deploy-path changes, verify with at least one smoke app.
- Keep the root README, [`relayd/README.md`](C:/Users/aloys/Downloads/relay/relayd/README.md), and [`relay-client/README.md`](C:/Users/aloys/Downloads/relay/relay-client/README.md) aligned.

## Pull Requests

Include:

- what changed
- why it changed
- how it was tested
- any security or migration impact

If the change affects deploy behavior, include the framework or app used for verification.
