---
title: Making a recording
description: How to create or change an MCM Recorder recording, check it, and save it.
order: 2
---

A recording is what MCM Recorder plays when you start a new game: a list of steps, each one setting one option in one menu. MCM Lexicon lets you build that list by picking values in the menus, the way you would in the game.

## Turn on editing

At the top of the column, switch from **Read-only** to **Editing**. Then choose the recording to add to under **Adding to**, or pick **New recording…** at the bottom of that list and give it a name.

## Set some values

Open a menu and set options the way you would in SkyUI: switches for toggles, a number (or a slider) for sliders, a list for menus, and a click for text options. Each value you set becomes a step in the recording.

A small amber diamond before a value means it's in the recording but not saved yet. A value in copper is different from the mod's own default, where MCM Lexicon knows the default.

**Record all settings on this page** adds a step for every option on the page at once, which is handy when you want a whole page to match what you see.

<div class="btw">

Why the exact labels matter

MCM Recorder finds each option by its label, and the match has to be exact, trailing spaces and typos included. One of WeaponThrowVR's toggles is literally "Allow Greanade and Other Throw". MCM Lexicon reads labels from the mods' own files, so you never have to type one.

</div>

## Check it

A recording's page lists its steps, grouped by step file, and checks each one against what's installed. A step that won't replay says why:

| Flag | What it means |
| --- | --- |
| NOT FOUND | The menu has no page or option by that name (anymore), so the step finds nothing. |
| NOT ALLOWED | The option is there, but the value is outside its range or isn't one of its entries. |
| MOD MISSING | There's no menu by that name in your list, so MCM Recorder skips the step. |
| GRAYED OUT | The menu grays the option out. MCM Recorder plays the step anyway. |
| DOES NOTHING | The option is there, but the menu doesn't respond to a click on it. |
| UNCHECKED | MCM Lexicon doesn't know all of that menu's options, so a miss proves nothing. |

A step that matches says nothing at all. The **Files** tab shows each file exactly as it will be written.

## Save it

Nothing is written until you save. **Review** (or the Changes page) shows every change as it will be written, compared with what's on disk now. Then click **Save**.

If you changed a recording that came with another mod, the change saves as a copy, in a mod folder of its own. The original stays untouched. See [Where things go](../where-things-go/).

**MO2 users:** the first time you save, MO2 lists the recordings folder as a new mod, disabled. Enable it, or MCM Recorder won't see your recording in the game.
