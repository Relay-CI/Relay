# Deploy your first app

This guide installs the published CLI and agent binaries. You do not need Go unless you want to build the agent from source.

[All documentation](README.md) · [Configuration](configuration.md) · [Troubleshooting](troubleshooting.md)

## Before you begin

- **On the server:** Docker installed and running. For a local trial, your laptop can be the server.
- **Wherever you run the CLI:** Node.js 18 or later. The CLI-based agent installer also needs Node.js on the server.
- **On your laptop:** an app with working build and start commands.

A **lane** is a deployment target identified by project, environment, and branch. This guide uses `my-app / dev / main`.

## 1. Install and start the agent — on the server

```sh
npm install -g @relay-org/relay
relay agent install
```

Follow the installer's printed instructions to add the downloaded binaries to your shell's PATH, then run:

```sh
relayd
```

Leave this terminal running. By default, the agent listens on port `8080` and stores state in a `data` folder relative to its working directory.

Open the dashboard at your server's address on port `8080`. For a local trial, use `http://127.0.0.1:8080`. Create the first owner account when prompted. If the server uses a configured static token, authenticate with that token instead.

For a persistent Linux installation that survives SSH disconnects and reboots, use the service installer:

```sh
sudo relayd service install --user relay --group relay --data-dir /var/lib/relayd
```

Choose the persistent data location before deploying apps. See the [agent reference](../relayd/README.md) before changing an existing installation's data directory.

## 2. Connect your project — on your laptop

Install the CLI. If your laptop is also the server, it is already installed; open a second terminal.

```sh
npm install -g @relay-org/relay
```

Change to your app's root folder and run:

```sh
relay init
```

Follow the wizard:

1. Enter the server URL. Use `http://127.0.0.1:8080` for a local trial or your server's HTTPS URL for a remote connection.
2. Follow the authentication prompts for your server.
3. Choose `my-app`, environment `dev`, branch `main`, and engine `docker`.

The wizard creates `.relay.json`. Subsequent project commands reuse its connection and lane settings.

### Optional: connect with explicit credentials

For token mode, replace the example URL and token with your server's values. This command is one line and works in any shell:

```sh
relay init --url https://relay.example.com --token YOUR_TOKEN --app my-app --env dev --branch main
```

On a machine where the agent exposes a Unix socket, use its absolute path. This example assumes a service installation using `/var/lib/relayd`:

```sh
relay init --socket /var/lib/relayd/relay.sock --app my-app --env dev --branch main
```

The socket belongs to the agent's data directory, not your app folder. It uses filesystem permissions and requires no token. Use HTTP if your system does not expose a Unix socket.

## 3. Check the connection and deploy — in your app folder

```sh
relay doctor
relay deploy --stream
```

Resolve reported connection or server setup failures first. Relay then detects the framework, uploads changed files, builds an image, and rolls out the container. Build logs stream to the terminal.

## 4. Verify the result

```sh
relay status
relay projects
```

Open the app URL reported by Relay or the dashboard. Its address depends on whether the server uses direct ports or managed hostnames.

**Dev and staging lanes default to Relay Login.** A login screen does not mean the build failed. Check the lane's access policy in the dashboard before changing how the app is shared.

For a failed deployment, copy its deployment ID and inspect the log:

```sh
relay logs <deploy-id>
```

## Next steps

- [Choose the right configuration file](configuration.md).
- [Troubleshoot a failed check or deployment](troubleshooting.md).
- [Explore CLI commands](../relay-client/README.md).
- [Configure server authentication, storage, and GitHub delivery](../relayd/README.md).
