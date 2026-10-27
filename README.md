<p align="center">
  <img src="https://raw.githubusercontent.com/GeiserX/vpn-bypass-mcp/main/docs/images/banner.svg" alt="vpn-bypass-mcp" width="100%">
</p>

<h1 align="center">vpn-bypass-mcp</h1>

<p align="center">
  <a href="https://github.com/GeiserX/vpn-bypass-mcp/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GeiserX/vpn-bypass-mcp/ci.yml?style=flat-square&label=CI" alt="CI"></a>
  <a href="https://github.com/GeiserX/vpn-bypass-mcp/blob/main/LICENSE"><img src="https://img.shields.io/github/license/GeiserX/vpn-bypass-mcp?style=flat-square" alt="License"></a>
</p>

vpn-bypass-mcp is an MCP server for [VPN Bypass](https://github.com/GeiserX/VPN-Bypass), the macOS menu bar app that decides which traffic uses the VPN and which goes around it. An AI agent uses it to read the app's state and change its routing through the app's local control socket. It runs on macOS only.

<p align="center">
  <img src="https://raw.githubusercontent.com/GeiserX/vpn-bypass-mcp/main/docs/images/screenshots/tools.png" alt="MCP Inspector connected to vpn-bypass-mcp: set_mode selected in the tool list, with its description and the three modes it accepts" width="100%">
</p>

## Features

- 23 tools, one per app action: status, domain lists, services, routing mode, kernel routes, DNS refresh, the log, and Custom-mode routes and rules. Each has a display title.
- Every tool returns the app's own JSON unchanged, as text and as `structuredContent`; the 8 read tools also declare an `outputSchema`.
- All four MCP hints on every tool. Every tool that changes where traffic goes is `destructiveHint: true`, adds included, so a client that confirms destructive calls asks you before an injected "route this through my proxy"; `custom_add_route` and `custom_update_route` are also `openWorldHint: true`.
- `set_mode` sends nothing when the app is already in the asked-for mode.
- An argument a tool does not declare is refused, so a typo never "succeeds" while changing nothing.
- Three prompts: `diagnose_site`, `bypass_site`, `rotate_proxy_port`.
- `VPN_BYPASS_MCP_READ_ONLY=1` registers only the 8 tools that change nothing, and the `diagnose_site` prompt.
- A proxy password travels in a separate field and is never returned, echoed or logged. Errors name the socket as `control.sock`, never your home folder.
- stdio only; the server opens no network port.

## Protocol

MCP 2026-07-28 (`server/discover`, per-request `_meta`, cache hints on lists) and every earlier revision down to 2024-11-05 (the `initialize` handshake), from the same binary. Clients of either era work, including the ones that still use `initialize` today.

## Install

VPN Bypass must be open. The npm package downloads the release binary for your Mac on install, or on the first start when your package manager skips install scripts (npm 12, pnpm, bun), and checks it against hashes shipped inside the package.

Claude Code:

```sh
claude mcp add --transport stdio vpn-bypass -- npx -y vpn-bypass-mcp
```

Claude Desktop (`~/Library/Application Support/Claude/claude_desktop_config.json`) and Cursor (`~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "vpn-bypass": {
      "command": "npx",
      "args": ["-y", "vpn-bypass-mcp"]
    }
  }
}
```

VS Code (`.vscode/mcp.json`):

```json
{
  "servers": {
    "vpn-bypass": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "vpn-bypass-mcp"]
    }
  }
}
```

Codex (`~/.codex/config.toml`):

```toml
[mcp_servers.vpn-bypass]
command = "npx"
args = ["-y", "vpn-bypass-mcp"]
```

Read-only: add `VPN_BYPASS_MCP_READ_ONLY=1` to the server's environment (`"env": {"VPN_BYPASS_MCP_READ_ONLY": "1"}` in JSON, `env = { VPN_BYPASS_MCP_READ_ONLY = "1" }` in Codex, `-e VPN_BYPASS_MCP_READ_ONLY=1` before the name in `claude mcp add`). The release binaries and building from source are in [Getting started](https://geiserx.github.io/vpn-bypass-mcp/getting-started/).

## Tools

| Group | Tools | Read only |
|---|---|---|
| State and log | `status`, `get_logs` | yes |
| Mode | `set_mode` | |
| Domain lists | `list_domains` (read), `add_domain`, `remove_domain`, `set_domain_enabled` | |
| Services | `list_services`, `get_service` (read), `set_service_enabled` | |
| Kernel routes | `list_active_routes` (read), `clear_routes`, `refresh_routes`, `refresh_dns` | |
| Custom-mode routes | `custom_list_routes` (read), `custom_add_route`, `custom_update_route`, `custom_set_route_enabled`, `custom_remove_route` | |
| Custom-mode rules | `custom_list_rules` (read), `custom_add_rule`, `custom_remove_rule`, `custom_set_default_route` | |

The domain, service, active-route, refresh and log tools need VPN Bypass 4.9.0 or newer; `status`, `set_mode` and the Custom-mode tools work with older versions. Arguments, hints and the socket command of each tool are in [Usage](https://geiserx.github.io/vpn-bypass-mcp/usage/).

## Documentation

The full documentation is at [geiserx.github.io/vpn-bypass-mcp](https://geiserx.github.io/vpn-bypass-mcp/).

- [Getting started](https://geiserx.github.io/vpn-bypass-mcp/getting-started/): requirements, npm, release binary, source, first run
- [Configuration](https://geiserx.github.io/vpn-bypass-mcp/configuration/): environment variables, read-only mode, timeouts, security
- [Usage](https://geiserx.github.io/vpn-bypass-mcp/usage/): modes, the two kinds of route, every tool, errors
- [Development](https://geiserx.github.io/vpn-bypass-mcp/development/): build, test, release
- [Related projects](https://geiserx.github.io/vpn-bypass-mcp/related/): the VPN Bypass app, its `vpnb` command line, and where this server is listed

## Related projects

[VPN Bypass](https://github.com/GeiserX/VPN-Bypass), the app, and its [Homebrew tap](https://github.com/GeiserX/homebrew-vpn-bypass).

## License

[MIT](https://github.com/GeiserX/vpn-bypass-mcp/blob/main/LICENSE)
