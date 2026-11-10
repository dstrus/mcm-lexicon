# MCM Lexicon

*Catalogs every option in every Mod Configuration Menu in your list and lets you create and edit MCM Recorder recordings without launching Skyrim.*

Yes, it's vibe-coded, but at least it's vibe-coded by a professional software engineer with lots of experience. 🤷🏻‍♂️

## Why?

[MCM Recorder](https://www.nexusmods.com/skyrimspecialedition/mods/61719) records your clicks in Mod Configuration Menus and replays them on a new game, so you don't have to change dozens of settings by hand every time. It's great!

So why wouldn't you just use MCM Recorder to make these recordings? If you're asking, then you probably _should_ just use MCM Recorder. There's nothing wrong with it!

But the recording has to be made in the game.

Personally, I play [Skyrim VR with 1700+ mods](https://synergyvr.org/mgo). A great many things involving menus are far less convenient in VR, and launching the game with so many mods takes a while. So if that's what it takes to add one more setting to one of my recordings, I'm probably just not going to bother. The next time I start a new playthrough (which I do far too often frankly), I'll probably wish I had.

You can edit recordings by hand (they're just JSON files), but every label and value must be exactly as each mod recognizes them, and they're pretty much never documented.

So that's why this exists: so I can easily create and edit these recordings, typo-free, without putting on my headset and waiting multiple minutes for the game to launch.

## Who?

I'm Davey. I like Skyrim VR so much that I sometimes make tools to avoid actually launching it. I've been a software engineer for somewhere in the neighborhood of a quarter-century.

## What?

MCM Lexicon reads your mod list from disk, with Skyrim closed, and builds a catalog of every MCM in it. That means each menu's name as it appears in SkyUI, its pages, its options, which values it allows, and its defaults. Then it lets you do things with that catalog.

_Please excuse the bullet points that begin with a bold verb or verb phrase, but it's an AI cliché for a reason. (That reason is because it's something real people actually do sometimes.)_

- **Browse** any menu, laid out as close as possible to the way it appears in SkyUI, with each option's help text, its label, and (whenever possible) its defaults.
- **Search** every option in every menu by label, page, menu, or mod.
- **Create and edit MCM Recorder recordings** with a graphical interface, like in the in-game menus. You don't have to remember what the JSON property names are for a million different things. Changes go into a reviewable draft, and nothing is actually written to a proper recording until you say so.
- **Check existing recordings** against what's installed, so you know before you launch which steps still have a matching menu/option in your game.
- **Compare** two versions of a list's catalog, to see which MCMs changed between releases and which recorded steps are broken by those changes.
- **Capture** what every menu shows in your running game with an optional SKSE plugin, for the values and labels that are not otherwise discoverable. (Yes, this one involves launching Skyrim. Sorry.)

It's a single program that runs a small web server on your machine and shows its interface in a window of its own (Microsoft Edge's WebView2, part of Windows 10 and 11). It reads MCMs defined in your load order and writes new recordings and changes to a separate mod.

It works with [MCM Recorder](https://www.nexusmods.com/skyrimspecialedition/mods/61719) but isn't part of it, and isn't affiliated with MCM Recorder, SkyUI, MCM Helper, or Mod Organizer. It does not include any files from those other sources.

## Installing it

**MO2 users:** Put `MCMLexicon.exe` in your MO2 instance folder (the one holding `ModOrganizer.ini`, `mods`, and `profiles`).

**Everyone else (including Vortex users):** Put `MCMLexicon.exe` in your Skyrim installation folder (the one holding `SkyrimVR.exe` or `SkyrimSE.exe`), or in its `Data` folder.

Releases are for Windows only, but you can build it for macOS or Linux. See [DEVELOPING.md](DEVELOPING.md).

**The capture plugin (optional):** Install `MCMLexicon-Capture.zip` as a mod, like any other. It's an SKSE plugin for Skyrim SE, AE, and VR, and needs SKSE and Address Library (or VR Address Library, as the case may be). It has no Skyrim plugin of its own, so it doesn't touch your load order, and it never writes to your saves, so you can remove it whenever you like.

## Running it

Just run `MCMLexicon.exe`. If you run it from your MO2 folder or game folder, it finds your mod list automatically, scans it (a few seconds for 1,800 mods), and opens in its own window. Closing the window stops it.

If it can't find a list because you ran it from somewhere else, it asks for the path to your list.

On the off chance your PC doesn't have WebView2, MCM Lexicon opens in your browser instead, with a small message box that stops it when you click OK.

If you want to run it from a terminal, there are fancy command-line switches.

```
MCMLexicon.exe [-list <MO2 instance or Data folder>] [-profile <MO2 profile>] [-port 8766] [-no-browser] [-version]
```

With Mod Organizer 2, MCM Lexicon reads the profile that's currently selected in MO2, or the one you name (and there's a dropdown in the GUI to switch), and follows its mod order, `overwrite` included. It watches the profile's `modlist.txt` and `plugins.txt`, so a change in MO2 shows up without restarting MCM Lexicon.

There are also a few commands for working with catalogs without the GUI. (Settings can save a catalog file or make a snapshot too.) (These exist because I build these utilities from a Mac where Skyrim isn't installed at all.)

Print all of the relevant mods from your list, with the files MCM Lexicon reads from each:
```
MCMLexicon.exe scan [-list …] [-all]
```

Generate the catalog file, which the app can open on any machine, including one that doesn't have your mod list (or even Skyrim) installed:
```
MCMLexicon.exe scan -json -o catalog.json [-list …]
```

Show the differences between two catalogs (_e.g._ from one version of a modlist to the next):
```
MCMLexicon.exe diff old.json new.json
```

Copy the portion of your load order that MCM Lexicon would actually read, for testing on a machine that doesn't have the full list:
```
MCMLexicon.exe snapshot [-list …] [-o out.zip]
```

## Where MCM Lexicon puts things

If you're using an Mod Organizer 2 list, **recordings** save to a mod folder of their own in your MO2 instance, called `[NoDelete] MCM Lexicon Recordings` unless you change it in Settings. (The `[NoDelete] ` prefix prevents a Wabbajack update from deleting it.) MO2 will list this folder as a new mod, disabled, the first time. Enable it so MCM Recorder will actually play it back in-game.

MCM Lexicon will never change a recording that came with another mod directly. Making a change saves a _copy_ to the same separate mod folder as a brand new recording. The original mod remains untouched.

If you're working with a plain `Data` folder instead of an MO2-managed list, recordings get saved there.

If you use the capture plugin, **Capture in game** writes a tiny file called `capture-request.json` to `Documents\My Games\<your game>\SKSE\MCM Lexicon`. The plugin deletes it at your next load and writes `capture.json` there in its place, plus a log (`MCMLexicon.log`) in the parent folder, alongside SKSE's logs.

When MCM Lexicon replaces or removes a file, **backups** go to a `lexicon-backups` folder beside the list (next to `ModOrganizer.ini` or `Data`). The Backups tab for a recording lists its backups and can restore them.

**Settings** (which list was open, the theme, the save folder's name, your capture settings) live in `MCM Lexicon/settings.json` under your user configuration folder. A scan cache is saved under your user cache folder.

Nothing else is written anywhere.

## Safety

- MCM Lexicon will not write anything while Skyrim is running. You can still browse.
- Nothing is written until you click the big golden **Save** (or **Save All**) button. Before you save, the **Changes** page shows every change as it will be written, compared to what's on disk now.
- Before a file is replaced or removed, a copy goes to `lexicon-backups`.
- If a save cannot finish completely, it leaves the old recording untouched. You can never end up with half a recording. (For the nerds: _writes are atomic_.)
- The server only listens on `127.0.0.1`, and every API request must carry a per-session token from the page. Nothing else on your PC, and no website open in a browser, can reach your files through it.
- The capture plugin only does anything when you ask it to, and each request is used up the moment a capture starts, so a menu that misbehaves can't do it twice. It never writes to your saves.
- It reads files from mods installed on your machine when creating the catalog, but there are no files from other mods included here. Snapshots generated from a list, which copy other mods' files for testing, are for your own machine.

## Development

How to build MCM Lexicon, run its tests, and find your way around the code is in [DEVELOPING.md](DEVELOPING.md).

## License

MCM Lexicon is released under the [MIT License](LICENSE). Copyright (c) 2026 Davey Strus. It includes other people's software and fonts, each under its own license. [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt) lists them with their license texts, and that's included in releases.

The capture plugin, `MCMLexicon.dll`, is built on [CommonLibSSE NG](https://github.com/alandtse/CommonLibSSE-NG), which is GPL-licensed, so the plugin as distributed is under GPL-3.0-or-later. Its source (the `skse` folder in this repo) is still MIT. [skse/LICENSE.md](skse/LICENSE.md) explains, and the plugin's zip carries its license and notices.

The Elder Scrolls V: Skyrim is a trademark of ZeniMax Media Inc.
