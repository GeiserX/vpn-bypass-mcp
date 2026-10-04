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

A `v*.*.*` tag runs `.github/workflows/release.yml`: GoReleaser builds darwin amd64 and arm64 binaries, signs them with a Developer ID certificate and notarizes them with Apple (secrets `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_ISSUER_ID`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_KEY`; a tag build missing any of them stops before GoReleaser), archives them, then npm publishes the wrapper (`run.js`, `postinstall.js`) with trusted publishing. `postinstall.js` downloads `vpn-bypass-mcp_<version>_darwin_<arch>.tar.gz`, so the archive name in `.goreleaser.yaml` and that script change together. `package.json` and `server.json` carry the version too, and the `check-version` job refuses a tag they disagree with. The MCP Registry job (`.github/workflows/mcp-registry.yml`) waits for npm to serve the new version before it publishes, through `.github/scripts/wait-for-npm.sh`; CI runs `.github/scripts/wait-for-npm_test.sh` against a fake `curl`.

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:970c3bf2 -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.

## Agent Context Profiles

The managed Beads block is task-tracking guidance, not permission to override repository, user, or orchestrator instructions.

- **Conservative (default)**: Use `bd` for task tracking. Do not run git commits, git pushes, or Dolt remote sync unless explicitly asked. At handoff, report changed files, validation, and suggested next commands.
- **Minimal**: Keep tool instruction files as pointers to `bd prime`; use the same conservative git policy unless active instructions say otherwise.
- **Team-maintainer**: Only when the repository explicitly opts in, agents may close beads, run quality gates, commit, and push as part of session close. A current "do not commit" or "do not push" instruction still wins.

## Session Completion

This protocol applies when ending a Beads implementation workflow. It is subordinate to explicit user, repository, and orchestrator instructions.

1. **File issues for remaining work** - Create beads for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **Handle git/sync by active profile**:
   ```bash
   # Conservative/minimal/default: report status and proposed commands; wait for approval.
   git status

   # Team-maintainer opt-in only, unless current instructions forbid it:
   git pull --rebase
   bd dolt push
   git push
   git status
   ```
5. **Hand off** - Summarize changes, validation, issue status, and any blocked sync/commit/push step

**Critical rules:**
- Explicit user or orchestrator instructions override this Beads block.
- Do not commit or push without clear authority from the active profile or the current user request.
- If a required sync or push is blocked, stop and report the exact command and error.
<!-- END BEADS INTEGRATION -->

## Where the tracker syncs

This repo is public, so its tracker syncs only to the private remote named by `sync.remote` in `.beads/config.yaml`. The block above says sync uses "your git remote". Here that never means this GitHub repo. Don't add it as a Dolt remote and don't push `refs/dolt/*` to it.

<!-- BEGIN BEADS CODEX SETUP: generated by bd setup codex -->
## Beads Issue Tracker

Use Beads (`bd`) for durable task tracking in repositories that include it. Use the `beads` skill at `.agents/skills/beads/SKILL.md` (project install) or `~/.agents/skills/beads/SKILL.md` (global install) for Beads workflow guidance, then use the `bd` CLI for issue operations.

### Quick Reference

```bash
bd ready                # Find available work
bd show <id>            # View issue details
bd update <id> --claim  # Claim work
bd close <id>           # Complete work
bd prime                # Refresh Beads context
```

### Rules

- Use `bd` for all task tracking; do not create markdown TODO lists.
- Run `bd prime` when Beads context is missing or stale. Codex 0.129.0+ can load Beads context automatically through native hooks; use `/hooks` to inspect or toggle them.
- Keep persistent project memory in Beads via `bd remember`; do not create ad hoc memory files.

**Architecture in one line:** issues live in a local Dolt DB; sync uses `refs/dolt/data` on your git remote; `.beads/issues.jsonl` is a passive export. See https://github.com/gastownhall/beads/blob/main/docs/SYNC_CONCEPTS.md for details and anti-patterns.
<!-- END BEADS CODEX SETUP -->
