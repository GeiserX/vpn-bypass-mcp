#!/usr/bin/env node
"use strict";

// Downloads the release binary that matches this package version and this
// Mac's CPU, checks its sha256 against the checksums.txt that the release
// workflow put inside this npm package (so npm's provenance covers the hashes;
// nothing is trusted from the release page itself), and unpacks it to bin/.
//
// npm 12, pnpm and bun skip install scripts by default, so run.js calls
// install() on the first start when bin/ is still empty. Messages go to
// stderr: stdout carries the MCP protocol once the server runs.

const { execFileSync } = require("child_process");
const crypto = require("crypto");
const fs = require("fs");
const path = require("path");
const https = require("https");

const VERSION = require("./package.json").version;
const REPO = "GeiserX/vpn-bypass-mcp";
const BIN_NAME = "vpn-bypass-mcp";

const MAX_REDIRECTS = 10;
const REQUEST_TIMEOUT_MS = 30_000;
// A release archive is a few MB; anything far bigger is not ours.
const MAX_BYTES = 64 * 1024 * 1024;

function assetName(platform = process.platform, arch = process.arch) {
  if (platform !== "darwin") {
    throw new Error(`vpn-bypass-mcp runs on macOS only, not ${platform}: VPN Bypass is a macOS app`);
  }
  const goArch = { x64: "amd64", arm64: "arm64" }[arch];
  if (!goArch) {
    throw new Error(`Unsupported CPU: ${arch}`);
  }
  return `${BIN_NAME}_${VERSION}_darwin_${goArch}.tar.gz`;
}

// expectedSum returns the sha256 that checksums.txt lists for name.
function expectedSum(sums, name) {
  const line = sums.split("\n").find((l) => l.trim().split(/\s+/)[1] === name);
  const sum = line && line.trim().split(/\s+/)[0];
  if (!sum || !/^[0-9a-f]{64}$/.test(sum)) {
    throw new Error(`No sha256 for ${name} in the bundled checksums.txt`);
  }
  return sum;
}

function bundledSums(dir) {
  const file = path.join(dir, "checksums.txt");
  if (!fs.existsSync(file)) {
    throw new Error("checksums.txt is missing from this package, so the download cannot be verified; reinstall vpn-bypass-mcp from npm");
  }
  return fs.readFileSync(file, "utf-8");
}

// GitHub serves release assets from github.com and redirects to its asset hosts.
function allowedURL(url) {
  let u;
  try {
    u = new URL(url);
  } catch {
    return false;
  }
  return u.protocol === "https:" && (u.hostname === "github.com" || u.hostname.endsWith(".githubusercontent.com"));
}

// download streams url to dest, following redirects to GitHub hosts over
// HTTPS only, stopping at maxBytes, and resolves with the sha256 of the bytes.
function download(url, dest, { get = https.get, maxBytes = MAX_BYTES, redirects = 0 } = {}) {
  return new Promise((resolve, reject) => {
    if (redirects > MAX_REDIRECTS) {
      return reject(new Error(`Too many redirects (>${MAX_REDIRECTS})`));
    }
    if (!allowedURL(url)) {
      return reject(new Error(`Refusing to download from ${url}: only https://github.com release assets are allowed`));
    }
    const req = get(url, { timeout: REQUEST_TIMEOUT_MS }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        const next = new URL(res.headers.location, url).toString();
        return download(next, dest, { get, maxBytes, redirects: redirects + 1 }).then(resolve, reject);
      }
      if (res.statusCode !== 200) {
        res.resume();
        return reject(new Error(`Download failed: HTTP ${res.statusCode} for ${url}`));
      }
      if (Number(res.headers["content-length"]) > maxBytes) {
        res.resume();
        return reject(new Error(`The download is larger than ${maxBytes} bytes; refusing it`));
      }
      const hash = crypto.createHash("sha256");
      const out = fs.createWriteStream(dest, { flags: "wx", mode: 0o600 });
      let size = 0;
      let failed = false;
      const fail = (err) => {
        if (failed) return;
        failed = true;
        res.destroy();
        out.destroy();
        reject(err);
      };
      res.on("data", (chunk) => {
        size += chunk.length;
        if (size > maxBytes) {
          return fail(new Error(`The download is larger than ${maxBytes} bytes; refusing it`));
        }
        hash.update(chunk);
      });
      res.on("error", fail);
      out.on("error", fail);
      out.on("finish", () => {
        if (!failed) resolve(hash.digest("hex"));
      });
      res.pipe(out);
    });
    req.on("timeout", () => {
      req.destroy();
      reject(new Error(`Request timed out after ${REQUEST_TIMEOUT_MS} ms`));
    });
    req.on("error", reject);
  });
}

// install puts the verified binary at <dir>/bin/vpn-bypass-mcp and returns
// its path. It writes into a private temporary folder first and renames the
// binary into place, so two starts racing each other never see half a file.
async function install({
  dir = __dirname,
  platform = process.platform,
  arch = process.arch,
  fetch = download,
  log = (msg) => console.error(msg),
} = {}) {
  const binDir = path.join(dir, "bin");
  const binPath = path.join(binDir, BIN_NAME);
  if (fs.existsSync(binPath)) {
    return binPath;
  }
  const name = assetName(platform, arch);
  const expected = expectedSum(bundledSums(dir), name);
  fs.mkdirSync(binDir, { recursive: true });
  const tmp = fs.mkdtempSync(path.join(binDir, ".download-"));
  try {
    const archive = path.join(tmp, name);
    log(`Downloading vpn-bypass-mcp v${VERSION} (${name})...`);
    const actual = await fetch(`https://github.com/${REPO}/releases/download/v${VERSION}/${name}`, archive);
    if (actual !== expected) {
      throw new Error(`Checksum mismatch for ${name}: expected ${expected}, got ${actual}`);
    }
    execFileSync("tar", ["-xzf", archive, "-C", tmp, BIN_NAME], { stdio: "ignore" });
    fs.chmodSync(path.join(tmp, BIN_NAME), 0o755);
    fs.renameSync(path.join(tmp, BIN_NAME), binPath);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
  log(`Installed vpn-bypass-mcp to ${binPath}`);
  return binPath;
}

module.exports = { install, download, expectedSum, assetName, MAX_BYTES };

if (require.main === module) {
  install().catch((err) => {
    console.error(`Failed to install vpn-bypass-mcp: ${err.message}`);
    process.exit(1);
  });
}
