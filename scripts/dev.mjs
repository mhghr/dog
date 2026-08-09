import { readFileSync } from "node:fs";
import { spawn } from "node:child_process";
import { resolve } from "node:path";
import { createConnection } from "node:net";

const root = resolve(import.meta.dirname, "..");

function loadEnv(path) {
  const values = {};

  for (const rawLine of readFileSync(path, "utf8").split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;

    const separator = line.indexOf("=");
    if (separator < 1) continue;

    const key = line.slice(0, separator).trim();
    let value = line.slice(separator + 1).trim();
    if ((value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1);
    }
    values[key] = value;
  }

  return values;
}

const baseEnv = { ...process.env, ...loadEnv(resolve(root, ".env")) };
const pnpm = "pnpm";
const go = process.platform === "win32" ? "go.exe" : "go";

function commandForPlatform(command, args) {
  if (process.platform === "win32" && command === "pnpm") {
    return {
      command: process.env.ComSpec ?? "C:\\Windows\\System32\\cmd.exe",
      args: ["/d", "/s", "/c", [command, ...args].join(" ")],
    };
  }

  return { command, args };
}

function waitForPort(port, timeoutMs = 60000) {
  return new Promise((resolvePort, reject) => {
    const start = Date.now();
    const tryConnect = () => {
      if (Date.now() - start > timeoutMs) {
        reject(new Error(`timeout waiting for port ${port}`));
        return;
      }
      const socket = createConnection({ host: "127.0.0.1", port }, () => {
        socket.destroy();
        resolvePort();
      });
      socket.on("error", () => {
        setTimeout(tryConnect, 500);
      });
    };
    tryConnect();
  });
}

const children = [];
let stopping = false;

function runCleanupCommand(command, args) {
  return new Promise((resolveCleanup) => {
    const cleanup = spawn(command, args, {
      stdio: "ignore",
      windowsHide: true,
    });
    cleanup.once("error", () => resolveCleanup());
    cleanup.once("close", () => resolveCleanup());
  });
}

async function stop(exitCode = 0) {
  if (stopping) return;
  stopping = true;

  if (process.platform === "win32") {
    await Promise.all(
      children
        .filter((child) => child.pid)
        .map((child) =>
          runCleanupCommand("taskkill.exe", [
            "/pid",
            String(child.pid),
            "/t",
            "/f",
          ]),
        ),
    );
    const cleanupPorts = [
      "$owners = Get-NetTCPConnection -State Listen",
      "-LocalPort 2000,5000,5001,5002",
      "-ErrorAction SilentlyContinue",
      "| Select-Object -ExpandProperty OwningProcess -Unique;",
      "foreach ($ownerPid in $owners)",
      "{ Stop-Process -Id $ownerPid -Force -ErrorAction SilentlyContinue }",
    ].join(" ");
    await runCleanupCommand("powershell.exe", [
      "-NoProfile",
      "-NonInteractive",
      "-Command",
      cleanupPorts,
    ]);
  } else {
    for (const child of children) {
      child.kill("SIGTERM");
    }
  }

  process.exit(exitCode);
}

function launch(proc) {
  const runnable = commandForPlatform(proc.command, proc.args);
  const child = spawn(runnable.command, runnable.args, {
    cwd: root,
    env: { ...baseEnv, ...proc.env },
    stdio: "inherit",
  });
  children.push(child);
  child.on("error", (error) => {
    console.error(`[${proc.name}] failed to start:`, error.message);
    void stop(1);
  });
  child.on("exit", (code, signal) => {
    if (!stopping) {
      console.error(
        `[${proc.name}] exited (${signal ?? `code ${code ?? 1}`}); stopping dev stack.`,
      );
      void stop(code ?? 1);
    }
  });
  return child;
}

// ── Ordered startup: API first, then web + scheduler + worker in parallel ──

console.log("[dev] starting API (port 5000)...");
launch({
  name: "api",
  command: go,
  args: ["run", "./apps/api/cmd/api"],
  env: { HTTP_ADDRESS: ":5000" },
});

waitForPort(5000, 120000)
  .then(() => {
    console.log("[dev] API ready, starting web + scheduler + worker...");
    launch({ name: "web", command: pnpm, args: ["--filter", "web", "dev"] });
    launch({
      name: "scheduler",
      command: go,
      args: ["run", "./apps/scheduler/cmd/scheduler"],
      env: { HEALTH_ADDRESS: ":5001" },
    });
    launch({
      name: "worker",
      command: go,
      args: ["run", "./apps/worker/cmd/worker"],
      env: { HEALTH_ADDRESS: ":5002" },
    });
  })
  .catch((err) => {
    console.error("[dev] API failed to start:", err.message);
    void stop(1);
  });

process.on("SIGINT", () => void stop(0));
process.on("SIGTERM", () => void stop(0));
