# Changelog

What changed in each release of MCM Lexicon, newest first.

## 0.1.0-beta.4 (2026-11-09)

MCM Lexicon can now read your menus from inside the running game with an optional SKSE plugin. This fills in the values and labels that can't be derived from just reading the mods' files. The other big change: every option can be changed straight away, without first hunting down the **editing** switch.

- Added the capture plugin, `MCMLexicon-Capture.zip`, an SKSE plugin for Skyrim SE, AE, and VR. Install it as a mod, press **Capture in game** in MCM Lexicon Settings, and load a save. The plugin works through every menu, one page at a time, and MCM Lexicon automatically picks up what it found. The plugin only runs the capture when you ask for it, adds nothing to your load order or saves, and can be removed at any time. See [Capturing from the game](https://daveystrus.com/mcm-lexicon/docs/capturing-from-the-game/).
- With a capture, the line under a menu shows **IN YOUR GAME**, and an option you haven't recorded yet starts from your game's value, after any value MCM Helper has saved. On Mad God's Overhaul, a capture matched all 83 menus and gave 4,144 options their value in the game.
- A capture also adds other information that couldn't be read from the files: pages MCM Lexicon couldn't find (one menu went from one page to 16), options a script adds once per item, and labels built while the game runs. Options loaded from a capture can be recorded like any other.
- Menus that only track your progress, like the LOTD Checklist, are skipped by a capture unless you turn on **Include trackers**. (But don't turn it on.) You can also exclude capture data from what the Lexicon displays, or read a capture file from another location.
- Every option now looks editable from the jump and can be changed at any time. WHen in read-only mode, the first change prompts you to start editing, and if no recording is chosen, it asks which recording to use.
- **Adding to** shows **None** when the recording it remembered isn't in the list anymore, instead of showing nothing.
- `MCMLexicon.exe scan -json -capture capture.json` adds a capture to the catalog it writes and prints how much matched.

## 0.1.0-beta.3 (2026-11-08)

Menus now show the default values for far more options across far more menus. On an internal build of Mad God's Overhaul, the options showing no default value went from about two-thirds to under a quarter.

- Added default values for new games. If an option isn't recorded and MCM Helper has no saved value for it, the menu shows what the mod's scripts and plugins say a new game starts with, and the info line labels it **NEW GAME**. That covers global variables, values a script sets up for itself, other scripts' properties, MCM Helper settings that classic menus read, and the INI files that certain mods keep their settings in. Several big mods (including VRIK and Spell Wheel VR, among others), went from missing most of their defaults to having most or all of them filled in.
- Fixed some bogus values derived from bad searches in compiled script.
- Tabbing past a field no longer records it. Leaving a number or text field records the option only when you've changed it or explicitly saved.
- Options added once per item (one per recording in MCM Recorder's menu, say) are marked as repeating.
- Snapshots (Settings → Make snapshot) now hold everything these values come from: SKSE plugins' INI files, the scripts other scripts' properties point to, and patches that change an MCM's quest.

## 0.1.0-beta.2 (2026-11-08)

MCM Lexicon now opens in a window of its own instead of your browser, and setting a value while editing no longer takes an extra step.

- `MCMLexicon.exe` shows MCM Lexicon in a window (Microsoft Edge WebView2, part of Windows 10 and 11), with no console window. Closing the window stops MCM Lexicon, after asking first if a recording has changes that aren't saved. Starting it again brings the open window to the front. On a PC without WebView2, it opens in the browser as before. The command-line modes still work; in PowerShell, add `| more` to see their output.
- While editing, a menu entry, slider, hotkey, color, or text value is set right in its row, and setting it adds the option to the recording, with no step to add it first. With no recording chosen, the first change asks which recording it goes to.
- A hotkey can be set to a controller button, including VR controllers. In Skyrim VR, the buttons are listed by controller, Quest-style ones with pictures. Which button writes which code is provisional until it's confirmed in the headset.
- You can save the list's catalog to a file from Settings, which Compare can compare the list against later, and makes a snapshot of the list for testing, so neither needs the command line. Compare has a **Browse…** button for the older catalog file.

## 0.1.0-beta.1 (2026-11-05)

The first public build: a catalog of every MCM option in a Skyrim mod list, read  at runtime with the game closed, and an editor for MCM Recorder recordings that checks them against what's installed and writes exactly what MCM Recorder reads. Built primarily for Skyrim VR players and list authors, and works with any Skyrim SE or AE setup managed with Mod Organizer 2, or a plain `Data` folder (including Vortex setups).
