# Configuration

The server reads two environment variables. It has no config file and no command-line options other than `--help` and `--version`.

| Variable | Default | What it does |
|---|---|---|
| `VPNB_SOCKET` | `~/Library/Application Support/VPNBypass/control.sock` | Path of the VPN Bypass control socket. The `vpnb` command-line client that ships with the app reads the same variable. |
| `VPN_BYPASS_MCP_READ_ONLY` | unset | Set to `1` (or `true`, `yes`) to register only the 8 tools that change nothing: `status`, `get_logs`, `list_domains`, `list_services`, `get_service`, `list_active_routes`, `custom_list_routes`, `custom_list_rules`. |

## Read-only mode in a client

```json
{
  "mcpServers": {
    "vpn-bypass": {
      "command": "npx",
      "args": ["-y", "vpn-bypass-mcp"],
      "env": { "VPN_BYPASS_MCP_READ_ONLY": "1" }
    }
  }
}
```

For Claude Code:

```sh
claude mcp add vpn-bypass -e VPN_BYPASS_MCP_READ_ONLY=1 -- npx -y vpn-bypass-mcp
```

## Timeouts

A read tool waits up to 35 seconds for the app (the app itself gives up on a read after 30 seconds and answers `timeout`). A tool that changes something waits up to 120 seconds, because the app answers only after the change is saved. When a change times out, the server says so and tells the agent to read the current state before retrying, since the change may still land.

## Security

- The control socket is a UNIX socket in your home folder, readable only by your user, and the app refuses any other user. The server opens no network port.
- A proxy password given to `custom_add_route` or `custom_update_route` goes to the app in a separate `secrets` field. The app never sends it back, and the server never logs or echoes it.
- The server writes its log to stderr: one start line with the version, the tool count, read-only or not, and the socket path. It logs no tool arguments.
