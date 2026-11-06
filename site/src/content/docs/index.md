---
title: Getting started
description: What MCM Lexicon needs, where to put it, and what happens the first time you run it.
order: 1
---

MCM Lexicon is one program. You put it next to your mod list, run it, and it opens in your browser. That's most of the setup, honestly.

## What you need

- Windows. (Releases are for Windows only, but you can [build it yourself](https://github.com/dstrus/mcm-lexicon/blob/main/DEVELOPING.md) for macOS or Linux.)
- A Skyrim mod list managed with Mod Organizer 2, or a plain `Data` folder (which includes Vortex setups). Skyrim VR, SE, and AE all work.
- [MCM Recorder](https://www.nexusmods.com/skyrimspecialedition/mods/61719) in that list, if you want to play your recordings in the game. MCM Lexicon writes them; MCM Recorder plays them.

## Installing it

[Download the latest release](https://github.com/dstrus/mcm-lexicon/releases/latest/download/MCMLexicon-windows-x64.zip) and unzip it. Then put `MCMLexicon.exe` where it can find your list:

- **MO2 users:** your MO2 instance folder, the one holding `ModOrganizer.ini`, `mods`, and `profiles`.
- **Everyone else (including Vortex users):** your Skyrim installation folder, the one holding `SkyrimVR.exe` or `SkyrimSE.exe`, or its `Data` folder.

## Running it

Run `MCMLexicon.exe`. It finds your list, scans it (a few seconds for 1,800 mods), and opens in your browser.

If you ran it from somewhere else and it can't find a list, it asks for the path to your list in the browser instead. You can change the list later in Settings, too.

With Mod Organizer 2, MCM Lexicon reads the profile that's currently selected in MO2, and there's a dropdown at the top of the page to switch. It follows the profile's mod order, `overwrite` included, and it watches `modlist.txt` and `plugins.txt`, so a change in MO2 shows up without restarting MCM Lexicon.

<div class="btw">

By the way…

A small window stays open while MCM Lexicon runs. Close it (or press Ctrl+C in it) when you're done. Closing the browser tab doesn't stop it.

</div>

## Looking around

Everything starts out read-only, so you can't break anything by poking around. The column on the left lists every menu in your list by the name SkyUI shows it under, and picking one draws it the way SkyUI draws it, with each option's help text, its exact label, and what it allows.

Press `/` anywhere to find a menu by name, or use Search to find an option in any menu.

When you're ready to change something, see [Making a recording](making-a-recording/).
