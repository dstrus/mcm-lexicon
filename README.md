# MCM Lexicon

*Catalogs every option in every Mod Configuration Menu in your list and lets you create and edit MCM Recorder recordings without launching Skyrim.*

Yes, it's vibe-coded, but at least it's vibe-coded by a professional software engineer with lots of experience. 🤷🏻‍♂️

## Why?

[MCM Recorder](https://www.nexusmods.com/skyrimspecialedition/mods/61719) records your clicks in Mod Configuration Menus and replays them on a new game, so you don't have to change dozens of settings by hand every time. It's great!

So why wouldn't you just use MCM Recorder to make these recordings? If you're asking, then you probably _should_ just use MCM Recorder. There's nothing wrong with it!

But the recording has to be made in the game, by clicking.

Personally, I play [Skyrim VR with 1700+ mods](https://synergyvr.org/mgo). A great many things involving menus are far less convenient in VR, and launching the game with so many mods takes a while. So if that's what it takes to add one more setting to one of my recordings, I'm probably just not going to bother. The next time I start a new playthrough (which I do far too often frankly), I'll probably wish I had.

You can edit recordings by hand (they're just JSON files), but every label and value must be exactly as each mod recognizes them, and they're pretty much never documented.

So that's why this exists: so I can easily create and edit these recordings, typo-free, without putting on my headset and waiting multiple minutes for the game to launch.

## Who?

I'm Davey. I like Skyrim VR so much that I sometimes make tools to avoid actually launching it. I've been a software engineer for somewhere in the neighborhood of a quarter-century.

## What?

MCM Lexicon reads your mod list from disk, with Skyrim closed, and builds a catalog of every MCM in it. That means each menu's name as it appears in SkyUI, its pages, its options, which values it allows, and its defaults. Then it lets you do things with that catalog.

_Please excuse the bullet points that begin with a bold verb or verb phrase, but it's an AI cliché for a reason. (That reason is because it's something real people actually do sometimes.)_

- **Browse** any menu, laid out as close as possible to the way it appears in SkyUI, with each option's help text, its exact label, and where it's defined.
- **Search** every option in every menu by label, page, menu, or mod.
- **Create and edit MCM Recorder recordings** with a graphical interface, like in the in-game menus. You don't have to remember what the JSON property names are for a million different things. Changes go into a reviewable draft, and nothing is actually written to a proper recording until you say so.
- **Check existing recordings** against what's installed, so you know before you launch which steps still have a matching menu/option in your game.
- **Compare** two versions of a list's catalog, to see which MCMs changed between releases and which recorded steps are broken by those changes.

It's a single program that runs a small web server on your machine and opens its interface in your browser. It reads MCMs defined in your load order and writes new recordings and changes to a separate mod.

It works with [MCM Recorder](https://www.nexusmods.com/skyrimspecialedition/mods/61719) but isn't part of it, and isn't affiliated with MCM Recorder, SkyUI, MCM Helper, or Mod Organizer. It does not include any files from those other sources.

## Installing it (Windows)

**MO2 users:** Put `MCMLexicon.exe` in your MO2 instance folder (the one holding `ModOrganizer.ini`, `mods`, and `profiles`).

**Everyone else (including Vortex users):** Put `MCMLexicon.exe` in your Skyrim installation folder (the one holding `SkyrimVR.exe` or `SkyrimSE.exe`), or in its `Data` folder.

## Running it

Just run `MCMLexicon.exe`. If you run it from your MO2 folder or game folder, it finds your mod list automatically, scans it (a few seconds for 1,800 mods), and opens in your browser.

If it can't find a list because you ran it from somewhere else, it asks for the path to your list in the browser.

If you want to run it from a terminal, there are fancy command-line switches.

```
MCMLexicon.exe [-list <MO2 instance or Data folder>] [-profile <MO2 profile>] [-port 8766] [-no-browser] [-version]
```

With Mod Organizer 2, MCM Lexicon reads the profile that's currently selected in MO2, or the one you name (and there's a dropdown in the GUI to switch), and follows its mod order, `overwrite` included. It watches the profile's `modlist.txt` and `plugins.txt`, so a change in MO2 shows up without restarting MCM Lexicon.

There are also a few commands for working with catalogs without the browser-based GUI. (These exist because I build these utilities from a Mac where Skyrim isn't installed at all.)

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

When MCM Lexicon replaces or removes a file, **backups** go to a `lexicon-backups` folder beside the list (next to `ModOrganizer.ini` or `Data`). The Backups tab for a recording lists its backups and can restore them.

**Settings** (which list was open, the theme, the save folder's name) live in `MCM Lexicon/settings.json` under your user configuration folder. A scan cache is saved under your user cache folder.

Nothing else is written anywhere.

## Safety

- MCM Lexicon will not write anything while Skyrim is running. You can still browse.
- Nothing is written until you click the big golden **Save** (or **Save All**) button. Before you save, the **Changes** page shows every change as it will be written, compared to what's on disk now.
- Before a file is replaced or removed, a copy goes to `lexicon-backups`.
- A save cannot finishes completely, it leaves the old recording untouched. You can never end up with half a recording. (For the nerds: _writes are atomic_.)
- The server only listens on `127.0.0.1`, and every API request must carry a per-session token from the page. Other sites open in your browser can't reach your files.
- It reads files from mods installed on your machine when creating the catalog, but there are no files from other mods included here. Snapshots generated from a list, which copy other mods' files for testing, are for your own machine.

## Building

You need Go 1.26 or later and Node 22.17 or later.

```sh
make build              # the frontend, then the program, as ./lexicon
make windows            # the program for 64-bit Windows, with its icon, as ./MCMLexicon.exe
make release            # every download, into dist/ (on macOS)
make notices            # THIRD_PARTY_NOTICES.txt, from the licenses of everything built into the program
make test               # Go tests, then type-checking the frontend
make dev LIST=/path/to/instance   # MCM Lexicon plus Vite's dev server at http://localhost:5174
make types              # regenerate the API's TypeScript types after changing them in Go
make icons              # the icon files, from the drawings in assets/
```

`go test ./internal/...` works without Node. Building the program, or `go vet ./...`, needs the frontend built first, since it's embedded from `web/build`. The tests that read real mod lists run only when `snapshots/` holds some (or `LEXICON_SNAPSHOTS` points at them); without any they're skipped, not failed.

To release a version, push an annotated tag such as `v0.1.0`, with the release notes as its message. The Release workflow tests, builds the downloads, and attaches them to a draft release with those notes, which goes live when you publish it.

## Layout

| Path | Purpose |
| --- | --- |
| `cmd/lexicon/` | Flags, startup, opening the browser, and the `scan`, `diff`, and `snapshot` commands |
| `internal/server/` | The web server: host and token checks, serving the frontend, mounting each domain's routes |
| `internal/httpapi/` | Turning typed Go functions into JSON handlers |
| `internal/apperr/` | Errors meant for the user, by kind; the server maps kinds to HTTP statuses |
| `internal/modlist/` | Finding and reading a mod list: an MO2 instance's profile and mod order, or a `Data` folder, as the virtual file order the game sees |
| `internal/bsa/` | Reading Bethesda archives, so MCMs packed in them are read too |
| `internal/plugin/` | Reading plugins as far as their quests and the script properties attached to them |
| `internal/papyrus/` | Papyrus source (`source`) and compiled scripts (`pex`) lowered into one intermediate form (`ir`), and the analysis that reads a menu out of a script (`mcm`) |
| `internal/mcmhelper/` | MCM Helper's `config.json` menus, and the settings files that hold their saved values |
| `internal/translations/` | The game's translation files, for the text behind `$KEY` labels |
| `internal/catalog/` | What MCM Lexicon knows about a list: the scan that builds it, its cache, search, catalog files, and comparing two |
| `internal/recordings/` | MCM Recorder's recordings: reading, checking against the catalog, the draft, saving with backups, and restoring |
| `internal/snapshot/` | Copying everything MCM Lexicon reads from a list, for testing |
| `internal/prefs/` | MCM Lexicon's own remembered settings |
| `internal/gamecheck/` | Whether Skyrim is running, per OS |
| `internal/textfile/` | How files the game reads are changed: atomic writes, backups, line endings kept |
| `internal/testcorpus/` | Small mod lists built in temporary folders, for tests, and the snapshots when there are any |
| `assets/` | The drawings behind the app icon and the mark in the top bar; `make icons` renders the icon files from them |
| `web/src/routes/` | The app's pages; each folder is a URL |
| `web/src/lib/copy/` | The words the app shows, in `en.yaml`, grouped by page; `web/copy.ts` turns them into a typed module (`messages.ts`, generated) as the frontend builds |
| `web/src/lib/ui/` | The look and its shared parts: theme tokens (`theme.css`), the keyboard-shortcut hint strip, sortable heads, row selection, the diff |
| `web/src/lib/<domain>/` | Each domain's API calls and components, mirroring `internal/<domain>/` |
| `web/src/lib/api/` | The API client, and TypeScript types generated from Go (`types/`, don't edit) |
| `tygo.yaml` | Which Go types become TypeScript types |

Domain packages own their logic, request and response types, routes, and tests. Only `textfile` writes files the game reads. The visual system is recorded in [DESIGN.md](DESIGN.md), and the plan it was built from in [PLAN.md](PLAN.md).

## License

MCM Lexicon is released under the [MIT License](LICENSE). Copyright (c) 2026 Davey Strus. It includes other people's software and fonts, each under its own license. [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt) lists them with their license texts, and that's included in releases.

The Elder Scrolls V: Skyrim is a trademark of ZeniMax Media Inc.
