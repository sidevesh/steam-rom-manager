import { test } from "node:test";
import * as assert from "node:assert/strict";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";
import {
  appendImporterDiagnostic,
  redactDiagnosticValue,
} from "./importer-diagnostics";

test("diagnostics redact credentials and the user home path", () => {
  const redacted = redactDiagnosticValue({
    steamApiKey: "private-api-key",
    target: path.join(os.homedir(), "Games", "Game.exe"),
    arguments:
      '--token "private-token" origin2://game/launch/?offerIds=42&key=private-query',
  }) as Record<string, string>;

  assert.equal(redacted.steamApiKey, "<redacted>");
  assert.match(redacted.target, /%USERPROFILE%/);
  assert.doesNotMatch(JSON.stringify(redacted), /private-/);
  assert.match(redacted.arguments, /offerIds=42/);
});

test("diagnostics redact Windows profile paths with regex characters", () => {
  const previous = process.env.USERPROFILE;
  process.env.USERPROFILE = "C:\\Users\\Player.Name";
  try {
    assert.equal(
      redactDiagnosticValue("C:\\Users\\Player.Name\\Games\\Game.exe"),
      "%USERPROFILE%\\Games\\Game.exe",
    );
  } finally {
    if (previous === undefined) {
      delete process.env.USERPROFILE;
    } else {
      process.env.USERPROFILE = previous;
    }
  }
});

test("diagnostics write JSON lines and rotate without interrupting the next event", () => {
  const directory = fs.mkdtempSync(
    path.join(os.tmpdir(), "srm-diagnostics-test-"),
  );
  try {
    const logPath = path.join(directory, "importers.log");
    appendImporterDiagnostic(logPath, "parser.start", {
      parser: "Epic",
      steamApiKey: "private",
    });
    const first = JSON.parse(fs.readFileSync(logPath, "utf8"));
    assert.equal(first.event, "parser.start");
    assert.equal(first.details.steamApiKey, "<redacted>");

    fs.writeFileSync(logPath, Buffer.alloc(8 * 1024 * 1024, 0x78));
    appendImporterDiagnostic(logPath, "parser.complete", { count: 1 });
    assert.equal(
      JSON.parse(fs.readFileSync(logPath, "utf8")).event,
      "parser.complete",
    );
    assert.equal(
      fs.statSync(path.join(directory, "importers.previous.log")).size,
      8 * 1024 * 1024,
    );
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
});
