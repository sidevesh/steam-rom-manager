import { app } from "electron";
import * as fs from "fs";
import * as path from "path";
import { createHash } from "crypto";
import * as paths from "../paths";
import { logImporterEvent } from "../lib/helpers/importer-diagnostics";

const executableName = "srm-store-launcher.exe";

export function deployStoreLauncherHelper(): void {
  if (process.platform !== "win32") {
    return;
  }

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

  logImporterEvent("helper.deploy.start", { packaged: app.isPackaged, architecture: process.arch, bundledPath, destination: paths.storeLauncherHelper, bundledExists: fs.existsSync(bundledPath) });
  const bundled = fs.readFileSync(bundledPath);
  const sha256 = createHash("sha256").update(bundled).digest("hex");
  if (
    fs.existsSync(paths.storeLauncherHelper) &&
    bundled.equals(fs.readFileSync(paths.storeLauncherHelper))
  ) {
    logImporterEvent("helper.deploy.unchanged", { destination: paths.storeLauncherHelper, bytes: bundled.length, sha256 });
    return;
  }

  fs.mkdirSync(path.dirname(paths.storeLauncherHelper), { recursive: true });
  const temporaryPath = `${paths.storeLauncherHelper}.tmp-${process.pid}`;
  try {
    fs.writeFileSync(temporaryPath, bundled, { mode: 0o755 });
    fs.renameSync(temporaryPath, paths.storeLauncherHelper);
    logImporterEvent("helper.deploy.updated", { destination: paths.storeLauncherHelper, bytes: bundled.length, sha256 });
  } finally {
    if (fs.existsSync(temporaryPath)) {
      fs.unlinkSync(temporaryPath);
    }
  }
}
