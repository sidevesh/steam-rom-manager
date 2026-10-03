import * as fs from "fs";
import * as os from "os";
import * as path from "path";

const maxLogBytes = 8 * 1024 * 1024;
let reportedWriteFailure = false;

export function importerDiagnosticsPath(): string {
  const root = process.env.LOCALAPPDATA || os.homedir();
  return path.join(root, "Steam ROM Manager", "logs", "importers.log");
}

function redactString(value: string): string {
  let redacted = value;
  for (const home of [process.env.USERPROFILE, os.homedir()]) {
    if (home) {
      const escaped = home.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
      redacted = redacted.replace(new RegExp(escaped, "gi"), "%USERPROFILE%");
    }
  }
  return redacted
    .replace(/(Bearer\s+)[^\s"']+/gi, "$1<redacted>")
    .replace(
      /([?&](?:access_token|refresh_token|token|key|auth|password|secret)=)[^&\s"']+/gi,
      "$1<redacted>",
    )
    .replace(
      /((?:--|\/)(?:token|key|password|secret|auth)(?:=|\s+))(?:"[^"]*"|'[^']*'|[^\s]+)/gi,
      "$1<redacted>",
    )
    .replace(
      /((?:api.?key|token|password|secret|authorization)\s*[:=]\s*)(?:"[^"]*"|'[^']*'|[^\s,}]+)/gi,
      "$1<redacted>",
    );
}

export function redactDiagnosticValue(value: unknown, key = ""): unknown {
  if (
    /(?:api.?key|token|password|secret|credential|authorization)/i.test(key)
  ) {
    return "<redacted>";
  }
  if (typeof value === "string") {
    return redactString(value);
  }
  if (Array.isArray(value)) {
    return value.map((item) => redactDiagnosticValue(item));
  }
  if (value instanceof Error) {
    return { name: value.name, message: redactString(value.message) };
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value).map(([name, item]) => [
        name,
        redactDiagnosticValue(item, name),
      ]),
    );
  }
  return value;
}

export function appendImporterDiagnostic(
  logPath: string,
  event: string,
  details: Record<string, unknown> = {},
  level: "info" | "warn" | "error" = "info",
): void {
  fs.mkdirSync(path.dirname(logPath), { recursive: true, mode: 0o700 });
  const line =
    JSON.stringify({
      time: new Date().toISOString(),
      level,
      event,
      details: redactDiagnosticValue(details),
    }) + "\n";
  if (
    fs.existsSync(logPath) &&
    fs.statSync(logPath).size + Buffer.byteLength(line) > maxLogBytes
  ) {
    const previous = path.join(path.dirname(logPath), "importers.previous.log");
    if (fs.existsSync(previous)) {
      fs.unlinkSync(previous);
    }
    fs.renameSync(logPath, previous);
  }
  fs.appendFileSync(logPath, line, { encoding: "utf8", mode: 0o600 });
}

export function logImporterEvent(
  event: string,
  details: Record<string, unknown> = {},
  level: "info" | "warn" | "error" = "info",
): void {
  // The new launcher integration is Windows-only. Other platforms retain
  // their existing logging behavior and do not get a new background log.
  if (process.platform !== "win32") {
    return;
  }
  try {
    appendImporterDiagnostic(importerDiagnosticsPath(), event, details, level);
  } catch (error) {
    // Diagnostics must never stop game discovery or shortcut generation.
    if (!reportedWriteFailure) {
      reportedWriteFailure = true;
      console.warn("Could not write SRM importer diagnostics", error);
    }
  }
}
