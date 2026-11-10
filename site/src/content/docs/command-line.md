---
title: Command line
description: Switches for running MCM Lexicon, and the commands that work with catalogs from a terminal.
order: 5
---

You never have to use any of this. Running `MCMLexicon.exe` is enough, and Settings can save a catalog file or make a snapshot too. But if you like a terminal, there are fancy switches.

```
MCMLexicon.exe [-list <MO2 instance or Data folder>] [-profile <MO2 profile>] [-port 8766] [-no-browser] [-version]
```

| Switch | What it does |
| --- | --- |
| `-list` | The MO2 instance or `Data` folder to open, instead of finding one. |
| `-profile` | The MO2 profile to read, instead of the one selected in MO2. |
| `-port` | The port to serve the page on: 8766 unless you say otherwise. If it's taken, the next nine are tried, then any free port. |
| `-no-browser` | Don't open a window. The address is printed, for a browser of your choosing. |
| `-version` | Print the version and stop. |

## Commands

These work with catalogs from a terminal, with no window at all. (They exist because I build MCM Lexicon on a Mac where Skyrim isn't installed at all.)

Print the relevant mods in your list, with the files MCM Lexicon reads from each:

```
MCMLexicon.exe scan [-list …] [-all]
```

Write the catalog to a file, which the app can open on any machine, including one without your list (or even Skyrim):

```
MCMLexicon.exe scan -json -o catalog.json [-list …]
```

Add a [capture](../capturing-from-the-game/) from your game to that catalog, and print how much of it matched:

```
MCMLexicon.exe scan -json -capture capture.json -o catalog.json [-list …]
```

Show the differences between two catalogs, such as one version of a list and the next:

```
MCMLexicon.exe diff old.json new.json
```

Copy the part of your list MCM Lexicon actually reads, for testing on a machine that doesn't have the whole list:

```
MCMLexicon.exe snapshot [-list …] [-o out.zip]
```

<div class="btw">

A word about snapshots

A snapshot copies other authors' files out of your list. It's for testing on your own machines, not for sharing.

</div>
