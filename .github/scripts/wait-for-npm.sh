#!/usr/bin/env bash
# Usage: wait-for-npm.sh <package> <version>
#
# Waits until npm serves <package>@<version>. npm accepts a publish with a 202
# and can take minutes to serve the new version (about four for v0.1.1), and
# the MCP Registry refuses server.json with a 404 until it does.
#
# WAIT_TIMEOUT (seconds, default 900) caps the wait; WAIT_INTERVAL (seconds,
# default 15) sets the pause between checks.
set -euo pipefail

pkg="${1:-}"
version="${2:-}"
# jq -r prints "null" for a missing key; polling for null@null would only
# fail 15 minutes later.
for arg in "$pkg" "$version"; do
  if [ -z "$arg" ] || [ "$arg" = null ]; then
    echo "::error::usage: wait-for-npm.sh <package> <version> (got '$pkg' '$version'); check server.json"
    exit 2
  fi
done
timeout="${WAIT_TIMEOUT:-900}"
interval="${WAIT_INTERVAL:-15}"
waited=0

while :; do
  # A version npm does not serve yet prints nothing on stdout: npm 10 exits 0,
  # npm 11 exits 1 with E404. Both mean wait.
  got=$(npm view "$pkg@$version" version 2>/dev/null || true)
  if [ -n "$got" ]; then
    echo "npm serves $pkg@$version (waited ${waited}s)"
    exit 0
  fi
  if [ "$waited" -ge "$timeout" ]; then
    echo "::error::npm still does not serve $pkg@$version after ${waited}s. Check that the npm-publish job published it, then rerun this job."
    exit 1
  fi
  echo "npm does not serve $pkg@$version yet (waited ${waited}s of ${timeout}s)"
  sleep "$interval"
  waited=$((waited + interval))
done
