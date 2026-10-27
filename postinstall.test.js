"use strict";

// Tests for the binary download in postinstall.js (run.js uses the same code).
// Run with: node --test. No network: every download goes through a fake get.

const test = require("node:test");
const assert = require("node:assert");
const crypto = require("crypto");
const { EventEmitter } = require("events");
const { execFileSync } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { Readable } = require("stream");

const { install, download, expectedSum, assetName } = require("./postinstall.js");
const VERSION = require("./package.json").version;

const sha = (buf) => crypto.createHash("sha256").update(buf).digest("hex");

// fakeGet answers each URL from routes: { status, location, body }.
function fakeGet(routes, seen = []) {
  return (url, _opts, cb) => {
    seen.push(url);
    const req = new EventEmitter();
    req.destroy = () => {};
    const r = routes[url] || { status: 404 };
    setImmediate(() => {
      const res = Readable.from(r.body ? [r.body] : []);
      res.statusCode = r.status;
      res.headers = r.location ? { location: r.location } : {};
      cb(res);
    });
    return req;
  };
}

function tmpdir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), "vpnb-npm-"));
}

test("expectedSum reads the line for the asset and nothing else", () => {
  const a = "a".repeat(64);
  const b = "b".repeat(64);
  const sums = `${a}  vpn-bypass-mcp_1.0.0_darwin_amd64.tar.gz\n${b}  vpn-bypass-mcp_1.0.0_darwin_arm64.tar.gz\n`;
  assert.strictEqual(expectedSum(sums, "vpn-bypass-mcp_1.0.0_darwin_arm64.tar.gz"), b);
  assert.throws(() => expectedSum(sums, "darwin_arm64.tar.gz"), /No sha256/);
  assert.throws(() => expectedSum("nothex  x.tar.gz\n", "x.tar.gz"), /No sha256/);
});

test("assetName refuses anything but macOS on x64 or arm64", () => {
  assert.strictEqual(assetName("darwin", "arm64"), `vpn-bypass-mcp_${VERSION}_darwin_arm64.tar.gz`);
  assert.throws(() => assetName("linux", "x64"), /macOS only/);
  assert.throws(() => assetName("darwin", "ia32"), /Unsupported CPU/);
});

test("download streams to disk and returns the sha256", async () => {
  const dir = tmpdir();
  const body = Buffer.from("archive bytes");
  const url = "https://github.com/x/releases/download/v1/a.tar.gz";
  const asset = "https://release-assets.githubusercontent.com/a";
  const got = await download(url, path.join(dir, "a"), {
    get: fakeGet({ [url]: { status: 302, location: asset }, [asset]: { status: 200, body } }),
  });
  assert.strictEqual(got, sha(body));
  assert.deepStrictEqual(fs.readFileSync(path.join(dir, "a")), body);
});

test("download refuses plain HTTP, a redirect to HTTP, and other hosts", async () => {
  const dir = tmpdir();
  const url = "https://github.com/x/a.tar.gz";
  await assert.rejects(download("http://github.com/x/a.tar.gz", path.join(dir, "1")), /Refusing/);
  await assert.rejects(
    download(url, path.join(dir, "2"), { get: fakeGet({ [url]: { status: 302, location: "http://github.com/x/a.tar.gz" } }) }),
    /Refusing/,
  );
  await assert.rejects(
    download(url, path.join(dir, "3"), { get: fakeGet({ [url]: { status: 302, location: "https://evil.example/a.tar.gz" } }) }),
    /Refusing/,
  );
});

test("download stops at the size cap", async () => {
  const dir = tmpdir();
  const url = "https://github.com/x/a.tar.gz";
  await assert.rejects(
    download(url, path.join(dir, "a"), { get: fakeGet({ [url]: { status: 200, body: Buffer.alloc(2048) } }), maxBytes: 1024 }),
    /larger than 1024 bytes/,
  );
});

// A package folder with checksums.txt listing sum for the arm64 archive.
function packageDir(sum) {
  const dir = tmpdir();
  fs.writeFileSync(path.join(dir, "checksums.txt"), `${sum}  ${assetName("darwin", "arm64")}\n`);
  return dir;
}

// A real tar.gz holding a file named vpn-bypass-mcp.
function archive() {
  const src = tmpdir();
  fs.writeFileSync(path.join(src, "vpn-bypass-mcp"), "#!/bin/sh\necho ok\n");
  const out = path.join(tmpdir(), "a.tar.gz");
  execFileSync("tar", ["-czf", out, "-C", src, "vpn-bypass-mcp"]);
  return fs.readFileSync(out);
}

const fetchFrom = (buf) => async (_url, dest) => {
  fs.writeFileSync(dest, buf);
  return sha(buf);
};

test("install unpacks a binary whose sha256 matches the bundled checksums.txt", async () => {
  const buf = archive();
  const dir = packageDir(sha(buf));
  const bin = await install({ dir, platform: "darwin", arch: "arm64", fetch: fetchFrom(buf), log: () => {} });
  assert.strictEqual(bin, path.join(dir, "bin", "vpn-bypass-mcp"));
  assert.strictEqual(fs.statSync(bin).mode & 0o777, 0o755);
  assert.deepStrictEqual(fs.readdirSync(path.join(dir, "bin")), ["vpn-bypass-mcp"]);
});

test("install refuses a binary whose sha256 does not match, and leaves nothing behind", async () => {
  const dir = packageDir("c".repeat(64));
  await assert.rejects(
    install({ dir, platform: "darwin", arch: "arm64", fetch: fetchFrom(archive()), log: () => {} }),
    /Checksum mismatch/,
  );
  assert.deepStrictEqual(fs.readdirSync(path.join(dir, "bin")), []);
});

test("install refuses to download without a bundled checksums.txt", async () => {
  let fetched = false;
  await assert.rejects(
    install({ dir: tmpdir(), platform: "darwin", arch: "arm64", fetch: async () => { fetched = true; }, log: () => {} }),
    /checksums.txt is missing/,
  );
  assert.strictEqual(fetched, false);
});
