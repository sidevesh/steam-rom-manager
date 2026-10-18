import { app } from "electron";
import * as fs from "fs";
import * as path from "path";
import * as paths from "../paths";

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

  const bundled = fs.readFileSync(bundledPath);
  if (
    fs.existsSync(destination) &&
    bundled.equals(fs.readFileSync(destination))
  ) {
    return;
  }

  fs.mkdirSync(path.dirname(destination), { recursive: true });
  const temporaryPath = `${destination}.tmp-${process.pid}`;
  try {
    fs.writeFileSync(temporaryPath, bundled, { mode: 0o755 });
    fs.renameSync(temporaryPath, destination);
  } finally {
    if (fs.existsSync(temporaryPath)) {
      fs.unlinkSync(temporaryPath);
    }
  }
}
