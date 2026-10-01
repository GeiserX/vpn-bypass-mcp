# vpn-bypass-mcp

MCP server (Go, stdio only, macOS only) for [VPN Bypass](https://github.com/GeiserX/VPN-Bypass). Each tool sends one request to the app's control socket and returns the app's `result` object unchanged.

## The wire contract

The app owns the contract. Its source of truth is in the VPN-Bypass repo: `Sources/VPNBypassCore/CommandRouter.swift` (Custom-mode verbs: `status`, `mode`, `default`, `route.*`, `rule.*`), `ClassicControl.swift` (the 4.9.0 `domain.*`, `service.*`, `routes.*`, `refresh`, `dns.refresh` and `logs` verbs; both hold their arguments and error codes), `ControlSocketServer.swift` (framing: one JSON line each way, 64 KiB per request line, 30 s server timeout on reads, no timeout on mutations) and `ControlSurface.swift`. Read them before adding or changing a tool.

- Every `args` value is a string. Tools convert numbers and booleans in `internal/tools/args.go`.
- A proxy password travels in `secrets.pass` only. It never appears in a result, an error message or a log line.
- The result stays raw JSON (`json.RawMessage`), so a field the app adds later reaches the agent with no change here.
- Read verbs are listed in `client.IsMutating`; every other verb counts as mutating, the same default the app uses. A new read verb goes into that list and into the tool's `ReadOnly` flag, or read-only mode hides it.
- The domain, service, `routes.active`, `routes.clear`, `refresh`, `dns.refresh` and `logs` verbs exist from VPN Bypass 4.9.0. Older apps answer `unknown_command`, which `internal/tools/tools.go` turns into the 4.9.0 message.

## Two meanings of "route"

Keep them apart in every description and doc. A kernel route is an entry the app installed in the macOS routing table (`routes.active`, `routes.clear`, `refresh`). A Custom-mode route is a named egress that rules point at (`route.*`, the `custom_*` tools).

## Tests

- `go test -race ./...` runs everything against `internal/fakeapp`, an in-process fake socket that records each request line.
- Every tool has a row in `cases` in `internal/tools/tools_test.go` pinning the exact request line it sends. `TestEveryToolSendsTheContractRequest` fails for a tool without a row.
- `TestLiveReadOnly` runs against the real app when its socket exists on this Mac. It sends read verbs only. A test that talks to the real socket stays read-only: a mutating verb changes the live routing of the machine.
- A new test earns trust by going red once: break the line it covers, watch it fail, restore the line by hand.

## Releases

A `v*.*.*` tag runs `.github/workflows/release.yml`: GoReleaser builds darwin amd64 and arm64 archives, then npm publishes the wrapper (`run.js`, `postinstall.js`) with trusted publishing. `postinstall.js` downloads `vpn-bypass-mcp_<version>_darwin_<arch>.tar.gz`, so the archive name in `.goreleaser.yaml` and that script change together. `package.json` and `server.json` carry the version too, and the `check-version` job refuses a tag they disagree with. The MCP Registry job (`.github/workflows/mcp-registry.yml`) waits for npm to serve the new version before it publishes, through `.github/scripts/wait-for-npm.sh`; CI runs `.github/scripts/wait-for-npm_test.sh` against a fake `npm`.
