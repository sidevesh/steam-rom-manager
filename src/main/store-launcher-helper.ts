import { app } from "electron";
import * as fs from "fs";
import * as path from "path";
import { createHash } from "crypto";
import * as paths from "../paths";
import { logImporterEvent } from "../lib/helpers/importer-diagnostics";

export function deployStoreLauncherHelper(): void {
  if (process.platform !== "win32") {
    return;
  }

  deploy("srm-store-launcher.exe", paths.storeLauncherHelper);
  deploy("srm-store-agent.exe", paths.storeAgent);
}

function deploy(executableName: string, destination: string): void {

  const bundledPath = app.isPackaged
    ? path.join(
        process.resourcesPath,
        "helpers",
        "store-launcher",
        executableName,
      )
    : path.join(
        app.getAppPath(),
        "native",
        "store-launcher",
        "dist",
        process.arch,
        executableName,
      );

  logImporterEvent("helper.deploy.start", { packaged: app.isPackaged, architecture: process.arch, bundledPath, destination, bundledExists: fs.existsSync(bundledPath) });
  const bundled = fs.readFileSync(bundledPath);
  const sha256 = createHash("sha256").update(bundled).digest("hex");
  if (
    fs.existsSync(destination) &&
    bundled.equals(fs.readFileSync(destination))
  ) {
    logImporterEvent("helper.deploy.unchanged", { destination, bytes: bundled.length, sha256 });
    return;
  }

  fs.mkdirSync(path.dirname(destination), { recursive: true });
  const temporaryPath = `${destination}.tmp-${process.pid}`;
  try {
    fs.writeFileSync(temporaryPath, bundled, { mode: 0o755 });
    fs.renameSync(temporaryPath, destination);
    logImporterEvent("helper.deploy.updated", { destination, bytes: bundled.length, sha256 });
  } finally {
    if (fs.existsSync(temporaryPath)) {
      fs.unlinkSync(temporaryPath);
    }
  }
}
