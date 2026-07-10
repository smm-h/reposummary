const assert = require("assert");
const fs = require("fs");
const path = require("path");

// Platform mapping (must match install.js)
const PLATFORM_MAP = { linux: "linux", darwin: "darwin", win32: "windows" };
const ARCH_MAP = { x64: "amd64", arm64: "arm64" };

// archiveExt / binName mirror install.js: Windows ships a .zip and a .exe.
function archiveExt(os) {
  return os === "windows" ? "zip" : "tar.gz";
}

function binName(os) {
  return os === "windows" ? "reposummary.exe" : "reposummary";
}

function getDownloadUrl(version, platform, arch) {
  const os = PLATFORM_MAP[platform];
  const goarch = ARCH_MAP[arch];
  if (!os) throw new Error(`Unsupported platform: ${platform}`);
  if (!goarch) throw new Error(`Unsupported architecture: ${arch}`);
  return `https://github.com/smm-h/reposummary/releases/download/v${version}/reposummary_${version}_${os}_${goarch}.${archiveExt(os)}`;
}

// --- Platform mapping ---
assert.strictEqual(getDownloadUrl("0.1.1", "linux", "x64"),
  "https://github.com/smm-h/reposummary/releases/download/v0.1.1/reposummary_0.1.1_linux_amd64.tar.gz");
assert.strictEqual(getDownloadUrl("0.1.1", "darwin", "arm64"),
  "https://github.com/smm-h/reposummary/releases/download/v0.1.1/reposummary_0.1.1_darwin_arm64.tar.gz");
assert.strictEqual(getDownloadUrl("0.1.1", "linux", "arm64"),
  "https://github.com/smm-h/reposummary/releases/download/v0.1.1/reposummary_0.1.1_linux_arm64.tar.gz");
assert.strictEqual(getDownloadUrl("0.1.1", "darwin", "x64"),
  "https://github.com/smm-h/reposummary/releases/download/v0.1.1/reposummary_0.1.1_darwin_amd64.tar.gz");

// --- Windows: .zip archive, both arches ---
assert.strictEqual(getDownloadUrl("0.1.1", "win32", "x64"),
  "https://github.com/smm-h/reposummary/releases/download/v0.1.1/reposummary_0.1.1_windows_amd64.zip");
assert.strictEqual(getDownloadUrl("0.1.1", "win32", "arm64"),
  "https://github.com/smm-h/reposummary/releases/download/v0.1.1/reposummary_0.1.1_windows_arm64.zip");

// --- Platform-map coverage ---
assert.strictEqual(PLATFORM_MAP.linux, "linux");
assert.strictEqual(PLATFORM_MAP.darwin, "darwin");
assert.strictEqual(PLATFORM_MAP.win32, "windows");

// --- Archive extension / binary name selection ---
assert.strictEqual(archiveExt("linux"), "tar.gz");
assert.strictEqual(archiveExt("darwin"), "tar.gz");
assert.strictEqual(archiveExt("windows"), "zip");
assert.strictEqual(binName("linux"), "reposummary");
assert.strictEqual(binName("windows"), "reposummary.exe");

// --- Unsupported platforms / arch ---
assert.throws(() => getDownloadUrl("0.1.1", "freebsd", "x64"), /Unsupported platform/);
assert.throws(() => getDownloadUrl("0.1.1", "linux", "ia32"), /Unsupported architecture/);

// --- Windows URLs end in .zip; unix in .tar.gz ---
assert.ok(getDownloadUrl("0.1.1", "win32", "x64").endsWith(".zip"));
assert.ok(getDownloadUrl("0.1.1", "linux", "x64").endsWith(".tar.gz"));
assert.ok(getDownloadUrl("0.1.1", "darwin", "arm64").endsWith(".tar.gz"));

// --- Agreement with goreleaser's name_template for Windows zips ---
// The wrapper filenames must match exactly what goreleaser produces, or the
// download URLs will 404. Render goreleaser's template from .goreleaser.yml
// (the source of truth) and compare against the wrapper's generated names.
const goreleaser = fs.readFileSync(path.join(__dirname, "..", ".goreleaser.yml"), "utf8");

const tmplMatch = goreleaser.match(/name_template:\s*"([^"]+)"/);
assert.ok(tmplMatch, "goreleaser name_template not found in .goreleaser.yml");
const nameTemplate = tmplMatch[1];

// Windows must be overridden to the zip format.
assert.ok(/goos:\s*windows/.test(goreleaser) && /formats:\s*\[zip\]/.test(goreleaser),
  "goreleaser is missing the windows -> zip format override");

function renderGoreleaserName(projectName, version, os, arch) {
  return nameTemplate
    .replace(/\{\{\s*\.ProjectName\s*\}\}/g, projectName)
    .replace(/\{\{\s*\.Version\s*\}\}/g, version)
    .replace(/\{\{\s*\.Os\s*\}\}/g, os)
    .replace(/\{\{\s*\.Arch\s*\}\}/g, arch);
}

for (const [nodeArch, goarch] of [["x64", "amd64"], ["arm64", "arm64"]]) {
  const filename = getDownloadUrl("0.1.1", "win32", nodeArch).split("/").pop();
  const expected = renderGoreleaserName("reposummary", "0.1.1", "windows", goarch) + ".zip";
  assert.strictEqual(filename, expected,
    `wrapper windows filename ${filename} disagrees with goreleaser ${expected}`);
}

console.log("All npm wrapper tests passed");
