#!/usr/bin/env node
"use strict";

// Starts the Go binary. stdin and stdout carry the MCP protocol, so they pass
// straight through. When the install script did not run (npm 12, pnpm and bun
// skip install scripts by default), the binary is downloaded and verified here
// on the first start, with the same code and checks as postinstall.js.

const { spawn } = require("child_process");
const os = require("os");
const { install } = require("./postinstall.js");

async function main() {
  if (process.platform !== "darwin") {
    console.error("vpn-bypass-mcp runs on macOS only: VPN Bypass is a macOS app.");
    process.exit(1);
  }

  let binPath;
  try {
    binPath = await install();
  } catch (err) {
    console.error(`vpn-bypass-mcp: could not install the binary: ${err.message}`);
    process.exit(1);
  }

  const child = spawn(binPath, process.argv.slice(2), { stdio: "inherit" });

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
}

main();
