# Getting started

## Requirements

- A Mac running macOS 13 or later with [VPN Bypass](https://github.com/GeiserX/VPN-Bypass) installed and open. The server is macOS only, because it talks to the app through a local socket.
- VPN Bypass 4.9.0 or newer for the domain, service, active-route, refresh and log tools. With an older app, `status`, `set_mode` and the `custom_*` tools work and every other tool answers that it needs 4.9.0 (see [Usage](usage.md#which-tools-need-vpn-bypass-490)).
- The MCP client runs as the same macOS user as VPN Bypass. The app accepts connections only from its own user.
- Node.js 18 or later for the npm install path.

## npm

```sh
npx -y vpn-bypass-mcp --version
```

The package downloads the release binary for your Mac (Intel or Apple silicon), checks it against the `checksums.txt` shipped inside the npm package, and runs it. When your package manager skips install scripts (npm 12, pnpm and bun do by default), the download happens on the first start instead, with the same checks. MCP clients start it the same way, without `--version`:

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

That block goes into Claude Desktop's `claude_desktop_config.json`, Cursor's `.cursor/mcp.json`, or any client that reads the `mcpServers` format. For Claude Code:

```sh
claude mcp add --transport stdio vpn-bypass -- npx -y vpn-bypass-mcp
```

VS Code reads `.vscode/mcp.json` with a `servers` key:

```json
{
  "servers": {
    "vpn-bypass": { "type": "stdio", "command": "npx", "args": ["-y", "vpn-bypass-mcp"] }
  }
}
```

Codex reads `~/.codex/config.toml`:

```toml
[mcp_servers.vpn-bypass]
command = "npx"
args = ["-y", "vpn-bypass-mcp"]
```

## Release binary

Each [release](https://github.com/GeiserX/vpn-bypass-mcp/releases) has `vpn-bypass-mcp_<version>_darwin_arm64.tar.gz` (Apple silicon) and `..._darwin_amd64.tar.gz` (Intel):

```sh
curl -LO https://github.com/GeiserX/vpn-bypass-mcp/releases/download/v0.2.0/vpn-bypass-mcp_0.2.0_darwin_arm64.tar.gz
tar -xzf vpn-bypass-mcp_0.2.0_darwin_arm64.tar.gz vpn-bypass-mcp
./vpn-bypass-mcp --version
```

Point your client's `command` at the full path of the binary, with no `args`.

## From source

With Go 1.27.3 or later:

```sh
git clone https://github.com/GeiserX/vpn-bypass-mcp
cd vpn-bypass-mcp
go build -o vpn-bypass-mcp ./cmd/server
```

## First run

Ask your agent what mode VPN Bypass is in. It calls `status`, which returns the routing mode and the live facts: whether the privileged helper is ready, whether a VPN is connected, how many kernel routes are installed, and from 4.9.0 on the app version.

If the app is closed, every tool answers `VPN Bypass is not running; start the app`. Open VPN Bypass and ask again.

To let the agent look without changing anything, start the server with `VPN_BYPASS_MCP_READ_ONLY=1` ([Configuration](configuration.md)).
