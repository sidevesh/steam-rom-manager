import { ParserInfo, GenericParser, ParsedData } from "../../models";
import { APP } from "../../variables";
import * as fs from "fs-extra";
import * as os from "os";
import * as path from "path";
import { parse } from "yaml";
import { SqliteWrapper } from "../helpers/sqlite";
import * as paths from "../../paths";
import { quoteWindowsArgument } from "../helpers/windows-arguments";
import { logImporterEvent } from "../helpers/importer-diagnostics";

export class AmazonGamesParser implements GenericParser {
  private get lang() {
    return APP.lang.amazonGamesParser;
  }
  getParserInfo(): ParserInfo {
    return {
      title: "Amazon Games",
      info: this.lang.docs__md.self.join(""),
      inputs: {
        amazonGamesExeOverride: {
          label: this.lang.exeOverrideTitle,
          placeholder: this.lang.exeOverridePlaceholder[os.type()],
          inputType: "path",
          info: this.lang.docs__md.input.join(""),
        },
        amazonGamesLauncherMode: {
          label: this.lang.launcherModeInputTitle,
          inputType: "toggle",
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
    return new Promise<ParsedData>((resolve, reject) => {
      try {
        if (os.type() != "Windows_NT") {
          return reject(this.lang.errors.osUnsupported);
        }

        const launcherMode = inputs.amazonGamesLauncherMode;

        const amazonGamesExe =
          inputs.amazonGamesExeOverride ||
          path.resolve(
            `${process.env.APPDATA}\\..\\local\\Amazon Games\\App\\Amazon Games.exe`,
          );
        const dbPath = path.resolve(
          `${path.dirname(amazonGamesExe)}\\..\\Data\\Games\\Sql\\GameInstallInfo.sqlite`,
        );
        logImporterEvent("amazon.source", { launcherMode, launcherExecutable: amazonGamesExe, launcherOverride: !!inputs.amazonGamesExeOverride, database: dbPath, databaseExists: fs.existsSync(dbPath) });

        if (!fs.existsSync(dbPath)) {
          return reject(this.lang.errors.databaseNotFound);
        }
        const sqliteWrapper = new SqliteWrapper("amazon-games", dbPath);
        sqliteWrapper
          .callWorker()
          .then((games: { [k: string]: any }[]) => {
            logImporterEvent("amazon.database.rows", { count: games.length });
            for (const game of games) {
              const fuelPath = game.InstallDirectory
                ? path.join(game.InstallDirectory, "fuel.json")
                : "";
              const fuelExists = !!fuelPath && fs.existsSync(fuelPath);
              if (!game.Installed || (!launcherMode && !fuelExists)) {
                logImporterEvent("amazon.game.skipped", { title: game.ProductTitle, productId: game.Id, installDirectory: game.InstallDirectory, installed: !!game.Installed, fuelExists }, "warn");
              }
            }
            const success = games
              .filter(
                ({
                  InstallDirectory,
                  Installed,
                }: {
                  [key: string]: string;
                }) => {
                  return (
                    (fs.existsSync(`${InstallDirectory}\\fuel.json`) ||
                      launcherMode) &&
                    Installed
                  );
                },
              )
              .map(
                ({
                  ProductTitle,
                  InstallDirectory,
                  Installed,
                  Id,
                }: {
                  [key: string]: string;
                }) => {
                  if (launcherMode) {
                    let filePath: string;
                    const fuelPath = path.join(InstallDirectory, "fuel.json");
                    if (fs.existsSync(fuelPath)) {
                      try {
                        const fuel = parse(fs.readFileSync(fuelPath, "utf8"));
                        if (fuel?.Main?.Command) {
                          filePath = path.join(
                            InstallDirectory,
                            fuel.Main.Command,
                          );
                        }
                      } catch (error) {
                        logImporterEvent("amazon.fuel.failed", { title: ProductTitle, fuelPath, error: String(error) }, "warn");
                        // The install directory can still identify the running game.
                      }
                    }
                    logImporterEvent("amazon.game.launcher", { title: ProductTitle, productId: Id, installDirectory: InstallDirectory, fuelExists: fs.existsSync(fuelPath), expectedExecutable: filePath, expectedExecutableExists: !!filePath && fs.existsSync(filePath) });
                    return {
                      extractedTitle: ProductTitle,
                      startInDirectory: InstallDirectory,
                      filePath,
                      launchOptions: [
                        "--store amazon",
                        "--launch-exe",
                        quoteWindowsArgument(amazonGamesExe),
                        "--launch-arg",
                        quoteWindowsArgument(`amazon-games://play/${Id}`),
                        "--launch-cwd",
                        quoteWindowsArgument(path.dirname(amazonGamesExe)),
                        ...(filePath
                          ? ["--exe", quoteWindowsArgument(filePath)]
                          : []),
                        "--install-dir",
                        quoteWindowsArgument(InstallDirectory),
                      ].join(" "),
                    };
                  }

                  const fuelJson = fs.readFileSync(
                    `${InstallDirectory}\\fuel.json`,
                  );
                  // not really json so need to parse with yaml parser
                  const {
                    Main: { Command, Args },
                  } = parse(fuelJson.toString());
                  return {
                    extractedTitle: ProductTitle,
                    startInDirectory: InstallDirectory,
                    filePath: `${InstallDirectory}\\${Command}`,
                    fileLaunchOptions: Args?.join(" "),
                  };
                },
              );

            resolve({
              executableLocation: launcherMode
                ? paths.storeLauncherHelper
                : null,
              success: success,
              failed: [],
            });
          })
          .catch((error) => {
            reject(
              this.lang.errors.fatalError__i.interpolate({ error: error }),
            );
          });
      } catch (error) {
        reject(this.lang.errors.fatalError__i.interpolate({ error: error }));
      }
    });
  }
}
