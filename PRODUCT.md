# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Hermaeus's, copied: each concern has its own place, with explicit contracts between them, to keep human review and maintenance easy.

- **Backend:** Go, organized by domain under `internal/` (`modlist`, `plugin`, `papyrus`, `mcmhelper`, `translations`, `catalog`, `recordings`, `snapshot`). Shared infrastructure gets its own packages (`server`, `httpapi`, `apperr`, `prefs`, `gamecheck`, `textfile`), and `cmd/lexicon` holds startup and the command-line modes. Each domain owns its logic, typed request and response structs, routes, and tests. Only `textfile` writes files.
- **Frontend:** Svelte 5 and TypeScript, built with Vite; SvelteKit in static/SPA mode for folder-based routing. Feature folders mirror the Go domains. TypeScript types are generated from the Go structs so contract drift breaks the build. The words the app shows live in `web/src/lib/copy/en.yaml`.
- **Delivery:** the built frontend is embedded in the Go binary, which ships as one program per OS (Windows, macOS, Linux). Pure Go, so cross-compiling needs no C toolchain. Node is needed only to build.

## Users

Primary user: the author, a Skyrim VR player with a heavy mod list (Mad God Overhaul, about 1,800 mods) who uses MCM Recorder to replay Mod Configuration Menu settings on every new game, and who contributes to other lists. Launching the game to find the exact label of an MCM option takes minutes and a headset.

Possible later audience: other MCM Recorder users, and people who maintain mod lists and want to know which MCMs changed between releases. Release is kept open, not committed: nothing may assume the author's machine, paths, mod manager, or lists.

## Product Purpose

MCM Lexicon reads a mod list from disk, with the game closed, and builds a catalog of every MCM: its display name, pages, options, option types, and where possible each option's allowed values. It lets the player browse and search that catalog, build or edit MCM Recorder recordings by picking values, check existing recordings against what the installed mods actually contain, compare two versions of a list, and save recordings in exactly the format MCM Recorder reads. Success means a player finds any MCM option by searching and saves a recording that replays correctly, without having launched Skyrim.

## Positioning

MCM Recorder records clicks in game and replays them, but writing or fixing a recording by hand means knowing every label exactly, and no file lists them. MCM Lexicon is the list of them: offline, read from the mods' own files, and precise about what it knows for certain and what it inferred.

## Operating Context

- Runs on the player's machine next to their mod manager: a Mod Organizer 2 instance (profiles, priority order, `overwrite`), or the game's `Data` folder for Vortex and manual installs.
- Reads plugins, Papyrus scripts (source and compiled), MCM Helper configs, translation files, and MCM Recorder recordings, through the virtual file order the game sees.
- Can open a catalog file exported elsewhere, so a list can be explored on a machine where it isn't installed.
- Builds and runs on Windows, macOS, and Linux; the game itself runs on Windows, or through Wine on the others.

## Capabilities and Constraints

Capabilities: the catalog of menus, pages, and options with their allowed values and certainty; search across every menu; recordings listed, validated against the catalog, edited, and written; catalogs exported and compared; snapshots of a list for testing.

Constraints:

- **Local web UI in a browser:** one program per OS that serves only `127.0.0.1` and opens the browser; every view has its own bookmarkable address, and Back and Forward work.
- **Offline only:** nothing that needs the game running belongs here.
- **Safety:** writes are refused while Skyrim is running (browsing still works); backups precede changes; writes are atomic; requests carry a per-session token. Nothing is written anywhere but the chosen recordings folder, the backups folder, and the files the player names: a catalog file or a snapshot saved from Settings, or a command's output path. A file already there is replaced only when the player says so.
- **Unofficial:** MCM Lexicon is not part of MCM Recorder, SkyUI, MCM Helper, or any mod list, and ships none of their files.

Terminology: menu (an MCM, as SkyUI lists it), page, option, selector (the raw label MCM Recorder matches), text (what the player reads), recording, step, step file, catalog, list, profile, source, certainty (exact, pattern, unknown).

## Brand Commitments

- Name: MCM Lexicon, settled.
- Voice: Hermaeus's: plain and precise, with a dry, slightly wry edge. It says exactly what is saved, where, and when, and what isn't possible and why.
- Visual identity: Hermaeus's design system, inherited whole (`DESIGN.md`). The mark is the face of a Dwemer lexicon with the game's diamond at the center of its dial.
- American English spelling and the serial comma.

## Evidence on Hand

- The author's real lists (MGO 4 RC 4.5 for Skyrim VR, and Nordic Adventures), as snapshots for local testing, with their real recordings as acceptance cases.
- No users, testimonials, or download counts; don't invent any.

## Product Principles

1. **The player's data comes first.** Every write is reversible, minimal, and in MCM Recorder's own format; when in doubt, refuse and explain.
2. **Offline is the point.** Anything that only works with the game running doesn't belong, and anything that doesn't need it shouldn't make the player launch Skyrim.
3. **Say what actually happens.** Name the file, the backup, and the consequence; don't soften or guess. Say what's certain and what's inferred.
4. **Write only what MCM Recorder reads, in its format.** A recording MCM Lexicon writes is one MCM Recorder could have written.
5. **Built for one, releasable to many.** Serve the author's own setup first, but nothing should hard-code that setup or need the author to explain it.
