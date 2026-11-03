# MCM Lexicon: development plan

*A catalog of every Mod Configuration Menu option in a mod list, and an editor for MCM Recorder recordings, built the way Hermaeus is built and looking the way Hermaeus looks.*

This plan is written for Claude Code. It assumes you can read the Hermaeus repository (the author will tell you where it's cloned) and the snapshots that come with it (§10). The facts about file formats below were checked against the author's real mod lists (Mad God Overhaul 4 RC 4.5 for Skyrim VR, about 1,800 mods, and Nordic Adventures), not assumed.

**Nothing in MCM Lexicon's design is specific to one mod list.** The author updates MGO often and contributes to other lists, so per-list knowledge lives in data, not in this plan:

- **snapshots** of a list's MCM files, made by a command
- **catalogs** MCM Lexicon exports
- **acceptance cases** taken from each list's own recordings

A new or updated list needs a new snapshot, not a new plan.

---

## 1. What we're building, and why

**MCM Recorder** (a Papyrus/JContainers mod) records clicks in Mod Configuration Menus and replays them later, so a player doesn't have to reconfigure dozens of MCMs by hand on every new game. Its recordings are plain JSON files, so they can be written by hand. But to write one you need to know the exact labels of every page and option in every MCM, and no file lists them. Finding them today means launching the game.

**MCM Lexicon** reads a mod list from disk, with the game closed, and builds a catalog of every MCM: its display name, pages, options, option types, and where possible each option's allowed values (slider ranges, menu choices). It then lets the player:

- browse and search that catalog
- build or edit recordings by picking values
- check existing recordings against what the installed mods actually contain
- compare two versions of a list
- save recordings in exactly the format MCM Recorder reads

The name is a working title, after MCM Lexicon gro-Shub, the librarian of the College of Winterhold's Arcanaeum, who catalogs everything. Renaming means changing the module name, the program name and the copy; do it early or not at all.

### Decisions already made

- **A separate program, not a Hermaeus feature.** Hermaeus is explicitly about SkyrimNet (`PRODUCT.md`: "Fit alongside SkyrimNet"), and MCM Recorder has nothing to do with SkyrimNet. MCM Lexicon is a sibling app.
- **Same stack and conventions as Hermaeus.** That means Go in domain packages under `internal/`, Svelte 5 + SvelteKit (static/SPA) + TypeScript embedded in the binary, TypeScript types generated from Go with tygo, and copy in `en.yaml`. It's pure Go (no cgo), and ships one program per OS from the same Makefile and release workflow. It also uses the same safety rules: `127.0.0.1` only, per-session token, refuse writes while Skyrim runs, back up before writing, atomic writes. Copy Hermaeus's infrastructure into the new repo rather than importing it; extracting a shared module can come later if both apps keep growing.
- **Same design language as Hermaeus, inherited rather than adapted.** The visual system in Hermaeus's `DESIGN.md` is MCM Lexicon's: the same tokens, fonts, shell, item table, details pane, hint strip and rules, copied with the code that implements them. MCM Lexicon adds only the meanings its product needs (what copper marks, how certainty and validation show, where the File font is used, nothing written before Save) and its own mark. §5.2 says exactly what's copied and what's new, and §5.4 composes every page from those parts.
- **Offline only.** Nothing that needs the game running. In particular, MCM Recorder can't produce this catalog in-game: its patched `SKI_ConfigBase` only tracks options while recording or playing, and `ClearOptionBuffers` marks the tracked set for reset on every page change, so its debug dump only ever holds the current page.
- **Any list, any machine.** MCM Lexicon works on whatever list it's pointed at, and it can also open a catalog file exported elsewhere, so a list can be explored on a machine where it isn't installed.

### Success

- A player points MCM Lexicon at their mod list, finds any MCM option by searching, and saves a recording that MCM Recorder replays correctly in game, without having launched Skyrim to find the labels.
- Existing recordings open as editable steps, with any step that no longer matches the installed mods flagged.
- After a list updates, comparing the old and new catalogs shows which MCMs changed and which recordings broke.

---

## 2. Background you need

### 2.1 MCM Recorder's files (authoritative: its own source, included in the snapshots)

The source is in the MGO snapshot under `mods/MCM Recorder/Scripts/Source/`. Read `McmRecorder_Recorder.psc`, `McmRecorder_Player.psc`, `McmRecorder_McmFields.psc`, `McmRecorder_RecordingFiles.psc`, and its patched `SKI_ConfigBase.psc`. The essentials:

**Location.** `Data/McmRecorder/` as the game sees it. With MO2 that means `McmRecorder/` at the top of any mod folder, or `overwrite/McmRecorder/`.

**A recording** named `<Name>` is:

- `McmRecorder/<Name>.json`, the info file: `{"name": "<Name>", "version": "1.0.0", "author": "<player name>", "autorun": "false"}`. `autorun` is a string. MGO's own recording uses `"True"`, so read it case-insensitively and write `"true"`/`"false"`.
- `McmRecorder/<Name>/NNNN_<Mod>.json`, the step files. Each is a JSON array of actions for one mod. `NNNN` is zero-padded to 4 digits. A new file starts whenever the mod changes, so consecutive actions for the same mod share a file. Playback order is the files' name order. MGO's hand-made recording has two files numbered `0001` (`0001_Be Seated.json`, `0001_VRIK.json`), so read such duplicates and order them by full name, but always write unique sequential numbers.
- In both names, `/` and `\` in `<Name>` and `<Mod>` become `_` (`FileSystemPathPart`). Other characters (spaces, `&`, apostrophes) are kept: `0005_Dirt & Blood.json`.

**An action** (one element of a step array):

| Key | Meaning |
|---|---|
| `mod` | The MCM's display name (`ModName` as SkyUI lists it). The recorder writes `mod`; MGO's file has `Mod`, and replays fine, because JContainers map keys aren't case-sensitive. Read keys case-insensitively and write lowercase. |
| `page` | The page name, exactly as passed to SkyUI (may be a `$KEY`). Omitted for the default page (`SKYUI_DEFAULT_PAGE`), i.e. MCMs without pages. |
| `option` | The option's label (selector), raw: `$EXP_ShowMessages`, not "Show Messages". |
| `index` | Only when several options of the same type on the same page share the label: which one, **1-based**, in the order they were added. Omitted otherwise. |
| `toggle` | `"on"` / `"off"`, for toggles. |
| `slider` | A number (written as a float, e.g. `25.0`), for sliders. |
| `choose` | The menu entry's text, for menu options (the text in the menu's options array, raw). |
| `shortcut` | An integer DirectX scan code, for keymaps. |
| `color` | An integer `0xRRGGBB`, for color options. |
| `text` | A string, for input options. |
| `click` | For text options: the label clicked. With `"side": "right"`, it matches the right-hand value instead (used when the label is empty). The catalog records that value, from `AddTextOption`'s second argument or an MCM Helper text control's fixed `value`, with its certainty, whether the menu changes it (`SetTextOptionValue` in an option event, or a value read from a setting), and an index among the page's text options showing the same value. A right-side step matches a value the menu shows; a miss is NOT FOUND only when every text option's value on the page is known and fixed, and UNCHECKED otherwise. Recording a text option without a label writes a right-side click on its value. |

**Matching on replay** (`McmRecorder_Player.psc`, `AttemptFindOption`): options are found by type, page and selector text. Matching is exact unless the selector is wrapped in asterisks (`*part*`), which means "contains". `index` picks among duplicates. The player also reads an optional `state` key; check `PlayAction`/`FindOption` before relying on it for state-based (`…ST`) options.

Labels must match **exactly**, trailing spaces included. Sofia's slider is literally `"Sofia Health: "`. SkyUI draws an MCM's text as Scaleform HTML, so some mods color or style it (`<font color='#DD3333'>…</font>`, `<b>`, `<br>`): the catalog keeps every label raw, markup and all, since that's what a recording matches, and gives the text for reading (names, labels, values, help, headers, choices) without the markup and with its entities written out, as the game shows it, both from a scan and from an older catalog file.

### 2.2 Where MCM options are defined

Every MCM is a Papyrus script extending SkyUI's `SKI_ConfigBase`, directly or through an intermediate (MCM Helper's `MCM_ConfigBase`, frameworks such as NL_MCM). MGO RC 4.5 has about 70 mods with MCM content, in three groups:

| Kind | Count in MGO RC 4.5 | How MCM Lexicon reads it | Expected fidelity |
|---|---|---|---|
| **MCM Helper** (`MCM/Config/<id>/config.json`) | 13 | Declarative JSON: `displayName`, `pages[].pageDisplayName` / top-level `content`, each control's `type`, `text`, `id`, `valueOptions` (ranges, `options` lists, `defaultValue`). | Complete |
| **Classic, with source** (`.psc` shipped) | 42 | Parse the Papyrus source. | Very good |
| **Classic, compiled only** (`.pex`) | 11 (e.g. Nether's Follower Framework, Sofia, Be Seated, NPC Situational Reactions, SkyUI's own menu) | Parse the compiled script (format below). | Good |

An MCM Helper mod also ships a small compiled script extending `MCM_ConfigBase`. When a `config.json` exists for it, the config is the source of truth.

MCM Helper control types seen in MGO: `toggle` 275, `slider` 203, `header` 91, `empty` 35, `keymap` 21, `CallFunction` 19, `text` 18, `stepper` 11, `enum` 8, `CallGlobalFunction` 5, `hiddenToggle` 1, `color` 1, `input` 1. Map them to SkyUI option types as follows. Treat the mapping as a hypothesis, and confirm it against MCM Helper's documentation and one in-game recording (§9):

- `toggle` → toggle
- `slider` → slider
- `keymap` → keymap
- `color` → color
- `input` → input
- `enum` → menu (`choose` from `valueOptions.options`)
- `text`, `CallFunction`, `CallGlobalFunction` → text option (`click`)
- `stepper` → text option that cycles on click. A recording can only click it, so represent it as "click N times" or mark it unsupported. A Papyrus text option whose click handler shows it literal values in turn is read as a stepper too (§3.3): the handler's If chain that sets the value, when three or more arms each set one value for the same option (the arms are chosen by a counter the click moves), is the cycle, attached to the option its branch is scoped to or, where the handler dispatches by the option's name (VRIK), to the option added with one of the chain's values. A chain of fewer arms, or one setting several values in an arm, is a message (Working, Done, Error), not a cycle. The catalog gives such an option `control: stepper` and its values as `choices`; the info line reads "Each click moves on: Both Hands, Left Hand Only, Right Hand Only". MCM Recorder's player plays each action in a file in turn, a text click as one OnOptionSelect, with no deduplication, so a value N along is N identical click steps. On the menu screen, while editing, a stepper's value shows where the recording's clicks leave it, counted from the mod's default (MCM Helper's, or the first value the script sets), with ×N before it; each click moves it one on, wrapping back to the default, which is no steps (`POST /api/recordings/{name}/steps` with `clicks`: that many identical steps in one run where the first was, 0 removes them, at most 64). The info area says the count holds from the default, so on a new game but not over a changed one. A stepper's run is the one exception to one step per option; the Steps tab marks each click of a run (CLICK 1 OF 2), and unsaved changes and the per-menu Discard count by option.
- `hiddenToggle`, `header`, `empty` → not recordable

### 2.3 Facts about the classic scripts

From reading the MGO snapshot:

- **Display name.** Either assigned in the script (`ModName = "EVG Conditional Idles"` in `OnConfigInit`), or only set as the `ModName` property in the plugin's script data (VMAD) on the quest the script is attached to. Dirt & Blood sets no `ModName` in code, but its `.esp` holds `Dirt & Blood`; Be Seated's `.esp` holds `Be Seated`; Sofia's holds `Sofia Follower`. So **plugins must be parsed** for VMAD properties. The script assignment wins at runtime if both exist, since `OnConfigInit` runs after properties are filled.
- **Pages.** `Pages = new string[N]` then `Pages[i] = "…"` (often in `OnConfigInit`), or a `Pages` property in VMAD.
- **Options.** Calls such as `AddToggleOption("label", value)`, `AddToggleOptionST("STATE", "label", value)`, `AddSliderOption(label, value, format)`, `AddMenuOption(label, display)`, `AddTextOption(label, value, flags)`, `AddKeyMapOption…`, `AddColorOption…`, `AddInputOption…`, each with an `…ST` variant whose first argument is a state name. `AddHeaderOption`/`AddEmptyOption` aren't recordable.
- **Which page an option is on.** Usually decided in `OnPageReset(string page)` by `if page == "Settings"`, but also `if (a_page == Pages[0])` (Experience) or `CurrentPage == Pages[0]`. Some scripts call helper functions per page; follow calls within the script.
- **Menu choices.** Usually string arrays filled element by element (`DirtSettingList[3] = "Deactivated"`), then passed to `SetMenuDialogOptions(array)` in `OnOptionMenuOpen` / `OnMenuOpenST`. Link them to the option via the option-ID variable (`dirtsetting_M = AddMenuOption(…)`, then `if option == dirtsetting_M`) or via the state name.
- **Slider ranges.** `SetSliderDialogRange(min, max)`, `SetSliderDialogInterval`, `SetSliderDialogDefaultValue` in `OnOptionSliderOpen` / `OnSliderOpenST`, linked the same way.
- **Papyrus is case-insensitive.** Identifiers, keywords, and the `ScriptName … Extends …` header are all case-insensitive. Dirt & Blood declares `DirtSettingList` and uses `dirtsettingList`.
- **Labels built at runtime.** Labels like `"Slot " + i`, or follower names, can't be known statically. Record the static parts and offer a `*wildcard*` selector, flagged as uncertain.
- **The MCM can live in a separate mod.** Experience's MCM script, translations and `.esl` are in a separate "Experience - MCM" mod from its main files.

### 2.4 Translations

`$KEY` labels (pages, options and menu entries alike) are recorded raw, but show people the text. Translation files are `Interface/Translations/<name>_<LANGUAGE>.txt`, **UTF-16 LE with BOM**, one `$KEY<TAB>Text` per line (verified: `Experience_ENGLISH.txt` begins `FF FE`, and holds `$EXP_ShowMessages<TAB>Show Messages`). Load all translation files the game would see for the chosen language (default English), and prefer the file named after the MCM's plugin when keys collide. Keys may carry `{…}` arguments; show them as is.

### 2.5 Mod Organizer 2: what the game sees

Hermaeus's `internal/layers` already reads `ModOrganizer.ini` (`selected_profile=@ByteArray(…)`) and the profile's `modlist.txt`. MCM Lexicon needs more, because two mods can ship the same script or config file:

- `modlist.txt` lists the **highest-priority mod first** (verified: DLC entries are at the bottom). `+` is enabled, `-` disabled, `*` unmanaged (DLC/Creation Club). `overwrite/` is above every mod. The game's real `Data` folder (`gamePath` in `ModOrganizer.ini`) is below every mod.
- Build a virtual file index: for each relative path (compared case-insensitively, as Windows does), the winning file and the mod it came from. Keep losing copies too, for a "this mod's version is overridden" note.
- An MCM exists only if its plugin is **active**: the profile's `plugins.txt`, where `*` marks active plugins. A config or script whose plugin is inactive is listed as inactive, not hidden. Treat a missing `plugins.txt` as "all active". Light (`.esl`) and ESL-flagged plugins count too.
- Never use `filepath.Glob` on MO2 paths. Folder names like `[NoDelete] …` are read as patterns (Hermaeus's `install.FilesWithSuffix` exists for this reason). The snapshots deliberately keep such folders.
- **Vortex/manual installs:** the game's `Data` folder is the whole file system, with no overrides.
- **macOS/Linux:** the author runs Skyrim VR on Windows, but MCM Lexicon must build and run anywhere (CrossOver/Whisky/Proton prefixes, or a copied folder or snapshot). Treat every input as plain folders; do all path matching case-insensitively (Hermaeus's `layers.Lookup`).

### 2.6 Archives (BSA): files the game sees that aren't loose

Many mods pack their files into Bethesda archives (`.bsa`) instead of shipping them loose, and some pack their compiled scripts. MCM Lexicon must read those scripts, or their menus are missing from the catalog.

**Verified** (from the author's lists, 2026-11-02):

- Extended Encounters ships `Extended Encounters.esp`, `Extended Encounters.bsa`, and `Extended Encounters - Textures.bsa`, with no loose `Scripts` folder, so its compiled scripts are in the archive. It also ships a loose `Source` folder.
- Its menu, and those of R.A.S.S., Wyrmstooth, and the mod named `$SS_ModName`, are missing from every snapshot so far, and the recorded steps for them report "mod not found" for that reason.
- Extended Encounters' archive, in the author's Skyrim VR list, is **version 105**, Skyrim SE's format. Its flags are `0x3`: folder names and file names are stored, files are uncompressed by default, and names aren't embedded in the file data. It holds 29 folders and 437 files. So VR loads version 105, and Phase 8 reads it first.
- Its MCM's source ships loose, flat in the mod's `Source` folder as `ee_mcmscript.psc`, beside the sources of its other scripts, while the compiled `ee_mcmscript.pex` is in the archive. So the check that a shipped source is the version the game runs must compare a loose source against an archived compiled script. Snapshots made before Phase 8 held none of it, because they found sources only through loose compiled scripts.

- **The format, against real archives.** Nine archives from MGO 4 RC5, every one version 105, with four combinations of flags: uncompressed (`0x3`), compressed by default (`0x7`, `0x87`), and names retained (`0x1b`, `0x1f`). All 1,254 files in them extract, LZ4-compressed scripts included, and every compiled script among them reads. One of them, `Accuracy - MCM.bsa`, packs a whole classic MCM, compiled script, source, and translations together; it reads as a menu of five pages.
- **Against the refreshed snapshots** (all four lists, taken with Phase 8's build): 17 to 20 menus per list are read from archives. Extended Encounters, R.A.S.S., Wyrmstooth, and Sanguine Symphony (the `$SS_ModName` menu) are all in the catalog, and every recorded step in every list matches its option, none of them "mod not found" any more.

**As built (Phase 8), from the format's documentation and the game's known behavior:**

- **Which archives load.** An archive named after an active plugin present in the list: `<plugin>.bsa` and `<plugin> - Textures.bsa`, as Skyrim SE loads them. Plugins load in plugins.txt's order after the game's own masters, or in the mods' order when there's no plugins.txt. An archive whose plugin is inactive or absent isn't loaded. The INI files' archive lists aren't read: they name the game's own archives, which hold no MCMs.
- **What wins.** A loose file wins over the same path in any archive, whatever the mods' order. Between archives, the one whose plugin loads later wins. Under MO2, an archive sits in its mod's folder like any other file, and its contents still lose to every loose file.
- **The format.** A header starting `BSA\0` and a version: 104 for the original Skyrim's archives, 105 for Skyrim SE's, which VR loads (verified above); both are read. Then folder records, file records, and the names, with flags saying whether files are compressed by default, a per-file bit that flips it, and whether each file's data starts with its full name. Version 104 compresses with zlib, 105 with LZ4 frames; both are pure Go, the LZ4 reader MCM Lexicon's own, tested against frames the `lz4` tool writes.
- **What's read from them.** Compiled scripts and sources (`scripts/`, `source/`), MCM Helper configs (`mcm/config/`), and translations. MCM Helper's settings and MCM Recorder's recordings are read only loose, since what writes them can't write into an archive. Archives are read in place, never unpacked to disk, and one that can't be read is listed as skipped with the reason.

---

## 3. Architecture

### 3.1 Repository layout

Mirror Hermaeus. New packages are marked ★; the rest are copied from Hermaeus and trimmed.

| Path | Purpose |
|---|---|
| `cmd/lexicon/` | Flags, startup, opening the browser, and the command-line modes: `scan`, `snapshot`, `diff` (§10) |
| `cmd/notices/` | THIRD_PARTY_NOTICES.txt, as in Hermaeus |
| `internal/server/` | Host and token checks, serving the frontend, mounting routes (copy) |
| `internal/httpapi/` | Typed JSON handlers (copy) |
| `internal/apperr/` | User-facing errors by kind (copy) |
| `internal/prefs/` | Remembered settings: mod list folder, theme, language, last recording folder (copy, new fields) |
| `internal/gamecheck/` | Whether Skyrim is running, per OS (copy) |
| `internal/textfile/` | Atomic writes and backups (copy) |
| ★ `internal/modlist/` | Finding the mod list (MO2 instance, or a game `Data` folder), the active profile, enabled mods in priority order, active plugins, and the virtual file index (`Open(rel) → winning file + owning mod`), with the contents of loaded archives beneath the loose files (§2.6) |
| ★ `internal/bsa/` | Reading Bethesda archives (versions 104 and 105) far enough to list their files and extract the few MCM Lexicon reads, zlib and LZ4 included (§2.6) |
| ★ `internal/plugin/` | Reading `.esp/.esm/.esl` far enough to get quest records' VMAD script properties (`ModName`, `Pages`) and the plugin's light flag. Handles zlib-compressed records. |
| ★ `internal/papyrus/source/` | Tokenizer and a small parser for `.psc`: script header, functions/events/states, assignments, calls, `if` conditions. Not a full compiler: just enough structure for extraction. |
| ★ `internal/papyrus/pex/` | Reader for compiled `.pex`: header, string table, debug info, objects, states, functions, instructions |
| ★ `internal/papyrus/mcm/` | Turns either representation into one intermediate form (§3.3), then extracts menus, pages, options, choices and ranges from it |
| ★ `internal/mcmhelper/` | MCM Helper `config.json` → catalog, plus reading `MCM/Settings/<id>.ini` user values for display |
| ★ `internal/translations/` | UTF-16 translation files → `$KEY` lookup per language |
| ★ `internal/catalog/` | Orchestrates a scan: finds MCM scripts (inheritance from `SKI_ConfigBase` across the virtual file index), joins plugin VMAD data, MCM Helper configs and translations. Caches results keyed by file size+mtime so rescans are quick. Reads and writes catalog files. Routes. |
| ★ `internal/recordings/` | Reading and writing MCM Recorder recordings (§2.1), validating steps against a catalog, routes |
| ★ `internal/snapshot/` | Writing a snapshot of a list (§10), and `diff` between two catalogs |
| ★ `internal/testcorpus/` | Test helpers: small synthetic MO2 instances built in temp dirs (like Hermaeus's `testenv`), plus the opt-in hook to real snapshots (§7) |
| `web/` | SvelteKit app mirroring Hermaeus's `web/`: `lib/ui/` (theme, hint strip and shortcuts, selection and address, sort heads, tabs, segmented choice, theme switch, quick find, clipped tips, line diffs), `lib/copy/` with `copy.ts`, `lib/api/`, `app.html` and the root layout shell, all copied (§5.2); with ★ `lib/menus/`, ★ `lib/recordings/` (the editor, the value editors, and the DirectX scan-code key table) and ★ `lib/compare/` |

The rules are Hermaeus's: domain packages own their logic, typed request/response structs, routes and tests. Only `textfile` writes files. Nothing writes anywhere except the chosen recordings folder, the backups folder, and the output paths a command is given.

### 3.2 Catalog model (Go types, exported to TypeScript via tygo)

```go
type Catalog struct {
    SchemaVersion int       // bump on any incompatible change; readers refuse newer majors
    List          ListInfo  // name (instance folder), profile, game, source path, scanned at, MCM Lexicon version
    Menus         []Menu
}

type Menu struct {
    ID          string   // stable across scans: source kind + owning mod + script or config id
    Name        string   // the display name recorded as "mod", raw (may be $KEY)
    NameText    string   // translated, for people
    Kind        string   // "mcmhelper" | "source" | "compiled" | "framework"
    Mod         string   // MO2 mod folder that wins for the defining file
    Plugin      string   // plugin the quest/script lives in, if found
    Active      bool     // plugin active in the profile
    Script      string   // Papyrus script name (classic), or MCM Helper id
    Pages       []Page   // in display order; a single unnamed page for page-less MCMs
    Certainty   Certainty
    Notes       []string // "ModName not found; using script name", "options added in a loop", …
}

type Page struct {
    Name     string // raw; "" for the default page
    Text     string
    Options  []Option
}

type Option struct {
    Type      string   // toggle | slider | menu | keymap | color | input | text
    Label     string   // raw selector
    Text      string   // translated
    Index     int      // 1-based among same type+label on the page; 0 when unique
    State     string   // ST options' state name, if any
    Slider    *Range   // min, max, step, default, format, when known
    Choices   []Choice // menu entries (raw + text), when known
    Default   *Value   // when the source states one
    Certainty Certainty
    Source    SourceRef // file (virtual path + mod) and line, or pex function, for "show me where"
}

type Certainty string // "exact" | "pattern" (runtime-built; offer *wildcard*) | "unknown"
```

`Certainty` is part of the product, not decoration. The UI shows it, and the recording editor warns before saving a step built on anything but `exact`. `Menu.ID` must be stable across scans and list versions, because `diff` matches menus by it, falling back to `Name`.

### 3.3 Papyrus extraction: one analysis, two front ends

Write the extractor against a small intermediate representation that both `.psc` and `.pex` lower into:

- **Script:** name, parent, properties (name, initial value if constant), states, each state's functions/events.
- **Function:** a list of statements: `Assign(target, expr)`, `ArrayNew(var, size)`, `ArrayStore(var, index, expr)`, `Call(dest?, object?, method, args)`, `If(cond, then, else)`, `Return`.
- **Expressions:** string/number/bool constants, variable and property references, array element `Pages[0]`, comparisons, concatenations. Anything else is "dynamic".

Then the analysis is shared:

1. **Find MCM scripts:** follow `Extends` chains through the virtual file index until `SKI_ConfigBase` (case-insensitive). Treat known intermediates specially: `MCM_ConfigBase` means MCM Helper, which the config.json covers; NL_MCM's framework is marked `framework`, partial.
2. **Constant propagation within a script:** resolve `ModName`, `Pages[i]` and string arrays from `OnConfigInit`, `OnInit`, `OnVersionUpdate`, `OnGameReload`, and any function those call, then `OnConfigOpen`, which SkyUI runs each time the menu opens, before any page. Use plugin VMAD values as the initial state. A property whose `Get` function works its value out (MCM Recorder's `IsSkyrimVR`) is unknown; one whose `Get` returns a constant has that value.
   - **Branches the game decides.** Where an `If`'s condition depends on the game, every arm runs, one after another, so a counter bumped in some arms keeps counting and every option is found (STFU builds its pages so). But where some arm is certain to run, as with an `Else`, and the arms leave a variable differently, it's unknown afterward: Accuracy's `If found / Status = True / Else / Status = False`. Option IDs are pooled, and of two lists where one starts the other, the longer is kept, so a VR-only last page survives. A compiled `ElseIf`, which is an `Else` holding an `If`, reads as its source does.
3. **Page attribution:** walk `OnPageReset`. Each `If` whose condition compares the page parameter (or `CurrentPage`) to a constant or a resolved `Pages[i]` scopes its branch to that page. Follow calls into the script's own functions, with a depth limit. An empty page (`page == ""`) is the splash screen.
4. **Option collection:** each `Add*Option` call yields an option. A constant label is `exact`; a concatenation with constant parts is `pattern`, with the constant parts kept; anything else is `unknown`. Assign `Index` by order within (page, type, label).
5. **Linking dialogs:** remember the variable each `Add*Option` result is stored in (`OID_x = AddSliderOption(…)`), and the state name for ST options. In `OnOptionSliderOpen(int option)` / `OnOptionMenuOpen` and the ST events, attribute `SetSliderDialog*` / `SetMenuDialogOptions` calls to options by matching `option == OID_x` conditions, or by state.

**Lowering `.pex`.** Follow UESP's "Compiled Script File Format" page (Skyrim):

- header: magic `0xFA57C0DE`, version, game ID, compile time, source/user/machine names
- string table
- debug info (function line maps, useful for `SourceRef`)
- user flags
- objects: parent class name, variables, properties (auto-property backing variables), states, then functions with typed parameters/locals and instructions

Turn each instruction into the statement forms above: `assign`, `array_create`, `array_setelement`, `callmethod`, `callparent`, `callstatic`, the comparison opcodes, `jmp`/`jmpt`/`jmpf`, `strcat`, `return`. Reconstruct `If` from conditional jumps over basic blocks; it doesn't need to be a full decompiler, just enough to know which calls are guarded by a page comparison. Temporaries (`::temp0`) are single-assignment in practice, so substitute them forward. Write the reader defensively: bounds-check everything and treat a bad file as "this MCM couldn't be read", never a crash.

**Parsing `.psc`.** Write a tokenizer that handles:

- `;` line comments, `;/ … /;` block comments, and `{…}` doc comments
- `\` line continuations
- strings with escapes (`\n`, `\t`, `\\`, `\"`)

Then a recursive-descent parser for declarations, statements and expressions, with error recovery at line boundaries. No full type checking.

---

## 4. Recordings: reading, validating, writing

- **Recording locations:** list every `McmRecorder/` folder in the virtual file index. The winning copy of each recording is what the game replays; show the owning mod.
- **Where Save writes** (as built in Phase 9): a recording saves back into the mod folder holding the copy of its info file the game reads, overwrite included, so an edit replaces what the game replays. A new recording saves to the folder chosen last time, defaulting to a new mod folder `MCM Lexicon Recordings`; the editor's SAVE TO select changes it, for one recording. Never into the game's real `Data` folder under MO2. A new mod folder isn't in the profile until MO2 lists it, disabled, and the editor says to enable it; meanwhile, and after a restart, MCM Lexicon reads the recordings in every folder it has saved to that the profile doesn't enable, so a recording saved there stays in the sidebar, the select, and Recordings (marked: the game won't play it until the mod is enabled) and can still be added to; a target whose copy loses to another mod's says which. With a catalog file open, the target is any folder the player names by its full path.
- **Deleting** is a change like any other: it goes into the draft, where it can be restored, and Save does it, backing up first. It's only for recordings in folders MCM Lexicon has saved recordings to (remembered per list), never a mod list's own.
- **Validation:** match each step to the catalog by `mod`, `page`, type, `option`, `index`. Results:
  - **matched**
  - **value not allowed:** a menu choice not in the list, or a slider outside the range
  - **option not found:** suggest close labels, since a mod update may have renamed it
  - **mod not found:** missing or inactive
  - **can't check:** the menu is uncertain

  This lint is a headline feature: it tells a player which of their recordings broke when a mod updated.
- **Editing:** steps are an ordered list. The UI groups them under a heading per step file because the file layout does, but order across mods matters and must be editable. Adding an option from a menu or a search result opens the type's value editor, prefilled with the option's default when known, then appends the step to the draft (§5.4).
- **Read-only unless editing.** MCM Lexicon starts read-only, unless a draft from last time is waiting (below), in which case it starts editing so the draft is in view: nothing can be added, made, changed, or deleted, and the server refuses such requests too, so another tab can't either. The player turns on **editing** at the top of the column (§5.3). Turning it off with changes pending asks first: review and save them, discard them, or keep editing.
- **One draft:** while editing, every change, to any recording, goes into one draft the server holds: new recordings, changed steps and values, a different save target, and deletions. It survives a page reload and is the same in every tab, and it's written to `draft.json` beside the settings as it changes and read back at the next start, so closing MCM Lexicon loses nothing; choosing another list drops it. **Undo:** while editing, ⌘Z takes back the draft's last change and ⇧⌘Z puts back the last undone, from any screen, in the hint strip while there's something to step through; a text field keeps ⌘Z for its own typing. One change is one edit as the player made it (a toggle flipped, a stepper clicked, a file typed, a menu's changes discarded), even where the server stores it in two writes. The history is the server's, in memory, up to 200 changes; Save, leaving editing, or another list starts it afresh, since a write to disk can't be undone. A new change drops what was undone. A recording edited back to what's on disk drops out of it. Nothing touches disk until the draft is saved, from SAVE in the column (§5.3) or the **Changes** page (§5.4), all of it at once: every new and changed recording written and every deletion done, each backed up first. Save writes nothing unless every recording in the draft can be written, refuses while Skyrim runs, and on a failure part way says what it had done, leaving the rest in the draft.
- **Writing:**
  - refuse while Skyrim runs (`gamecheck`), like Hermaeus
  - back up the existing recording folder and info file (copies under `lexicon-backups/<recording>/<time>/`, beside the list where the game doesn't look: the MO2 instance's folder, or the game folder for a Data folder)
  - write the info file and step files atomically: write them to a temporary folder beside the target, then swap
  - write keys in lowercase, in the alphabetical order JContainers writes them (as every recording in the snapshots has them), slider values as floats (`25.0`), and toggles as `"on"`/`"off"`; a click step has `click` and no `option`
  - number step files `0001…` in order, with a new file whenever the mod changes
  - apply `FileSystemPathPart` to names, and turn the other characters Windows refuses in a file name into `_` too
  - a file Save would only reformat (keys in another case or order, `True` for `true`) counts as unchanged, since MCM Recorder reads it the same
  - pretty-print with 2-space indents, which matches what JContainers writes
- **Non-ASCII labels.** Write JSON as UTF-8. JContainers' handling of non-ASCII characters against Skyrim's strings isn't verified, so flag such steps with a note until the author confirms one in game.

---

## 5. API and pages

### 5.1 API

API routes follow Hermaeus's shape (`httpapi.JSON`, typed structs, `apperr` kinds):

| Route | What |
|---|---|
| `GET /api/status` | Source (live list or catalog file), profile, language, theme, Skyrim running, scan state (idle, or running with mods done and total, and when the last scan finished), the open recording (§5.3) |
| `PUT /api/source` | Choose an MO2 instance / `Data` folder (and profile), or a catalog file; remembered |
| `PUT /api/theme` | `system`, `dark` or `light`; remembered (Hermaeus's route, copied) |
| `POST /api/scan` | Rescan (incremental by default; full on request). The list is also rescanned by itself when the MO2 profile's `modlist.txt` or `plugins.txt` changes: the server looks every 2 seconds, compares the files' contents with what they held when the last scan began (MO2 rewrites them unchanged too), and rescans once two looks agree, so MO2 has finished writing; the page follows the scan as it does any other. A Data folder isn't watched, since the game keeps its plugins.txt elsewhere, and neither is a catalog file. |
| `GET /api/menus` | Catalog summary: each menu's name, kind, mod, counts, certainty |
| `GET /api/menus/{id}` | One menu with pages and options, plus saved MCM Helper values where its `MCM/Settings/<id>.ini` exists |
| `GET /api/search?q=` | Options across all menus by label, text, page or mod |
| `GET /api/catalog` | The whole catalog as a downloadable file |
| `POST /api/compare` | Diff the current catalog against a catalog file at a given path |
| `GET /api/recordings` | Recordings found, with owning mod and validation summary |
| `PUT /api/recordings/open` | Which recording new steps go to (§5.3), or none; remembered |
| `GET /api/recordings/{name}` | Steps with per-step validation; the draft instead, when one exists |
| `POST /api/recordings` | A new recording, as a draft (name, author, autorun) |
| `PUT /api/recordings/{name}/draft` | Replace the draft's steps, info, and save target (§4, drafts) |
| `POST /api/recordings/{name}/steps` | Append one step to the draft, from adding (§5.4); says its number |
| `PUT /api/editing` | Turn editing on or off (§4); off with changes pending needs `discard` |
| `DELETE /api/recordings/{name}/draft` | Discard one recording's changes from the draft |
| `POST /api/recordings/{name}/restore` | Take a recording's deletion out of the draft |
| `GET /api/changes` | The whole draft, recording by recording, with every file Save writes or removes |
| `PUT /api/recordings/{name}/files/{n}` | Replace one file with its text as typed: 0 the info file, n the nth step file, whose run of steps the text's take the place of (§5.4, Files) |
| `POST /api/changes/save` | Save the whole draft (§4) |
| `POST /api/changes/undo`, `POST /api/changes/redo` | Step back through the draft's changes, or forward again (§4, undo) |
| `DELETE /api/changes` | Discard the whole draft |
| `DELETE /api/recordings/{name}` | Put its deletion in the draft; only in a folder MCM Lexicon saved to (§4) |

Scans can take a while on 1,800 mods. Run them in the background with progress shown in status, and serve the previous catalog until the new one is ready. Cache parsed scripts by (path, size, mtime) in the OS cache dir.

### 5.2 The design language: Hermaeus's, inherited

MCM Lexicon looks like Hermaeus because it *is* Hermaeus's design system, applied to a sibling. Read Hermaeus's `DESIGN.md` before building any page; it is the authority, and nothing in this plan overrides it. The North Star is the same: one more menu of the game you just closed. Every list screen is the sections column, a sortable item table with the diamond on the selected row, and a full-height details pane, on smoked glass with hairline rules, driven from the keyboard, with the button-hint strip along the bottom showing real shortcuts.

**Copied, not adapted.** These come over from Hermaeus's `web/` with nothing changed but names and the comments that mention SkyrimNet: `lib/ui/theme.css` (every token in both themes, the gradient, the controls), the three @fontsource packages (Jost 400 and 500, Barlow Condensed 500 and 600, DM Mono 400), `ui/HintBar.svelte` with `ui/shortcuts.svelte.ts`, `ui/Diamond.svelte`, `ui/SortHeader.svelte` with `ui/sort.ts`, `ui/Tabs.svelte`, `ui/Segmented.svelte`, `ui/ThemeSwitch.svelte`, `ui/Up.svelte`, `ui/selection.svelte.ts` with `ui/address.svelte.ts` (the selected row and what the pane is doing live in the address, as `?item=` and `?mode=`), `ui/fit.ts` (the pane ends at the hint strip; a form's footer sticks above it), `ui/clipped.ts` with the `.tip` style, `ui/linediff.ts`, `copy/Fill.svelte`, `web/copy.ts` and the generated `copy/messages.ts`, `api/client.ts`, `api/live.svelte.ts` (Skyrim's state, polled), `format.ts`, `app.html` (its token and theme placeholders renamed), the root `+layout.svelte` shell with its grid and breakpoints, and `+error.svelte`. Hermaeus's quick-find dialog (`characters/QuickFind.svelte`) comes over as `ui/QuickFind.svelte`, finding menus instead of characters. **Nothing in `theme.css` changes:** no new token, no new font, no new radius, no new shadow. If a page seems to need one, it's composed wrongly; recompose it from the parts above.

**Rules that carry over unchanged**, by the names `DESIGN.md` gives them: The Scarce Accent Rule (the accent means selected, actionable, or a group heading inside a list, and never fills anything larger than a button), The One Token Set Rule (light is the same roles with other values; a new color ships as a pair or not at all), The Hairline Rule (separate with 1px rules; no cards, no shadows except under the dialog and the tip), The Label Versus Read Rule (what labels is tracked Barlow Condensed caps; what is read is Jost in sentence case), The Lined-Up Figures Rule (tabular figures; numeric columns right-aligned), the Item Table and Details Pane as specified, and the hint strip's two rules: no hint without a working shortcut, and Enter on a focused link or button is never a shortcut.

**What MCM Lexicon gives its own meaning**, because its product differs. These are the only additions to the system. The new repo's `DESIGN.md` records them in place of Hermaeus's product-specific rules (the Template Markup Rule and the Diary Dateline go; everything else stays).

- **Copper (`changed`) marks what the player set, where the mod's default is known.** In a menu's option table, a saved MCM Helper value (from `MCM/Settings/<id>.ini`) that differs from the config's `defaultValue`. In a recording, a step's value when the option's default is known and the step sets something else. It marks the value itself, never a row or a fill, and never doubles as a warning. A step that sets an option to its own default is therefore *not* copper, which is useful: it's a step that does nothing.
- **Certainty shows as words, not color.** An `exact` option shows nothing extra. `pattern` and `unknown` show as a muted label-caps word after the option's text in tables (PATTERN, UNKNOWN), and in the pane as an amber notice saying what it means: the label is built at runtime, so a recording can only match it with a `*wildcard*`, or can't match it at all. Certainty is never a column of its own; most rows are exact, and a column of EXACT is noise.
- **Validation is amber (`warn`) when a step won't replay**, muted when it can't be checked, and silent when it matches. A step's flag is a label-caps word in the step table (NOT FOUND, NOT ALLOWED, MOD MISSING, UNCHECKED) and, in the pane, a sentence that says what the catalog has instead, with the suggestion as a text button that applies it. A recording that couldn't be read at all is an error (`danger`), as in Hermaeus.
- **The File font (DM Mono) is for anything matched character for character:** raw selectors, page names and menu entries as the recording will hold them, the files the editor previews, and catalog and recording paths. The translated text is for reading, in Jost; the raw selector sits beneath it, muted, in DM Mono, only when the two differ or when the selector has a space at either end. A trailing space is invisible in any font, so copy names it: "Ends with a space. MCM Recorder matches it exactly."
- **The Draft Rule.** Nothing is written until the draft is saved, and only while editing can there be a draft. Each recording with changes says so in a muted line under its title, naming the folder Save will write to, with REVIEW CHANGES to the Changes page, where Save is. Hermaeus's "say what actually happens" is the test for every notice: name the file, the backup, and the consequence.

### 5.3 The shell

The grid is Hermaeus's: a top bar, a 13rem sections column, main (1.5rem top, 2rem sides, 5rem bottom to clear the hints), and the fixed hint strip. Under 60rem the column becomes a scrolling row under the bar, and panes stack under their tables with a top rule instead of a left one.

**Top bar.** The mark and MCM LEXICON at left, in headline caps. At right, in this order:

- **Source:** with a live list, PROFILE and a select of the instance's profiles, the counterpart of Hermaeus's playthrough select; changing it rescans. With a catalog file, CATALOG and the file's name, since there's nothing to choose. The select's `title` is the path on disk.
- **Scan state,** label caps in muted ink: SCANNING · 412 OF 1,819 MODS while a scan runs (the count comes from `/api/status`, polled the way Skyrim's state is), then SCANNED 11:02 PM. A scan error is a `danger` notice on the Menus page, not in the bar.
- **Skyrim's state:** SKYRIM CLOSED muted, or SKYRIM RUNNING · READ-ONLY in amber, exactly as Hermaeus.
- **Theme switch:** Hermaeus's.

There are no quick toggles, so the bar is one line; under 60rem its right cluster wraps as Hermaeus's does.

**The column** is laid out as SkyUI's MCM lays out its left side, in three bands over hairlines: the editing session at the top (below); the sections SEARCH, COMPARE, and RECORDINGS; and MENUS, which takes the rest of the height: its count beside its label (a link to the list's home), a filter field, and every menu by display text in alphabetical order, the one on screen in the accent with the diamond, and inactive menus muted below an INACTIVE label. How sure a label is shows on its option, not in the list. The list scrolls on its own. There is no listing of recordings in the column: the chosen one is linked from the session, and the rest are on the Recordings page.

**The editing session** sits at the column's top, always visible: the Segmented READ-ONLY / EDITING, read-only at every start (§4); RECORDING (ADDING TO while editing) and the select of the **open recording**, the one to look at and, while editing, the one A adds steps to from any menu or search result (§5.4), remembered with the source (`PUT /api/recordings/open`), its last entry while editing NEW RECORDING…, which asks for the name at once in a dialog on the smoke surface and chooses what it makes (cancelling leaves the select as it was), with a muted CHOOSE A RECORDING in place while none is chosen; the chosen recording's name as a link to its page, with the hollow diamond, in full ink while it has unsaved changes and in amber when its mod is disabled; and, while editing, CHANGES with the draft's count, the way to the Changes page; once the draft holds anything, in its place an amber SAVE button (DESIGN.md, the editing session), which writes the whole draft as the Changes page's Save does (⌘⏎ does the same from any screen, in the hint strip), REVIEW beside it to the Changes page, and a muted line saying how many recordings have changes not yet saved. A save that can't be written goes to the Changes page, which says why; after one, a line says how many recordings were saved. Under 60rem it's one row under the top bar.

**Hint strip.** As Hermaeus's: right-aligned, each hint a real shortcut, showing only what the current screen offers; the hints are listed per page in §5.4. `/` opens Quick Find from any page. Esc runs the page's Up link (back to the list an item came from) unless a form claims it for Cancel.

**Quick Find** (`/`): Hermaeus's dialog on the smoke surface, listing menus by display text, twelve at a time, the diamond on the selected one; Enter opens the menu.

### 5.4 Pages

Every page has its own URL, and the selected row and the pane's mode live in the address (`?item=`, `?mode=`), so Back closes a form or returns to the entry before it, as in Hermaeus. Each page's table columns, pane contents and hints are given here so that the surface briefs (§5.6) and the copy can be written from them.

**First run** (`/`, no source). Hermaeus's folder form, in its voice: WHERE IS YOUR MOD LIST? in display caps; a sentence saying what MCM Lexicon looks for (the folder holding `ModOrganizer.ini`, or a game `Data` folder); a field for the folder and a primary USE THIS FOLDER. Under a hairline, the second way in: OPEN A CATALOG FILE, a path field and a plain button, for a machine without the list. Errors are `danger` notices under the button, naming the path tried.

**Menus** (`/menus`). The list's home, shown while no menu is chosen from the column: MENUS in display caps with the count ("65 menus · 3 inactive"), the list's path and profile, a sentence saying to choose a menu from the column, a scan error as a `danger` notice, and the scripts that aren't menus under a SHOW THEM text button, with unreadable ones as an amber notice. The menus themselves are the column's list (§5.3). Hints: R RESCAN (disabled while scanning) · / FIND.
Hints: ↑ ↓ (hidden) · ENTER OPEN · R RESCAN (disabled while scanning) · / FIND.

**A menu** (`/menus/{id}`, each page `/menus/{id}/{n}`), drawn as SkyUI draws it (DESIGN.md, The MCM Screen). The menu's name in display caps; under it, muted, the mod, the plugin and the kind joined by middots; for an inactive menu, an amber notice. A page list down the left with the diamond on the current page and each page's option count; a page-less menu has none, and the empty-string splash page is left out unless it has options, and also when it's a copy of a named page: a script that answers "" as it answers its first page (`If page == "VRIK Options" || page == ""`) draws the screen the menu opens on, which the game doesn't list. The catalog marks such a pair (`sameAs` on the default page, `landing` on the named one) only when every option matches in type, label, position, and the script line that adds it, and the same headers and gaps; labels alone aren't enough (VRIK's Leg and Arm Holsters share theirs). **Disabled options.** An option added with `OPTION_FLAG_DISABLED` (a version number, a check that passed) is drawn greyed out and can't be selected, by the player or by MCM Recorder: the catalog marks it `disabled: always`, the menu screen draws it muted and read-only, Record all and A leave it out, and a recording's step on it is NOT ALLOWED ("the step does nothing"). Flags from a value the game decides, a flag that differs between an If's arms, or a `SetOptionFlags` call make it `disabled: sometimes`, which stays recordable with a notice. The menu then opens on the named page, Search finds the option there once, and a recording's step on the default page counts as that option: it shows as recorded, and a change to it stays on the page it's on rather than adding a second step. The default page stays in the catalog, so recordings that use it still validate. The options on the game's grid of two columns, each where SkyUI's cursor put it (§3.3's cursor positions): the option's text at the left, its value at the right edge, headers as accent group rows, empty options as blank rows, and a slot nothing fills blank. A page the script places at runtime is one column in added order, without headers. The value is the recorded one when the open recording has the option, else the menu's known value (saved, else the default; a dash when unknown), copper where it differs from a known default. The arrow keys move as on the game's grid: ↑ ↓ within a column, ← → to the nearest option in the column beside (between two as near, the one above), and ← from the left column into the page list, then ← again into the column's list of menus (from the options straight there for a menu without pages). In either list ↑ ↓ open the page or menu above or below, in the order and filter the column shows, replacing the history entry, and quick presses each move one further; → comes back toward the options. A click anywhere takes the keys back to the options. Under the grid, the info area: the selected option's help, then one muted line with SELECTOR (in DM Mono, "Ends with a space" named where true), INDEX, TYPE, what it allows, DEFAULT, SAVED, DEFINED IN (file and line, and the archive), and its step in the open recording; the certainty notice when not exact, and the note when a text option's value changes. No details pane.
**While editing** (§4), with a recording chosen under ADDING TO, the menu page is where a recording is built. Each row's value is the option's value in that recording, copper where it differs from a known default, in a cell drawn as a field so it reads as editable (an option not recorded yet shows the value it has, saved or default, muted, so defaults stay in view while editing), with a hollow diamond before a recorded value and an amber one before a value the draft hasn't saved yet (DESIGN.md, The MCM Screen), with DISCARD N UNSAVED CHANGES TO THIS MENU beside Record all while there are any, which puts this menu's steps back as saved (a step the saved recording hasn't is removed) and leaves the rest of the draft, and is edited in place: A text option has nothing to set: its value shows as plain text, a click on it records a click step (the way a button is pressed), the word CLICK then stands before its value, and the info area explains that nothing looks different because the step is the click. a toggle whose value is known is a sliding switch (Space flips the selected one), dimmed while it isn't recorded, showing what the menu has so a click records the other; any other value, and a toggle whose value isn't known, turns into its compact editor when clicked or on E, a dash standing in while it isn't recorded. Any change records the option: a new step at the end, or its step changed, since a recording holds an option once. RECORD ALL SETTINGS ON THIS PAGE records every option whose value is known (saved, else the default) and whose label is exact; options recorded already keep their values, and a line says how many were recorded, kept, and skipped. A records the selected option as it shows (an option whose value isn't known opens its editor instead); with no recording chosen, A asks for one in a dialog on the smoke surface, listing the recordings found and NEW RECORDING…. The Search page's pane keeps the IN <RECORDING> section with the type's value editor and APPLY or ADD TO <RECORDING>.
Hints: ↑ ↓ (hidden) · while editing, SPACE TURN ON/OFF or E EDIT (the selected option's value) and A ADD TO <RECORDING> (an option not recorded yet) · ESC MENUS · / FIND.

**Search** (`/search?q=`). SEARCH in display caps, one search input focused on arrival, the query kept in the address. Results are an item table of options across every menu, **grouped under accent group-heading rows per menu** (Hermaeus's day and plugin headings), with columns Option, Page, Type and Allowed, and the same pane as a menu's option, A included. Empty: Hermaeus's "Nothing matches "…"". Past a few hundred results, cut the list and say so in a muted line; search is for finding, the menu page is for browsing.
Hints: ↑ ↓ (hidden) · ENTER OPEN MENU · A ADD TO <RECORDING> · / FIND.

**Recordings** (`/recordings`). RECORDINGS in display caps, the count, a filter. The item table: diamond; Recording; In (the mod folder holding the winning copy, or OVERWRITE, with "also in <mod>" muted when another copy loses); Steps and Menus as numbers; Autorun as ON or OFF in label caps; Issues (nothing when every step matches, else "3 WON'T REPLAY" in amber, or "2 UNCHECKED" muted). The pane: the name in headline caps; AUTHOR, VERSION, AUTORUN, FOLDER (the path), FILES (the count, and the backup folder's path once one exists); the validation summary as sentences; a primary OPEN; and a plain OPEN FOR ADDING that makes it the open recording without leaving the page.
New (`?mode=new`, while editing): the pane form asks for the name (with a note that `/` and `\` become `_`), the author (remembered), and autorun; CREATE makes the recording in the draft and opens it. DEL puts the selected recording's deletion in the draft at once, with no confirmation, since the draft is reviewed before anything happens; the row then says DELETED IN THE DRAFT, and R or RESTORE takes it back out.
Hints: ↑ ↓ (hidden) · ENTER OPEN · and, while editing, N NEW · DEL DELETE (disabled for a recording outside the folders MCM Lexicon saved to, with the reason in the pane) or R RESTORE · / FIND.

**A recording** (`/recordings/{name}`): the editor, and the one screen MCM Lexicon composes without a Hermaeus twin. Up: RECORDINGS. The name in display caps; under it, muted, the folder it lives in and the step and file counts; and, when the draft differs from disk, the Draft Rule's line: "Not saved. Save writes 4 files to <folder>, backing up the current ones first." Tabs: STEPS (count), FILES (count), and INFO.

- *Steps:* an item table of the draft's steps in replay order, **grouped under accent headings per step file** (0001 · DIRT & BLOOD), because a new file begins whenever the mod changes and the file layout is what MCM Recorder reads. Columns: diamond; # (the step's number in the whole recording, right-aligned); Option (text, raw beneath as elsewhere); Page; Type; Value in the type's own words (ON, 25, "Deactivated", a key's name with its scan code muted beside it, a 9px color square beside its hex, text in quotes; copper when it differs from a known default); Issues (the amber or muted word from §5.2; nothing when matched). Steps are selected with ↑ ↓, and **values change in the table**, since opening a form to flip a setting is a step too many: a toggle's Value cell is a sliding switch, flipped by a click or Space; any other value's cell becomes its type's smallest editor when clicked or on E (`?mode=edit`), a number field for a slider, a select for a menu, a key capture, a color input, or a text field, where Enter applies and Esc cancels, and a menu entry, color, or key applies as soon as it's chosen. A click step has nothing to set. Every change goes to the draft. The draft line and Save keep their place whether or not anything has changed, so the table never moves under the pointer. The pane shows the step's option as the menu page does; its check as a sentence with the suggestion as a text button ("The catalog has "Dirt on the Player". Use it"); the value; and EDIT (TURN ON or TURN OFF for a toggle), MOVE UP, MOVE DOWN, REMOVE, with OPEN IN MENUS as a text button to the option's own page.
- *Info:* the info file: NAME (the name MCM Recorder shows in its list; the files keep their name, which is how MCM Recorder finds the recording, and the field says so), AUTHOR, VERSION, and AUTORUN as a Segmented OFF / ON. While editing, each goes into the draft as it changes, a text field when it's left or on Enter (Esc puts it back); read-only, they're text. A name or author changed only in case is a change, since people read them; MCM Recorder's own comparisons ignore case everywhere else.
- *Files:* every file Save will write, as it will be written, in the File font: the info file, then each step file by name. Where the file exists on disk, differences from it are marked as Hermaeus marks a bio's earlier version (`linediff`, `ins` and `del`); a new file is all `ins`. This is the JSON preview from §4, and the proof the Draft Rule promises. While editing, each file has EDIT AS TEXT: the file's text in a mono field, and APPLY (⌘⏎) reads it as MCM Recorder reads it into the draft (`PUT /api/recordings/{name}/files/{n}`), the typed steps taking the place of that file's run, or the info file's fields; a key MCM Recorder doesn't read, a step with no value key or two, a value of the wrong kind, or a step naming no menu or option is refused with the step's number and nothing changes. The draft is then checked and shown as any other edit; Save writes MCM Recorder's own form, so spacing and key order can differ from what was typed, which the diff shows. Nothing is ever edited on disk directly.

Read-only, the editor shows what's on disk: values as text, no checkboxes or editors, no pane buttons. While editing, nothing on this page writes; the draft line's REVIEW CHANGES goes to Changes.

**Changes** (`/changes`, while editing). CHANGES in display caps; a sentence saying how many files Save writes or removes for how many recordings, and that changes last until saved, discarded, the program closes, or another list is chosen; the primary SAVE ALL (⌘ ENTER) and a plain DISCARD ALL. Then each recording in the draft under an accent heading with NEW, CHANGED or DELETED, the folder it's written to or removed from, its target notices (a new mod folder, a copy that wins over it, a problem), a text button to discard its changes (or keep it, for a deletion), and every file that changes, as a line diff against what's on disk. After Save, a line per recording names what was written and where the previous files went. Hints: ⌘ ENTER SAVE ALL · / FIND.
Hints: ↑ ↓ (hidden) · while editing, SPACE TURN ON/OFF (a toggle) or E EDIT (any other value) · U MOVE UP · D MOVE DOWN · DEL REMOVE · ESC RECORDINGS (CANCEL while a value is being edited) · / FIND.

**Value editors**, one per option type, for adding a step from a menu or a search result; editing a step's value uses the compact ones in its table cell, above. Each uses Hermaeus's field style (smoked field, strong hairline, the label in muted caps above) and has a keyboard path:

- toggle: a sliding switch, with ON or OFF beside it
- slider: when its range is known, a slider (DESIGN.md, Range) with a number input beside it for an exact value, the range named under them ("0 to 100, by 5; default 25"); without a range, the number input alone; a value outside the range is refused with a sentence. In a table cell, the slider applies as the pointer lets go and the field on Enter
- menu: a select of the choices, showing text and storing raw; a choice not in the list (from an older recording) stays selectable, marked "(not in this menu now)", so it can be seen and replaced A disabled CHOOSE AN ENTRY heads the list, chosen when no entry is known; otherwise the entry the option has is selected, and Enter records the entry shown (choosing the entry already selected fires no change). The default entry is marked (DEFAULT) unless its text already says so.
- keymap: Hermaeus's hotkey capture (the "Press the key to use" button in `settings/SettingsTable.svelte`, with `settings/keys.ts`), with the table swapped: SkyrimNet's hotkeys are Windows virtual-key codes, MCM Recorder's are DirectX scan codes, so write `keys.ts` from `KeyboardEvent.code` to scan code (mouse buttons from 256 up), and show the key's name beside its code
- color: a color input and the hex, stored as `0xRRGGBB`
- input: a text field
- text: nothing to set, the step is the click; with an empty label, a Segmented FOUND BY of LABEL / VALUE (§2.1), with a line saying MCM Recorder then finds the option by the value shown on its right. Copy never says "right-side click": a step is found by its label or by its value

**Compare** (`/compare`). COMPARE in display caps. First the choice: the current catalog on the left, named (the list, the profile, when it was scanned), a path field for the other, and a primary COMPARE; the result keeps both paths in the address. Results: the pane carries the summary as Hermaeus's Overview carries Stored, a stat list under CHANGED in accent label caps with the rows MENUS ADDED, MENUS REMOVED, OPTIONS ADDED, OPTIONS REMOVED, OPTIONS RENAMED, VALUES CHANGED, STEPS THAT BREAK, each a doorway that filters the table (The Numbers Are Doors Rule: the diamond marks the row the arrow keys reach, Enter applies it). The table: changes grouped under accent menu headings, with What (page · option), Change as label caps (ADDED, REMOVED, RENAMED, RANGE, CHOICES, DEFAULT, CERTAINTY), Before and After, removed values as `del` and added as `ins` as in Hermaeus's bio diff. Rows for steps that break name the recording and step. Empty: Hermaeus's "Nothing differs."
Hints: ↑ ↓ (hidden) · ENTER OPEN (the menu in the current catalog, when it still exists) · ESC CHOOSE AGAIN · / FIND.

**Errors:** Hermaeus's error page: NOT FOUND or SOMETHING WENT WRONG, the message, and a link back to Menus.

### 5.5 Copy

Copy goes in `web/src/lib/copy/en.yaml` with Hermaeus's conventions: lowerCamelCase keys grouped by page, `one:` and `other:` forms, typographic quotes and apostrophes. The voice is Hermaeus's: plain and precise, with a dry edge, and it says exactly what's written where. Reuse Hermaeus's `common`, `shell`, `folders`, `ui` and error strings where they fit, with the product renamed; MCM Lexicon's own groups are `menus`, `menu`, `search`, `recordings`, `editor`, `values`, `compare` and `scan`. Terms, used the same way in copy, code and this plan, and never mixed: **menu** (an MCM, as SkyUI lists it), **page**, **option**, **selector** (the raw label MCM Recorder matches), **text** (what the player reads), **recording**, **step**, **step file**, **catalog**, **list** (the mod list), **profile**, **source**.

### 5.6 Checking the pages

A page isn't done until it has been looked at: at 1280, 1440 and 1600 wide and stacked under 60rem, in both themes, with real data from a snapshot (§10), beside Hermaeus's own screens for the same part (its `.impeccable/review/` captures are the reference). Check WCAG AA contrast on every new use of copper or amber (the values pass on both grounds, but a new use on another surface must be checked), visible focus on every control, and a complete keyboard path through every page using only the hints the strip shows. Copy Hermaeus's `.impeccable/config.json` without its `site` project root and its ignore rules, so the design detector runs on UI edits, and write a surface brief per page under `.impeccable/surfaces/` from §5.4 as each is built, with Mode: Operate.

---

## 6. Build and release

Copy Hermaeus's `Makefile` (build, windows, release, notices, types, test, dev, icons), `.github/workflows/release.yml`, `tygo.yaml` layout, `web/copy.ts`, `web/third-party.ts` and `LICENSE`/notices approach, renaming throughout.

**The mark and icons.** The icon pipeline (`make icons`) is copied as is; the drawings are new (§9). The mark follows Hermaeus's in every way but subject: one drawing in `assets/`, used in the top bar in the text color at 1.35em with its central shape cut out so the ground shows through, and as the app icon and favicon in verdigris on the game's near-black in a rounded square, with a simpler drawing for the 16-pixel size. It must read beside the wordmark in both themes and at 16 pixels, which rules out fine detail and lettering.

**`PRODUCT.md` and `DESIGN.md`.** `PRODUCT.md` is written for MCM Lexicon from §1 of this plan (run `/impeccable init` in the new repo and answer from the plan); its product principles carry over from Hermaeus almost word for word ("The player's data comes first", "Say what actually happens", "Built for one, releasable to many"), with "Fit alongside SkyrimNet" replaced by "Write only what MCM Recorder reads, in its format". `DESIGN.md` is Hermaeus's, copied whole, with only these edits: the name, description and mark paragraph; §5.2's copper meaning and its certainty, validation, File-font and Draft rules in place of the Template Markup Rule and the Diary Dateline; the page list under Layout; and the Quick Find finding menus. Its frontmatter tokens don't change. Copy `.impeccable/config.json` too (§5.6).

---

## 7. Testing

**Committed tests (always run).** Realistic synthetic inputs built in temp dirs by `internal/testcorpus`:

- an MO2 instance (`ModOrganizer.ini`, profile `modlist.txt` and `plugins.txt`, folders with brackets and apostrophes, a **disabled mod** whose MCM must not appear, an **inactive plugin**, and a profile with no `plugins.txt`)
- hand-written `.psc` snippets covering each pattern in §2.3
- hand-built `.pex` bytes (write a tiny encoder in test code for the instruction subset used)
- a hand-built minimal plugin with a QUST record, a VMAD `ModName` property, and one compressed record
- hand-built archives in both versions, with compressed and uncompressed files and embedded names, plus a loose copy of one file that must win over the archive's
- translation files written as UTF-16 LE with BOM
- MCM Helper configs with both page styles

Plus round-trip tests for recordings: read the MGO-style files, including the duplicate `0001` numbering and the `Mod` key casing, then write and compare. Also round-trip catalog files, and test `diff` on hand-built catalog pairs.

**Snapshot tests.** The real lists' snapshots, unpacked under `snapshots/` in the repository (gitignored), are where the author tests live, and the acceptance tests read them too: `go test ./...` runs them whenever that folder holds any (`LEXICON_SNAPSHOTS` points somewhere else), and a test that finds none is skipped, not failed, so the committed tests pass on a machine without them. **Never commit snapshots:** they're other authors' work, for local testing only.

There are two kinds of acceptance cases:

- **Generic, for any list.**
  - Every step of every recording in the snapshot must resolve to an `exact` option, or report **mod not found** when its MCM isn't in the list. Read the steps from the snapshot's recordings, not from a table. `SNAPSHOT.md` lists them for people.
  - The scan must finish without a crash or an unreadable script.
  - Every MCM Helper config and every MCM script in the snapshot must appear in the catalog, or in its list of skipped files with a reason.
- **Golden cases** for specific behaviors, all in the MGO RC 4.5 snapshot. Each one is skipped when its mod isn't in the snapshot being tested, so they keep working as lists change. Every one of these steps must resolve to a catalog option with `Certainty == exact`:

| Recording step | Menu kind | What it proves |
|---|---|---|
| `Dirt & Blood` / `Settings` / menu `Dirt on Player` → `Deactivated` | source | Display name from **plugin VMAD**; menu choices from an array filled element by element (`DirtSettingList[3]`), with mixed-case identifiers |
| `Dirt & Blood` / `Settings` / toggles `Dirt on NPCs Based on Profession`, `Dirt on Bandits` | source | Page attribution via `page == "Settings"` |
| `EVG Conditional Idles` / `Toggle - NPCs` / toggles `Cold`, `Modesty Male`, `Modesty Female`, `Injured` | source | `ModName` assigned in script; `AddToggleOptionST` (state + label) |
| `Experience` / `$EXP_General` / toggle `$EXP_ShowMessages` | source | Raw `$KEY` selectors; page via `Pages[0]`; English text "Show Messages" from UTF-16 translations; ModName from the `.esl`; MCM in a separate mod |
| `Be Seated` / (no page) / toggle `Enable / Disable Sitting Prompts` | **compiled** | `.pex` extraction; page-less MCM; ModName from plugin |
| `Sofia Follower` / `Sofia Settings` / slider `Sofia Health: ` (trailing space), `Sofia Stamina: `, `Sofia Magicka: `; toggle `Disable Comments Dialogue` | **compiled** | `.pex` extraction, exact labels |

Plus golden checks without recordings:

- Physical Collision VR's 12 pages, in order (MCM Helper `pages[]`)
- Prisma UI AddItem Menu's keymap `Toggle Hotkey` with default `59`
- Nether's Follower Framework, a large compiled MCM: extraction completes without errors and finds a plausible number of options
- the `[NoDelete] Davey's MCM Recordings` folder is read correctly despite its brackets and apostrophe
- **from an archive:** Extended Encounters' menu appears, read from the compiled scripts in its `.bsa`, and the recorded steps for it resolve to exact options. This needs a snapshot taken once snapshots carry archives.
- **source vs. compiled:** for every script in a snapshot that ships both `.psc` and `.pex`, the two front ends produce the same options. This is the strongest test available, and it gets stronger with every list added.

`make test` runs `go test ./internal/...` then `svelte-check`, as in Hermaeus.

---

## 8. Phases

Each phase ends with everything working, tested and committed.

1. **Skeleton.** New repo with Hermaeus's infrastructure copied and renamed: server, token, prefs, gamecheck, textfile, Makefile, and the frontend shell of §5.3 with everything §5.2 lists as copied (theme, fonts, hint strip, theme switch, Skyrim's state, copy pipeline), showing the first-run form of §5.4. `PRODUCT.md`, `DESIGN.md` and `.impeccable/` per §6. `make build`, `make test` and `make windows` all pass, and the shell has been looked at in both themes (§5.6).
2. **Mod list, virtual files, snapshots.**
   - `internal/modlist`: MO2 instance detection, profile, priority order, `plugins.txt`, `overwrite`, game `Data` (from `gamePath`), the case-insensitive virtual index, and the Vortex/plain `Data` mode.
   - `lexicon scan` prints mods and winning-file counts. Tests cover priority, disabled mods, and bracketed names.
   - **`lexicon snapshot`**, ported from `New-MCM-Lexicon-Snapshot.ps1` (§10). At first it uses the script's text heuristic for finding MCM scripts; Phase 5 switches it to following `Extends` chains.
3. **MCM Helper and translations.** `internal/translations` and `internal/mcmhelper`, the catalog model, and the Menus, menu and Search pages composed as §5.4 says, with Quick Find, sort heads, the pane, and their hints, minus adding to a recording (its A hint isn't shown until Phase 9, since a hint needs a working shortcut). The MCM Helper golden checks pass. `lexicon scan --json` writes a catalog file. *This alone is a useful release.*
4. **Plugins.** `internal/plugin` VMAD reading (compressed records included).
5. **Papyrus source.** Tokenizer, parser, IR, and the shared extraction. The Dirt & Blood, EVG and Experience golden cases pass. Run over every snapshot and log what's `pattern`/`unknown` and why.
6. **Compiled Papyrus.** `.pex` reader and lowering. The Be Seated and Sofia golden cases pass, NFF completes, and the source-vs-compiled check passes.
7. **Recordings, read and validate; comparing lists.**
   - The Recordings page and the editor read-only (the Steps and Files tabs, validation words and sentences, copper on values that differ from a known default), the open-recording select in the top bar, and the recordings group in the sections column. The generic recording checks pass on every snapshot.
   - Opening a catalog file read-only, `lexicon diff`, and the Compare page with its stat-list pane. Test the diff with the MGO RC 4.5 and Nordic snapshots, and with two catalogs of the same list where one mod's options were edited by hand.
8. **Archives.** `internal/bsa`, per §2.6: confirm the format against the author's archives first, then read both versions, add loaded archives' files to the virtual index beneath the loose files, and read MCMs whose scripts are packed. Snapshots copy the archives that hold MCM files, and the loose sources of the scripts inside them. The archive tests and the Extended Encounters golden check pass, and Extended Encounters, R.A.S.S., Wyrmstooth, and `$SS_ModName` appear in the catalog or are listed as skipped with a reason.
9. **Recordings, write.** Drafts on the server; adding from a menu or search result (`?mode=add`, and the choose-a-recording dialog); editing a step in the pane (`?mode=edit`); the value editors and the DirectX scan-code key table; moving and removing steps; the Files tab's diff against disk; Save with backups and atomic writes; New and Delete. Round-trip tests.
10. **Polish and release.** Background scan with the bar's scan state, scan cache, the mark and icons (§6, §9), every page checked per §5.6 with a surface brief written for each, `README.md` in Hermaeus's voice, notices, a release tag.

Later, if wanted:

- writing MCM Helper settings directly to `MCM/Settings/<id>.ini`, which applies without MCM Recorder
- NL_MCM framework support
- "click N times" for steppers
- trimming plugins in snapshots to just their header and quest records with script data, which would make snapshots far smaller and copy much less of other authors' work
- a site under `site/`, on Hermaeus's Astro setup and the site's own `DESIGN.md` (the same world, larger and bolder), once there's a release to point to

---

## 9. Author tasks (before or during development)

These need the game, the author's Windows machine, or the author's decision, so Claude Code can't do them alone:

1. **Copy the snapshots to the Mac** and give Claude Code the folder (`URAG_SNAPSHOTS`). After a list updates, or to cover another list, run `New-MCM-Lexicon-Snapshot.ps1` on Windows (§10) and copy the new zip over. Once Phase 2 is done, `lexicon snapshot` replaces the script and runs on any OS.
2. **Record one MCM Helper menu in game** (for example two toggles and a slider in Physical Collision VR, and the keymap in Prisma UI AddItem Menu), save it in an MO2 mod, and take a new snapshot. That confirms what MCM Helper menus record as `mod` and `page` (display name vs. `modName`, page display names), which §2.2's mapping assumes.
3. **Once Phase 9 works, replay one written recording in game,** ideally one with a non-ASCII label and one with an `index`, to confirm the file format end to end.
4. **Done: snapshots carry archives.** All four were retaken with Phase 8's build, Synergy again after the fix for archives whose plugin is in another mod (§10); its Sofia Follower menu now reads from the SkyUI replacer's `SofiaFollower.bsa`. Retake a snapshot whenever its list updates.
5. Decide the name (MCM Lexicon is a placeholder) and whether the save target should default to a new mod folder or the author's existing `[NoDelete] Davey's MCM Recordings`.
6. **Choose the mark.** Hermaeus's is the eye of Hermaeus Mora among his tentacles; MCM Lexicon needs its own subject drawn the same way (§6): a single shape that survives 16 pixels, with one cut-out. The librarian's book or a card from a catalog are the obvious subjects; Claude Code can draft candidates as SVG for the author to choose from or redraw, and the shell ships with a plain wordmark until one is chosen.

---

## 10. Snapshots, catalogs, and comparing lists

Three kinds of per-list data, from heaviest to lightest:

**Snapshots** hold the files MCM Lexicon reads, for developing and testing MCM Lexicon. A snapshot is a zip containing:

- `mo2/`, a trimmed MO2 instance that MCM Lexicon opens like any other. For every enabled mod with MCM content, it has the plugins at the mod's top level, `MCM/`, `McmRecorder/`, translation files for the chosen languages, the MCM scripts (`.pex`, plus `.psc` wherever the mod keeps it), and, from Phase 8, the archives that hold any of those (§2.6), copied whole, with the plugin each is named after even when another mod holds it. It also has `overwrite/MCM/Settings` and `overwrite/McmRecorder`, and trimmed `ModOrganizer.ini`, `modlist.txt` (real priority order) and `plugins.txt` (real active marks).
- `SNAPSHOT.md`, for people and for Claude Code: where and when it was made, the mods included with their kinds (`mcmhelper:N`, `source:N`, `compiled:N`, `recordings`), every recorded step as an acceptance case, and known gaps.

A whole list is small. MGO RC 4.5 is 72 of 1,819 enabled mods and 277 files: 17.5 MB, or 2.7 MB zipped. Nordic Adventures is 52 mods and 224 files: 1.6 MB zipped. Snapshots hold other authors' files, so they're for local use only.

**Today:** `New-MCM-Lexicon-Snapshot.ps1` (Windows PowerShell 5.1, comes with this plan) makes them:

```powershell
.\New-MCM-Lexicon-Snapshot.ps1 -ModList F:\modlists\mgo-4-rc5
.\New-MCM-Lexicon-Snapshot.ps1 -ModList F:\modlists\nordic -Profile 'Nordic Adventures - High Fantasy Skyrim' -Languages ENGLISH, GERMAN -Out D:\snapshots
```

It finds MCM scripts by the text `SKI_ConfigBase`/`MCM_ConfigBase` in compiled scripts, which misses scripts that extend another MCM script without naming either. **In MCM Lexicon:** `lexicon snapshot -dir <MO2 instance> [-profile …] [-languages …] -o <zip>` does the same in Go on any OS, using the real `Extends` resolution once it exists. It also adds `snapshot.json`, a machine-readable inventory (list info, mods, files with owning mod and SHA-256, recordings), which the generic acceptance tests read.

**Catalogs** are what MCM Lexicon knows about a list's MCMs: the `Catalog` type (§3.2) as JSON, written by `lexicon scan --json -o catalog.json` or downloaded from the app. Catalogs contain labels and structure but no mod files, so they're small and fine to share (list curators could publish one with each release). MCM Lexicon opens a catalog file read-only to browse, search, and build or validate recordings on a machine that doesn't have the list.

**Comparing** two catalogs, as `lexicon diff old.json new.json` or the `/compare` page, reports:

- menus added and removed (matched by `Menu.ID`, then `Name`)
- pages and options added, removed, or probably renamed (same type and position, similar label)
- changed menu choices, slider ranges and defaults
- certainty changes
- for a given set of recordings, which steps break

This is the answer to "MGO updated; what do I need to redo?", and it's equally useful to someone maintaining a list. For the author's current lists, the snapshots that come with this plan are:

| Snapshot | Mods | Files | Recorded steps |
|---|---|---|---|
| `lexicon-snapshot-mgo-4-rc4.5-…zip` | 72 | 277 | 57 |
| `lexicon-snapshot-nordic-…zip` | 52 | 224 | 23 |
