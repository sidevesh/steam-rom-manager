import { app } from "electron";
import * as fs from "fs";
import * as path from "path";
import * as paths from "../paths";

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

  const bundled = fs.readFileSync(bundledPath);
  if (
    fs.existsSync(paths.storeLauncherHelper) &&
    bundled.equals(fs.readFileSync(paths.storeLauncherHelper))
  ) {
    return;
  }

  fs.mkdirSync(path.dirname(paths.storeLauncherHelper), { recursive: true });
  const temporaryPath = `${paths.storeLauncherHelper}.tmp-${process.pid}`;
  try {
    fs.writeFileSync(temporaryPath, bundled, { mode: 0o755 });
    fs.renameSync(temporaryPath, paths.storeLauncherHelper);
  } finally {
    if (fs.existsSync(temporaryPath)) {
      fs.unlinkSync(temporaryPath);
    }
  }
}
