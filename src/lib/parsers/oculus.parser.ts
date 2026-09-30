import { GenericParser, ParsedData, ParserInfo } from "../../models";
import { APP } from "../../variables";
import * as fs from "fs-extra";
import * as os from "os";
import * as path from "path";
import { execFileSync } from "child_process";
import Registry from "winreg";
import {
  OculusManifest,
  parseOculusManifest,
} from "../helpers/oculus-manifest";

const OCULUS_LIBRARIES_KEY = "\\Software\\Oculus VR, LLC\\Oculus\\Libraries";

export class OculusParser implements GenericParser {
  private get lang() {
    return APP.lang.oculusParser;
  }

  getParserInfo(): ParserInfo {
    return {
      title: "Oculus",
      info: this.lang.docs__md.self.join(""),
      inputs: {
        oculusLibraryDir: {
          label: this.lang.libraryDirTitle,
          placeholder: this.lang.libraryDirPlaceholder[os.type()],
          inputType: "dir",
          info: this.lang.docs__md.input.join(""),
        },
      },
    };
  }

  execute(
    directories: string[],
    inputs: { [key: string]: any },
    cache?: { [key: string]: any },
  ): Promise<ParsedData> {
    return new Promise<ParsedData>(async (resolve, reject) => {
      if (os.type() !== "Windows_NT") {
        return reject(this.lang.errors.oculusNotCompatible);
      }

      try {
        const configuredLibrary = this.cleanLibraryPath(
          inputs.oculusLibraryDir,
        );
        const libraryPaths = configuredLibrary
          ? this.driveLetterLibraryPaths([configuredLibrary])
          : await this.getOculusLibraryPaths();

        const parsedData: ParsedData = { success: [], failed: [] };
        const seenApps = new Set<string>();

        for (const libraryPath of libraryPaths) {
          this.addLibraryGames(libraryPath, parsedData, seenApps);
        }

        if (!libraryPaths.length) {
          return reject(this.lang.errors.oculusNotInstalled);
        }

        resolve(parsedData);
      } catch (err) {
        reject(this.lang.errors.fatalError__i.interpolate({ error: err }));
      }
    });
  }

  private addLibraryGames(
    libraryPath: string,
    parsedData: ParsedData,
    seenApps: Set<string>,
  ) {
    const manifestsDir = path.win32.join(libraryPath, "Manifests");
    if (!fs.existsSync(manifestsDir)) {
      parsedData.failed.push(
        `Oculus library has no Manifests directory: ${libraryPath}`,
      );
      return;
    }

    const manifestFiles = fs
      .readdirSync(manifestsDir)
      .filter(
        (filename) =>
          (filename.endsWith(".json") || filename.endsWith(".json.mini")) &&
          !filename.endsWith("_assets.json"),
      )
      // Prefer the full manifest. The mini manifest remains a fallback for
      // installations where Oculus only kept the compact copy.
      .sort(
        (left, right) =>
          Number(left.endsWith(".mini")) - Number(right.endsWith(".mini")),
      );
    const seenManifestFiles = new Set<string>();

    for (const filename of manifestFiles) {
      const manifestKey = filename.replace(/\.mini$/, "").toLowerCase();
      if (seenManifestFiles.has(manifestKey)) {
        continue;
      }
      const manifestPath = path.win32.join(manifestsDir, filename);
      try {
        const manifest = JSON.parse(
          fs.readFileSync(manifestPath, "utf8"),
        ) as OculusManifest;
        seenManifestFiles.add(manifestKey);
        const parsedGame = parseOculusManifest(libraryPath, manifest);
        if (!parsedGame) {
          continue;
        }

        const appKey = String(manifest.appId);
        if (seenApps.has(appKey)) {
          continue;
        }
        seenApps.add(appKey);
        parsedData.success.push(parsedGame);
      } catch (err) {
        parsedData.failed.push(
          `Could not parse Oculus manifest ${manifestPath}: ${err}`,
        );
      }
    }
  }

  private cleanLibraryPath(value: unknown) {
    if (typeof value !== "string" || !value.trim()) {
      return "";
    }
    return path.win32.normalize(value.trim().replace(/^"|"$/g, ""));
  }

  private driveLetterLibraryPaths(candidates: string[]) {
    const cleaned = candidates.map((candidate) => this.cleanLibraryPath(candidate));
    const mountedVolumes = new Map<string, string>();
    if (cleaned.some((candidate) => /^\\\\\?\\Volume\{/i.test(candidate))) {
      try {
        const output = execFileSync("mountvol", [], {
          encoding: "utf8",
          timeout: 5000,
          windowsHide: true,
        });
        for (const block of output.split(/(?=\\\\\?\\Volume\{)/i)) {
          const [volume, ...mounts] = block.trim().split(/\r?\n/).map((line) => line.trim());
          const guid = /^\\\\\?\\Volume\{[0-9a-f-]+\}/i.exec(volume)?.[0];
          const drivePath = mounts
            .filter((mount) => /^[a-z]:\\/i.test(mount))
            .sort((left, right) => left.length - right.length)[0];
          if (guid && drivePath) {
            mountedVolumes.set(guid.toLowerCase(), drivePath);
          }
        }
      } catch {
        // A missing drive mount must never become a GUID-based Steam shortcut.
      }
    }

    const resolved = cleaned
      .map((candidate) => {
        const volume = /^(\\\\\?\\Volume\{[0-9a-f-]+\})(.*)$/i.exec(candidate);
        if (!volume) {
          return candidate;
        }
        const drivePath = mountedVolumes.get(volume[1].toLowerCase());
        return drivePath ? path.win32.join(drivePath, volume[2]) : "";
      })
      .filter(Boolean);

    return Array.from(
      new Map(resolved.map((candidate) => [candidate.toLowerCase(), candidate])).values(),
    );
  }

  private async getOculusLibraryPaths() {
    const candidates = (
      await Promise.all([
        this.getRegistryLibraryPaths("x64"),
        this.getRegistryLibraryPaths("x86"),
      ])
    ).flat();

    return this.driveLetterLibraryPaths(candidates).filter((candidate) =>
      fs.existsSync(path.win32.join(candidate, "Manifests")),
    );
  }

  private getRegistryLibraryPaths(arch: "x64" | "x86") {
    return new Promise<string[]>((resolve) => {
      const librariesKey = new Registry({
        hive: Registry.HKCU,
        key: OCULUS_LIBRARIES_KEY,
        arch,
      });

      librariesKey.keys((keysError, keys) => {
        if (keysError || !keys) {
          return resolve([]);
        }

        Promise.all(
          keys.map(
            (key) =>
              new Promise<string[]>((resolveKey) => {
                key.values((valuesError, values) => {
                  if (valuesError || !values) {
                    return resolveKey([]);
                  }
                  const valueMap = Object.fromEntries(
                    values.map((value) => [
                      value.name.toLowerCase(),
                      value.value,
                    ]),
                  );
                  resolveKey(
                    [valueMap.path, valueMap.originalpath].filter(Boolean),
                  );
                });
              }),
          ),
        ).then((paths) => resolve(paths.flat()));
      });
    });
  }
}
