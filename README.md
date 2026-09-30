<p align="center">
  <img src="https://raw.githubusercontent.com/GeiserX/vpn-bypass-mcp/main/docs/images/banner.svg" alt="vpn-bypass-mcp" width="100%">
</p>

<h1 align="center">vpn-bypass-mcp</h1>

<p align="center">
  <a href="https://github.com/GeiserX/vpn-bypass-mcp/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GeiserX/vpn-bypass-mcp/ci.yml?style=flat-square&label=CI" alt="CI"></a>
  <a href="https://github.com/GeiserX/vpn-bypass-mcp/blob/main/LICENSE"><img src="https://img.shields.io/github/license/GeiserX/vpn-bypass-mcp?style=flat-square" alt="License"></a>
</p>

vpn-bypass-mcp is an MCP server for [VPN Bypass](https://github.com/GeiserX/VPN-Bypass), the macOS menu bar app that decides which traffic uses the VPN and which goes around it. An AI agent uses it to read the app's state and change its routing through the app's local control socket. It runs on macOS only.

## Features

- 23 tools, one per app action: status, domain lists, services, routing mode, kernel routes, DNS refresh, the log, and Custom-mode routes and rules.
- Every tool returns the app's own JSON unchanged, so fields a newer app adds reach the agent.
- Tool descriptions tell an agent that has never seen the app what each mode does, and keep kernel routes apart from Custom-mode egress routes.
- MCP hints on every tool (`readOnlyHint`, `destructiveHint`, `idempotentHint`), so a client can ask before a destructive call.
- `VPN_BYPASS_MCP_READ_ONLY=1` registers only the 8 tools that change nothing.
- Errors an agent can act on: the app is not running, the app is older than 4.9.0 (with its version when it reports one), or the app's own error code.
- A proxy password travels in a separate field and is never returned, echoed or logged.
- stdio only; the server opens no network port.

## Quick start

```sh
npx -y vpn-bypass-mcp --version
```

Then add the server to your MCP client (Claude Desktop, Cursor, or any client that reads `mcpServers`):

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

VPN Bypass must be open. The domain, service, active-route, refresh and log tools need VPN Bypass 4.9.0 or newer; `status`, `set_mode` and the Custom-mode tools work with older versions. Claude Code, the release binaries and building from source are in [Getting started](https://github.com/GeiserX/vpn-bypass-mcp/blob/main/docs/getting-started.md).

## Documentation

- [Getting started](https://github.com/GeiserX/vpn-bypass-mcp/blob/main/docs/getting-started.md): requirements, npm, release binary, source, first run
- [Configuration](https://github.com/GeiserX/vpn-bypass-mcp/blob/main/docs/configuration.md): environment variables, read-only mode, timeouts, security
- [Usage](https://github.com/GeiserX/vpn-bypass-mcp/blob/main/docs/usage.md): modes, the two kinds of route, every tool, errors
- [Development](https://github.com/GeiserX/vpn-bypass-mcp/blob/main/docs/development.md): build, test, release

## Related projects

[VPN Bypass](https://github.com/GeiserX/VPN-Bypass), the app, and its [Homebrew tap](https://github.com/GeiserX/homebrew-vpn-bypass).

## License

[MIT](https://github.com/GeiserX/vpn-bypass-mcp/blob/main/LICENSE)
