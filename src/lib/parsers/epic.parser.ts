import { ParserInfo, GenericParser, ParsedData } from "../../models";
import { APP } from "../../variables";
import * as _ from "lodash";
import * as fs from "fs-extra";
import * as os from "os";
import * as path from "path";
import { glob } from "glob";
import * as paths from "../../paths";
import { quoteWindowsArgument } from "../helpers/windows-arguments";
import { logImporterEvent } from "../helpers/importer-diagnostics";

export class EpicParser implements GenericParser {
  private get lang() {
    return APP.lang.epicParser;
  }
  getParserInfo(): ParserInfo {
    return {
      title: "Epic",
      info: this.lang.docs__md.self.join(""),
      inputs: {
        epicManifests: {
          label: this.lang.manifestsInputTitle,
          placeholder: this.lang.manifestsInputPlaceholder[os.type()],
          inputType: "dir",
          info: this.lang.docs__md.input.join(""),
        },
        epicLauncherMode: {
          label: this.lang.launcherModeInputTitle,
          inputType: "toggle",
          hidden: os.type() !== "Windows_NT",
          info: this.lang.docs__md.input.join(""),
        },
      },
    };
  }

  execute(
    directories: string[],
    inputs: { [key: string]: any },
    cache?: { [key: string]: any },
  ) {
    return new Promise<ParsedData>(async (resolve, reject) => {
      let appTitles: string[] = [];
      let epicManifestsDir: string = "";
      if (inputs.epicManifests) {
        epicManifestsDir = inputs.epicManifests;
      } else {
        if (os.type() == "Windows_NT") {
          epicManifestsDir =
            "C:\\ProgramData\\Epic\\EpicGamesLauncher\\Data\\Manifests";
        } else if (os.type() == "Linux") {
          return reject(this.lang.errors.epicNotCompatible);
        } else if (os.type() == "Darwin") {
          epicManifestsDir = path.join(
            os.homedir(),
            "/Library/Application Support/Epic/EpicGamesLauncher/Data/Manifests",
          );
        }
      }
      if (!fs.existsSync(epicManifestsDir)) {
        logImporterEvent("epic.manifests.missing", { directory: epicManifestsDir }, "error");
        return reject(this.lang.errors.epicNotInstalled);
      }
      try {
        logImporterEvent("epic.manifests.start", { directory: epicManifestsDir, override: !!inputs.epicManifests, launcherMode: !!inputs.epicLauncherMode });
        let parsedData: ParsedData = {
          executableLocation: paths.storeLauncherHelper,
          success: [],
          failed: [],
        };
        const files: string[] = await glob(
          [epicManifestsDir.replace(/\\/g, "/"), "*.item"].join("/"),
        );
        logImporterEvent("epic.manifests.found", { count: files.length });
        for (let file of files) {
          if (fs.existsSync(file) && fs.lstatSync(file).isFile()) {
            let item = JSON.parse(fs.readFileSync(file).toString());
            let launchPath = path.join(
              item.InstallLocation,
              item.LaunchExecutable,
            );
            if (
              item.LaunchExecutable &&
              fs.existsSync(launchPath) &&
              !appTitles.includes(item.DisplayName)
            ) {
              appTitles.push(item.DisplayName);
              const epicUri = `com.epicgames.launcher://apps/${item.AppName}?action=launch&silent=true`;
              parsedData.success.push({
                extractedTitle: item.DisplayName,
                extractedAppId: item.AppName,
                launchOptions: [
                  "--store",
                  "epic",
                  "--uri",
                  quoteWindowsArgument(epicUri),
                  "--exe",
                  quoteWindowsArgument(launchPath),
                  "--install-dir",
                  quoteWindowsArgument(item.InstallLocation),
                ].join(" "),
                filePath: launchPath,
                fileLaunchOptions: item.LaunchCommand,
              });
            } else {
              logImporterEvent("epic.manifest.skipped", {
                manifest: file,
                title: item.DisplayName,
                hasLaunchExecutable: !!item.LaunchExecutable,
                gameExecutableExists: fs.existsSync(launchPath),
                duplicateTitle: appTitles.includes(item.DisplayName),
              }, "warn");
            }
          }
        }
        resolve(parsedData);
      } catch (err) {
        reject(this.lang.errors.fatalError__i.interpolate({ error: err }));
      }
    });
  }
}
