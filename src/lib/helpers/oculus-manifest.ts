import type { ParsedSuccess } from "../../models";
import * as fs from "fs-extra";
import * as path from "path";

export interface OculusManifest {
  appId?: string | number;
  canonicalName?: string;
  displayName?: string;
  isCore?: boolean;
  launchFile?: string;
  launchParameters?: string;
  name?: string;
  packageType?: string;
  thirdParty?: boolean;
  title?: string;
}

export function titleFromOculusCanonicalName(canonicalName: string) {
  return canonicalName
    .split(/[-_]+/)
    .filter(Boolean)
    .map((word) => word[0].toUpperCase() + word.slice(1))
    .join(" ");
}

export function parseOculusManifest(
  libraryPath: string,
  manifest: OculusManifest,
  pathExists: (path: string) => boolean = fs.existsSync,
): ParsedSuccess | null {
  if (
    !manifest.appId ||
    !manifest.canonicalName ||
    !manifest.launchFile ||
    manifest.isCore ||
    manifest.thirdParty ||
    (manifest.packageType && manifest.packageType !== "APP")
  ) {
    return null;
  }

  const installDirectory = path.win32.join(
    libraryPath,
    "Software",
    manifest.canonicalName,
  );
  const executablePath = path.win32.join(
    installDirectory,
    manifest.launchFile.replace(/\//g, "\\"),
  );
  if (!pathExists(executablePath)) {
    return null;
  }

  return {
    extractedTitle:
      manifest.displayName ||
      manifest.name ||
      manifest.title ||
      titleFromOculusCanonicalName(manifest.canonicalName),
    extractedAppId: String(manifest.appId),
    filePath: executablePath,
    fileLaunchOptions:
      manifest.launchParameters && manifest.launchParameters !== "None"
        ? manifest.launchParameters
        : "",
    startInDirectory: installDirectory,
    openVR: true,
  };
}
