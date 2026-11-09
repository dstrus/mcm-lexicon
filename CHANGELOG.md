# Changelog

What changed in each release of MCM Lexicon, newest first.

## 0.1.0-beta.2 (2026-11-08)

MCM Lexicon now opens in a window of its own instead of your browser, and setting a value while editing no longer takes an extra step.

- `MCMLexicon.exe` shows MCM Lexicon in a window (Microsoft Edge WebView2, part of Windows 10 and 11), with no console window. Closing the window stops MCM Lexicon, after asking first if a recording has changes that aren't saved. Starting it again brings the open window to the front. On a PC without WebView2, it opens in the browser as before. The command-line modes still work; in PowerShell, add `| more` to see their output.
- While editing, a menu entry, slider, hotkey, color, or text value is set right in its row, and setting it adds the option to the recording, with no step to add it first. With no recording chosen, the first change asks which recording it goes to.
- A hotkey can be set to a controller button, including VR controllers. In Skyrim VR, the buttons are listed by controller, Quest-style ones with pictures. Which button writes which code is provisional until it's confirmed in the headset.
- You can save the list's catalog to a file from Settings, which Compare can compare the list against later, and makes a snapshot of the list for testing, so neither needs the command line. Compare has a **Browse…** button for the older catalog file.

## 0.1.0-beta.1 (2026-11-05)

The first public build: a catalog of every MCM option in a Skyrim mod list, read  at runtime with the game closed, and an editor for MCM Recorder recordings that checks them against what's installed and writes exactly what MCM Recorder reads. Built primarily for Skyrim VR players and list authors, and works with any Skyrim SE or AE setup managed with Mod Organizer 2, or a plain `Data` folder (including Vortex setups).
