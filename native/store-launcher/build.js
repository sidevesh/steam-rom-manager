const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const architectures = {
  ia32: "386",
  x64: "amd64",
  arm64: "arm64",
};

const electronArch = process.argv[2];
const goArch = architectures[electronArch];

if (!goArch) {
  console.error(
    `Unsupported Electron architecture: ${electronArch || "(missing)"}. ` +
      `Expected one of ${Object.keys(architectures).join(", ")}.`,
  );
  process.exit(1);
}

const moduleDir = __dirname;
const packageVersion = require(
  path.join(moduleDir, "..", "..", "package.json"),
).version;
const artifactPath = path.join(
  moduleDir,
  "dist",
  `srm-store-launcher-windows-${goArch}.exe`,
);
const outputDir = path.join(moduleDir, "dist", electronArch);
const outputPath = path.join(outputDir, "srm-store-launcher.exe");
fs.mkdirSync(outputDir, { recursive: true });

const result = spawnSync(
  "go",
  [
    "build",
    "-trimpath",
    `-ldflags=-s -w -H=windowsgui -X main.version=${packageVersion}`,
    "-o",
    artifactPath,
    "./cmd/srm-store-launcher",
  ],
  {
    cwd: moduleDir,
    env: {
      ...process.env,
      CGO_ENABLED: "0",
      GOOS: "windows",
      GOARCH: goArch,
    },
    stdio: "inherit",
  },
);

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

if (result.status !== 0) {
  process.exit(result.status ?? 1);
}

fs.copyFileSync(artifactPath, outputPath);
console.log(
  `Built ${artifactPath} and staged ${outputPath} ` +
    `(${electronArch} -> windows/${goArch})`,
);
