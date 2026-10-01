#!/usr/bin/env bash
# Tests wait-for-npm.sh against a fake npm on PATH, so it needs no network.
# The fake answers "npm view <pkg>@<version> version" with the version once
# FAKE_NPM_READY_AFTER calls have answered nothing. Until then it prints
# nothing and exits FAKE_NPM_MISSING_EXIT: 0 like npm 10, 1 (with E404 on
# stderr) like npm 11.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/npm" <<'FAKE'
#!/usr/bin/env bash
n=$(( $(cat "$FAKE_NPM_CALLS" 2>/dev/null || echo 0) + 1 ))
echo "$n" > "$FAKE_NPM_CALLS"
if [ "$n" -gt "$FAKE_NPM_READY_AFTER" ]; then
  echo "${2#*@}"
  exit 0
fi
if [ "$FAKE_NPM_MISSING_EXIT" = 1 ]; then
  echo "npm error code E404" >&2
  exit 1
fi
FAKE
chmod +x "$tmp/npm"

fail=0
run() { # name, ready-after, missing-exit, want-exit, want-calls, want-output, [args]
  local name="$1" ready="$2" missing_exit="$3" want_exit="$4" want_calls="$5" want_out="$6" code=0
  shift 6
  [ $# -gt 0 ] || set -- vpn-bypass-mcp 1.2.3
  rm -f "$tmp/calls"
  PATH="$tmp:$PATH" FAKE_NPM_CALLS="$tmp/calls" FAKE_NPM_READY_AFTER="$ready" \
    FAKE_NPM_MISSING_EXIT="$missing_exit" \
    WAIT_TIMEOUT=2 WAIT_INTERVAL=1 \
    "$here/wait-for-npm.sh" "$@" > "$tmp/out" 2>&1 || code=$?
  local calls; calls=$(cat "$tmp/calls" 2>/dev/null || echo 0)
  if [ "$code" != "$want_exit" ] || [ "$calls" != "$want_calls" ] || ! grep -qF -- "$want_out" "$tmp/out"; then
    echo "FAIL $name: exit $code (want $want_exit), npm calls $calls (want $want_calls), want output containing: $want_out"
    sed 's/^/  | /' "$tmp/out"
    fail=1
  else
    echo "ok   $name"
  fi
}

served="npm serves vpn-bypass-mcp@1.2.3"
gave_up="::error::npm still does not serve vpn-bypass-mcp@1.2.3"
usage="::error::usage: wait-for-npm.sh"

run "served at once"                   0  0 0 1 "$served"
run "served on the 3rd check (npm 10)" 2  0 0 3 "$served"
run "served on the 3rd check (npm 11)" 2  1 0 3 "$served"
run "never served, times out (npm 10)" 99 0 1 3 "$gave_up"
run "never served, times out (npm 11)" 99 1 1 3 "$gave_up"
run "null version from jq, no polling" 0  0 2 0 "$usage" vpn-bypass-mcp null
run "missing version, no polling"      0  0 2 0 "$usage" vpn-bypass-mcp

exit "$fail"
