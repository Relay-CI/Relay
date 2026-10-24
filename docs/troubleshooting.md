# Troubleshooting

[All documentation](README.md) · [First deployment](getting-started.md)

Run these checks from the app folder so Relay uses the same `.relay.json` and lane as your deployment.

## The CLI cannot connect to the agent

```sh
relay doctor
```

1. Confirm `relayd` is running on the server.
2. Check the URL or socket in `.relay.json`. For remote HTTP, check that the hostname and port are reachable from your laptop.
3. For HTTP authentication failures, use `relay login --url https://relay.example.com` with your actual server URL, or check the configured static token.
4. For a Unix socket, use the agent's real socket path and check the current user's filesystem permissions. A relative path is resolved from where the CLI runs, so prefer an absolute path.

If the operating system does not expose a Unix socket, connect through HTTP. See the [CLI connection reference](../relay-client/README.md).

## A build or deployment fails

```sh
relay status
relay list
relay logs <deploy-id>
```

Replace `<deploy-id>` with the failed deployment's ID. Read the first failing build step and check:

- Docker is running and accessible to the agent on the server.
- The project builds locally with the same install/build/start commands.
- `relay.config.json` points to the correct app root, build context, and Dockerfile.
- `.relayignore` does not exclude required source files.
- The server has enough resources. For constrained machines, see [small-host memory behavior](small-host-memory.md).

## The deployed app asks for login

Dev and staging lanes default to Relay Login. Authenticate with the Relay identity authorized for the lane, or review the lane's access policy in the dashboard. A login screen is not evidence of a failed build.

## The deployment succeeded but its URL does not work

1. Use the URL reported by Relay or the dashboard; do not assume it is the same port as the dashboard.
2. In direct port mode, check the server's host port, network access, and the app's container service port.
3. In managed host mode, check the hostname's DNS and server routing settings.
4. Run `relay doctor` and follow any DNS, TLS, or proxy findings.
5. Confirm that you are viewing the intended app/environment/branch lane with `relay projects`.

## Data or history appears missing after a restart

Check that the agent is using the same persistent data directory. A foreground `relayd` process defaults to a `data` folder relative to its working directory; starting it elsewhere can select a different directory. Review `RELAY_DATA_DIR` or `--data-dir` and the [agent storage reference](../relayd/README.md) before moving or replacing files.

## A GitHub push does not create a deployment

- For a connected GitHub App, confirm its installation has access to the repository, that the Relay app is linked, and that the production branch is correct.
- Inspect GitHub's webhook delivery response and Relay's deployment list.
- For a manual webhook, check the repository URL mapping, selected events, and matching webhook secret.

See [GitHub delivery in the agent reference](../relayd/README.md#connected-github-delivery-workflow).

## Information to include in a bug report

Include `relay version`, the failed command, the app/environment/branch, relevant `relay doctor` findings, and the failing log excerpt. Remove tokens, passwords, connection strings, and other secret values before sharing.
