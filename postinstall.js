#!/usr/bin/env node
"use strict";

// Downloads the release binary that matches this package version and this
// Mac's CPU, checks it against the release's checksums.txt, and unpacks it to
// bin/. Messages go to stderr.

const { execFileSync } = require("child_process");
const crypto = require("crypto");
const fs = require("fs");
const path = require("path");
const https = require("https");

const VERSION = require("./package.json").version;
const REPO = "GeiserX/vpn-bypass-mcp";
const BIN_NAME = "vpn-bypass-mcp";
const BIN_DIR = path.join(__dirname, "bin");
const BIN_PATH = path.join(BIN_DIR, BIN_NAME);

const MAX_REDIRECTS = 10;
const REQUEST_TIMEOUT_MS = 30_000;

function assetName() {
  if (process.platform !== "darwin") {
    throw new Error(`vpn-bypass-mcp runs on macOS only, not ${process.platform}: VPN Bypass is a macOS app`);
  }
  const arch = { x64: "amd64", arm64: "arm64" }[process.arch];
  if (!arch) {
    throw new Error(`Unsupported CPU: ${process.arch}`);
  }
  return `${BIN_NAME}_${VERSION}_darwin_${arch}.tar.gz`;
}

function download(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > MAX_REDIRECTS) {
      return reject(new Error(`Too many redirects (>${MAX_REDIRECTS})`));
    }
    if (!url.startsWith("https://")) {
      return reject(new Error(`Refusing a non-HTTPS URL: ${url}`));
    }
    const req = https.get(url, { timeout: REQUEST_TIMEOUT_MS }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        return download(new URL(res.headers.location, url).toString(), redirects + 1).then(resolve, reject);
      }
      if (res.statusCode !== 200) {
        res.resume();
        return reject(new Error(`Download failed: HTTP ${res.statusCode} for ${url}`));
      }
      const chunks = [];
      res.on("data", (chunk) => chunks.push(chunk));
      res.on("end", () => resolve(Buffer.concat(chunks)));
      res.on("error", reject);
    });
    req.on("timeout", () => {
      req.destroy();
      reject(new Error(`Request timed out after ${REQUEST_TIMEOUT_MS} ms`));
    });
    req.on("error", reject);
  });
}

async function verify(buffer, name, base) {
  const sums = (await download(`${base}/checksums.txt`)).toString("utf-8");
  const line = sums.split("\n").find((l) => l.trim().endsWith(` ${name}`));
  if (!line) {
    throw new Error(`No checksum for ${name} in checksums.txt`);
  }
  const expected = line.trim().split(/\s+/)[0];
  const actual = crypto.createHash("sha256").update(buffer).digest("hex");
  if (actual !== expected) {
    throw new Error(`Checksum mismatch for ${name}: expected ${expected}, got ${actual}`);
  }
}

function extract(buffer) {
  fs.mkdirSync(BIN_DIR, { recursive: true });
  const tmp = path.join(BIN_DIR, "download.tar.gz");
  fs.writeFileSync(tmp, buffer);
  try {
    execFileSync("tar", ["-xzf", tmp, "-C", BIN_DIR, BIN_NAME], { stdio: "ignore" });
  } finally {
    fs.unlinkSync(tmp);
  }
  fs.chmodSync(BIN_PATH, 0o755);
}

async function main() {
  if (fs.existsSync(BIN_PATH)) {
    console.error("vpn-bypass-mcp binary already present, skipping the download.");
    return;
  }
  const name = assetName();
  const base = `https://github.com/${REPO}/releases/download/v${VERSION}`;
  console.error(`Downloading vpn-bypass-mcp v${VERSION} (${name})...`);
  const buffer = await download(`${base}/${name}`);
  await verify(buffer, name, base);
  extract(buffer);
  console.error(`Installed vpn-bypass-mcp to ${BIN_PATH}`);
}

main().catch((err) => {
  console.error(`Failed to install vpn-bypass-mcp: ${err.message}`);
  process.exit(1);
});
