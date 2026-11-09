It's a beta. It has worked well on my own lists, but you will find things that don't. Please [open an issue](https://github.com/dstrus/mcm-lexicon/issues) if and when something looks wrong, especially a menu, page, or option that reads differently in the game.

## Download

- **Windows:** `MCMLexicon-windows-x64.zip`. Put `MCMLexicon.exe` in your MO2 instance folder (beside `ModOrganizer.ini`), or in your Skyrim folder (beside `SkyrimVR.exe` or `SkyrimSE.exe`) if you use Vortex or nothing, and run it. It opens in a window of its own; closing the window stops it.

Releases are for Windows only. On macOS or Linux, you can build it yourself; see [DEVELOPING.md](https://github.com/dstrus/mcm-lexicon/blob/main/DEVELOPING.md). `SHA256SUMS` lists the download's checksum.

## Before you start

- MCM Lexicon won't write anything while Skyrim is running. You can still browse.
- Nothing is written until you press **Save**, and the **Changes** page shows every file as it will be written first. Before a file is replaced or removed, a copy goes to `lexicon-backups`, beside your mod list.
- Recordings save to a separate mod folder from any that are already in your list, named `[NoDelete] MCM Lexicon Recordings` unless you name it otherwise in **Settings**. MO2 lists it as a new mod, disabled, the first time. Enable it so the game reads what's in it. A recording that came with another mod is never changed in place. If you change one, it saves a copy to your folder.
- Keyboard shortcuts are shown in a shortcut bar in the lower-right. Those are also clickable buttons.
- The build isn't code-signed yet, so Windows will warn you the first time: if SmartScreen says "Windows protected your PC," choose **More info**, then **Run anyway**.
