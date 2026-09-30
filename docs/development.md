# Development

## Build and test

```sh
go build -o vpn-bypass-mcp ./cmd/server
go vet ./...
go test -race ./...
```

The tests run against `internal/fakeapp`, an in-process fake of the app's control socket that frames requests the way the app does and records every request line. Each tool has a test that pins the exact line it sends and checks that the app's result comes back unchanged.

`TestLiveReadOnly` also runs on a Mac where the VPN Bypass socket exists. It calls the read tools only, so it never changes the Mac's routing, and it prints no data from the app.

## Try the binary by hand

```sh
(printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"cli","version":"0"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"status","arguments":{}}}'
 sleep 1) | ./vpn-bypass-mcp
```

The `sleep` keeps stdin open until the answer arrives: the server stops when stdin closes and cancels a call still in flight, which then answers `context canceled`.

The [MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector) works too: `npx @modelcontextprotocol/inspector ./vpn-bypass-mcp`.

## The wire contract

The socket protocol belongs to the app. Its source of truth is in the [VPN Bypass repository](https://github.com/GeiserX/VPN-Bypass): `Sources/VPNBypassCore/CommandRouter.swift` for the Custom-mode commands (`status`, `mode`, `default`, `route.*`, `rule.*`), `Sources/VPNBypassCore/ClassicControl.swift` for the domain, service, active-route, refresh and log commands (4.9.0), both with their arguments and error codes, and `Sources/VPNBypassCore/ControlSocketServer.swift` for the framing (one JSON line each way, at most 64 KiB per request line). `AGENTS.md` in this repository lists the rules a change here has to keep.

## Release

A `v*.*.*` tag runs `.github/workflows/release.yml`: GoReleaser publishes the darwin amd64 and arm64 archives and `checksums.txt` to a GitHub release, then the npm job publishes the wrapper package with npm trusted publishing. The last job publishes `server.json` to the official MCP Registry, signed in with the workflow's OIDC token. Before tagging, set the new version in `package.json` and `server.json`; the workflow stops before publishing anything when either differs from the tag.

## Credits

[mcp-go](https://github.com/mark3labs/mcp-go) for the MCP implementation, [GoReleaser](https://goreleaser.com/) for the release builds.
