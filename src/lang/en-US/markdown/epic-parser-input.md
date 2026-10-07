# Epic Games Store Specific Inputs

## Manifests Directory Override

Most users shouldn't use this, as they use the standard `Epic Games Store` installation where the game manifests will be located in, for example on Windows, `C:\ProgramData\Epic\EpicGamesLauncher\Data\Manifests`.

If, however, for some reason your manifests directory is located in a non-typical location then you can specify the correct manifests directory here.

## Launch Via Epic Games Store `[Recommend enabled]`

This toggle lets you set whether games will launch via the Epic Games Store or directly. If you don't launch via the `Epic Games Store` you will not have access to Epic's online services in game.

## Epic cloud sync and exit waits (Windows)

The helper waits up to 15 seconds for an existing Epic cloud sync before restarting Epic under Steam, and up to 15 seconds for this game's post-exit cloud sync before closing Epic. If the log is missing or does not report completion, it closes Epic when the wait expires. It requests a normal exit first, then forces the process to end after the 5 second exit grace. A timeout can interrupt a cloud sync or other Epic background work. These values can be changed in the parser inputs or in a shortcut's launch options with `--epic-pre-sync-wait`, `--epic-post-sync-wait`, and `--epic-exit-grace`. Durations use `ms`, `s`, `m`, or `h`; `0s` disables a wait. The Epic log path can be overridden with `--epic-log-file`.
