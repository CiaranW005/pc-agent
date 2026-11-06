# BMC PC Agent

The PC agent runs on the managed Linux PC. It accepts commands from the ESP32, requests shutdown or reboot through systemd, and returns availability telemetry for [BMC](../README.md).

## Implementation

`cmd/start-agent/main.go` starts a TCP listener on all interfaces at port 9090. The connection manager handles one connection at a time and waits for another after the current session ends. JSON packets are separated by newlines, matching the ESP32's TCP client.

The router selects an operation without executing it. The connection manager sends an acknowledgement first, then runs the operation. This ordering allows the ESP32 to receive confirmation before shutdown or reboot disconnects the PC.

A five-minute deduplication window prevents repeated execution of the same message ID. Duplicate commands receive another acknowledgement but run a no-op. Acknowledgement packets are consumed without acknowledging them back.

`internal/executor` invokes `systemctl poweroff` or `systemctl reboot`. Successful invocation confirms that the request was made, not that the machine has completed the transition. Execution failures produce a tracked response with `success: false` and an error message. Successful power operations do not send completion responses.

Telemetry currently returns only `{"pc": true}`. It means the agent is running and responding; CPU, memory, temperature, and other hardware metrics are not collected. Telemetry replies carry the original request ID in `reply_to` and are tracked for acknowledgement.

## Run and build

Use the parent BMC workspace so the shared protocol module resolves locally. The agent's module declares Go 1.25.0; the workspace and release workflow use Go 1.26.0.

From this directory:

```bash
go run ./cmd/start-agent
go build -o bmc-agent ./cmd/start-agent
```

For the same target as the release workflow:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bmc-agent ./cmd/start-agent
```

Configure the ESP32 to connect to this PC's LAN address on port 9090. The current TCP listener has no authentication or TLS; deployment assumes access is restricted to the intended local network. The account running the agent must be authorized to perform systemd shutdown/reboot operations.

## systemd deployment

`deploy/systemd/bmc-agent.service` expects:

- A `bmc-agent` user and group.
- The executable at `/usr/local/bin/bmc-agent`.
- Authorization for that user to request power operations.

The unit runs with `NoNewPrivileges`, a read-only system filesystem, protected home directories, and a private temporary directory. The repository does not include the host's power-operation authorization policy.

The agent wants `bmc-agent-update.service` and starts after its update attempt. An updater failure does not by itself prevent the installed agent from starting.

## Updates

Install `deploy/scripts/update.sh` at `/usr/local/libexec/bmc-agent/update.sh` for the supplied updater unit. It runs as root and expects a GitHub token at `/etc/bmc-agent/github-token`, plus `curl`, `jq`, and `sha256sum`.

The updater reads the latest release from `CiaranW005/BMC`, downloads `bmc-agent` and `bmc-agent.sha256`, verifies the checksum, and compares the binary with the installed copy. A changed binary is staged beside the installed executable and renamed into place. The old executable is retained as `/usr/local/bin/bmc-agent.previous` for manual rollback.

The updater is a one-shot service, not a periodic timer. It does not restart a running agent itself. Releases are built and published by the parent repository's PC agent workflow.
