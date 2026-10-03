# Windows importer and launcher diagnostics

The Windows build records store-import and launcher diagnostics automatically. No developer tools or command prompt are needed.

1. Run Steam ROM Manager and regenerate the preview for the affected parser. Save the shortcuts if the problem is with a generated shortcut.
2. If the problem is with launching, start the game from Steam, wait for the failure (or exit the game normally), then close it.
3. Press **Win + R**, paste `%LOCALAPPDATA%\Steam ROM Manager\logs`, and press Enter.
4. Share `importers.log` and `store-launcher.log` from that folder. Include `importers.previous.log` or `store-launcher.previous.log` if the problem happened earlier. Tell the developer the approximate time, game, store, and whether launch via the store was enabled.

`importers.log` contains parser inputs and source locations, discovery counts, per-game inclusion or skip decisions, generated shortcut targets, launcher arguments, and helper deployment results. It is written when SRM runs a parser. It rotates at 8 MiB; the previous file is retained.

`store-launcher.log` contains the Windows helper's startup options, activation result, process-discovery decisions, selected game process, tracking/handoff events, and exit reason. It is written only when a shortcut uses the helper (Epic, EA, Ubisoft, Amazon, GOG, or Xbox launcher mode). It rotates at 1 MiB and also retains a previous file.

The logs do not intentionally record API keys, passwords, or URI query secrets. They **can** contain game titles, store IDs, installation paths, process names, and error messages. Review them before sharing. They are local files and are not automatically uploaded by this diagnostic feature.

For a direct game-executable shortcut, SRM logs how it generated the shortcut, but it does not run while Steam starts the game and therefore cannot record that game's later process lifecycle. For those failures, include the two log files plus what Steam showed when you clicked Play.
