# Related projects

## The app

- [VPN Bypass](https://github.com/GeiserX/VPN-Bypass): the macOS menu bar app this server drives. Its site is [geiserx.github.io/VPN-Bypass](https://geiserx.github.io/VPN-Bypass/), and its [MCP server page](https://geiserx.github.io/VPN-Bypass/mcp-server/) shows the same setup from the app's side.
- [`vpnb`](https://geiserx.github.io/VPN-Bypass/usage/#command-line-control-vpnb): the command-line client that ships with the app. It talks to the same control socket and reads the same `VPNB_SOCKET` variable, so a shell script can do what an agent does here.
- [homebrew-vpn-bypass](https://github.com/GeiserX/homebrew-vpn-bypass): the Homebrew tap that installs the app.

## Where this server is listed

- [npm](https://www.npmjs.com/package/vpn-bypass-mcp): `vpn-bypass-mcp`, the package `npx` runs.
- The official MCP Registry, as `io.github.GeiserX/vpn-bypass-mcp` ([registry entry](https://registry.modelcontextprotocol.io/v0/servers?search=io.github.GeiserX/vpn-bypass-mcp), JSON). The release workflow publishes each version there.
- [Glama](https://glama.ai/mcp/servers/@GeiserX/vpn-bypass-mcp).
- [GitHub releases](https://github.com/GeiserX/vpn-bypass-mcp/releases): the darwin binaries and `checksums.txt`.
