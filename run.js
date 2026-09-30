#!/usr/bin/env node
"use strict";

// Starts the Go binary that postinstall.js downloaded. stdin and stdout carry
// the MCP protocol, so they pass straight through.

const { spawn } = require("child_process");
const os = require("os");
const path = require("path");
const fs = require("fs");

const BIN_PATH = path.join(__dirname, "bin", "vpn-bypass-mcp");

if (process.platform !== "darwin") {
  console.error("vpn-bypass-mcp runs on macOS only: VPN Bypass is a macOS app.");
  process.exit(1);
}

if (!fs.existsSync(BIN_PATH)) {
  console.error(`vpn-bypass-mcp binary not found at ${BIN_PATH}. Run: node ${path.join(__dirname, "postinstall.js")}`);
  process.exit(1);
}

const child = spawn(BIN_PATH, process.argv.slice(2), { stdio: "inherit" });

for (const signal of ["SIGINT", "SIGTERM", "SIGHUP"]) {
  process.on(signal, () => child.kill(signal));
}

child.on("error", (err) => {
  console.error(`Failed to start vpn-bypass-mcp: ${err.message}`);
  process.exit(1);
});

child.on("exit", (code, signal) => {
  if (signal) {
    process.exit(128 + (os.constants.signals[signal] || 0));
  }
  process.exit(code === null ? 1 : code);
});
