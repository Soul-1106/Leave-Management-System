import { existsSync, readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { resolve } from "node:path";

const root = resolve(import.meta.dirname, "..");
const errors = [];
const warnings = [];

function fail(message) {
  errors.push(message);
}

function commandVersion(command, args) {
  const result = spawnSync(command, args, {
    cwd: root,
    encoding: "utf8",
  });
  return result.status === 0
    ? (result.stdout || result.stderr).trim()
    : null;
}

function runInstall(command, args, label, cwd = root) {
  console.log(`  Installing ${label}...`);
  const result = spawnSync(command, args, {
    cwd,
    stdio: "inherit",
  });
  if (result.status !== 0) {
    fail(`Unable to install ${label}. Check your internet connection and try again.`);
    return false;
  }
  console.log(`  ${label}: installed`);
  return true;
}

function readEnv(path) {
  const values = {};
  const lines = readFileSync(path, "utf8").split(/\r?\n/);

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;

    const separator = trimmed.indexOf("=");
    if (separator < 1) continue;

    const key = trimmed.slice(0, separator).trim();
    let value = trimmed.slice(separator + 1).trim();
    if (
      value.length >= 2 &&
      ((value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'")))
    ) {
      value = value.slice(1, -1);
    }
    values[key] = value;
  }

  return values;
}

console.log("\nRunning development preflight checks...");

const nodeMajor = Number(process.versions.node.split(".")[0]);
console.log(`  Node.js: v${process.versions.node}`);
if (nodeMajor < 22) fail("Node.js 22 or newer is required.");

const goVersion = commandVersion("go", ["version"]);
if (goVersion) {
  console.log(`  Go: ${goVersion.replace(/^go version /, "")}`);
} else {
  fail("Go is not installed or is not available on PATH.");
}

if (!existsSync(resolve(root, "backend/go.mod"))) {
  fail("Backend Go module is missing.");
}
if (!existsSync(resolve(root, "frontend/package.json"))) {
  fail("Frontend package.json is missing.");
}

const npmCli = resolve(
  process.execPath,
  "..",
  "node_modules",
  "npm",
  "bin",
  "npm-cli.js",
);
const npmCommand =
  process.platform === "win32" && existsSync(npmCli) ? process.execPath : "npm";
const npmArguments =
  process.platform === "win32" && existsSync(npmCli) ? [npmCli] : [];
if (!existsSync(resolve(root, "node_modules/concurrently"))) {
  runInstall(npmCommand, [...npmArguments, "install"], "root dependencies");
}
if (
  existsSync(resolve(root, "frontend/package.json")) &&
  !existsSync(resolve(root, "frontend/node_modules"))
) {
  runInstall(
    npmCommand,
    [...npmArguments, "install", "--prefix", "frontend"],
    "frontend dependencies",
  );
}
if (
  goVersion &&
  existsSync(resolve(root, "backend/go.mod")) &&
  !existsSync(resolve(root, "backend/.gomodcache/github.com/lib/pq@v1.10.9"))
) {
  runInstall(
    "go",
    ["mod", "download"],
    "backend Go dependencies",
    resolve(root, "backend"),
  );
}

const envPath = resolve(root, ".env");
if (!existsSync(envPath)) {
  fail("Root .env file is missing.");
} else {
  const env = readEnv(envPath);
  const requiredVariables = [
    "VITE_SUPABASE_URL",
    "VITE_SUPABASE_ANON_KEY",
    "SUPABASE_DB_URL",
    "SUPABASE_SERVICE_ROLE_KEY",
  ];

  for (const key of requiredVariables) {
    const value = env[key];
    if (!value) {
      fail(`${key} is missing or empty in .env.`);
    } else if (/YOUR[-_ ]|PLACEHOLDER|CHANGE[-_ ]?ME/i.test(value)) {
      fail(`${key} still contains a placeholder value.`);
    }
  }

  if (env.VITE_SUPABASE_URL) {
    try {
      const url = new URL(env.VITE_SUPABASE_URL);
      if (url.protocol !== "https:" || !url.hostname.endsWith(".supabase.co")) {
        warnings.push("VITE_SUPABASE_URL does not look like a Supabase HTTPS project URL.");
      }
    } catch {
      fail("VITE_SUPABASE_URL is not a valid URL.");
    }
  }

  if (env.SUPABASE_DB_URL) {
    try {
      const url = new URL(env.SUPABASE_DB_URL);
      if (!["postgres:", "postgresql:"].includes(url.protocol)) {
        fail("SUPABASE_DB_URL must use the postgresql:// protocol.");
      }
      if (!url.username || !url.password || !url.hostname) {
        fail("SUPABASE_DB_URL must include a username, password, and hostname.");
      }
    } catch {
      fail(
        "SUPABASE_DB_URL is invalid. URL-encode special characters in the database password.",
      );
    }
  }

  console.log("  Environment: required variables found (values hidden)");
}

for (const warning of warnings) console.warn(`  WARNING: ${warning}`);

if (errors.length) {
  console.error("\nPreflight failed:");
  for (const error of errors) console.error(`  - ${error}`);
  console.error("\nFix the items above, then run npm run dev again.\n");
  process.exit(1);
}

console.log("Preflight passed. Starting backend and frontend...\n");
