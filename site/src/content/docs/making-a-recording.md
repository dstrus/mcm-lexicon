---
title: Making a recording
description: How to create or change an MCM Recorder recording, check it, and save it.
order: 2
---

A recording is what MCM Recorder plays when you start a new game (if autorun is on) or when you choose to play it. It's a list of steps, each one setting a single option in one menu. MCM Lexicon lets you build that list by picking values in the menus, the way you would in the game.

## Turn on editing

Near the top of the column, switch from **Read-only** to **Editing**. Then choose which recording to add to under **Adding to**, or pick **New recording…** at the bottom of that list and give it a name. (You can also leave it at **None** and just start changing things. The first change asks which recording it should go to.)

<figure class="shot themed" style="--w: 300px">
  <img class="dark" src="/mcm-lexicon/docs/shots/editing-controls-dark.webp" alt="The top of the column while editing, its parts numbered." loading="lazy" decoding="async" />
  <img class="light" src="/mcm-lexicon/docs/shots/editing-controls-light.webp" alt="The top of the column while editing, its parts numbered." loading="lazy" decoding="async" />
  <figcaption><ol><li>Read-only or Editing.</li><li>The recording you're adding to, with a button to open its page.</li><li>Once something has changed, Save and Review appear here.</li></ol></figcaption>
</figure>

## Set some values

Open a menu and set options the way you would in SkyUI. While you're editing, every option shows an appropriate control: a toggle switch, a slider, a dropdown, etc. You can also press `E` to edit the selected option from the keyboard. In either case, changing one records it in your draft recording.

A value in copper is already saved in the recording and differs from the mod's default (when MCM Lexicon knows the default). A small amber diamond before a value means it's in the draft recording but not saved to disk yet.

<figure class="shot themed" style="--w: 1280px">
  <img class="dark" src="/mcm-lexicon/docs/shots/menu-editing-dark.webp" alt="WeaponThrowVR's Impact page while editing, its parts numbered." loading="lazy" decoding="async" />
  <img class="light" src="/mcm-lexicon/docs/shots/menu-editing-light.webp" alt="WeaponThrowVR's Impact page while editing, its parts numbered." loading="lazy" decoding="async" />
  <figcaption><ol><li>The amber diamond: this step is in the recording, but not saved yet.</li><li>The value, in copper because it differs from the mod's default of 2.0.</li><li>The menu's pages.</li><li>The selected option's help, its exact label, what it allows, its default, and which step of the recording it is.</li></ol></figcaption>
</figure>

### Where the values come from

Before you record an option, its control shows the value it already has, in muted text, so you can see what you'd be changing. MCM Lexicon takes that value from the first of these it knows:

1. **Saved:** what MCM Helper has saved for the option, if it's an MCM Helper menu and you've changed it in the game.
2. **New game:** what the mod's scripts and plugins say a new game starts with. That's read from the mod's global variables, the values its scripts set up for themselves, other scripts' settings that it reads, MCM Helper settings, and the INI files that some SKSE plugins keep their settings in (VRIK, Experience, Ricochet, Accuracy, and Core Impact Framework so far).
3. **Default:** the default the menu itself states, which is what it resets to.

The line under the menu names each value it knows, as DEFAULT, NEW GAME, and SAVED. NEW GAME only appears when it says something DEFAULT doesn't. An option shows a dash when none of them is known, usually because its value depends on something only the running game knows (what's in your inventory, say), or because a plugin keeps it out of reach inside its DLL.

<div class="btw">

Why a new game, and not my game?

MCM Lexicon reads your mods' files, not your save games. If you've changed a setting in your current playthrough, that change often lives in your save, where MCM Lexicon can't see it. For a recording, that's likely what you want anyway, since recordings mostly play on a new game.

</div>

### Hotkeys

To record a hotkey, click into the key box and press the key you want, a mouse button (other than the left one), or a button on a controller that's plugged into your PC. In Skyrim VR, your motion controllers show up to menus as a regular gamepad, and those mappings may or may not make sense. So a list beside the box names each VR controller button by its controller and shows the gamepad button it gets translated to.

<figure class="shot themed" style="--w: 400px">
  <img class="dark" src="/mcm-lexicon/docs/shots/controller-buttons-dark.webp" alt="The controller button list open beside a hotkey, listing Quest-style controllers' buttons with pictures, then Vive wands." loading="lazy" decoding="async" />
  <img class="light" src="/mcm-lexicon/docs/shots/controller-buttons-light.webp" alt="The controller button list open beside a hotkey, listing Quest-style controllers' buttons with pictures, then Vive wands." loading="lazy" decoding="async" />
  <figcaption>Choosing a controller button for a hotkey. The names are provisional until I've confirmed each one in the headset.</figcaption>
</figure>

**Record all settings on this page** adds a step for every option on the page at once, which is handy when you want to be sure that a whole page of options will match what you see in MCM Lexicon.

<div class="btw">

Why the exact labels matter

MCM Recorder finds each option by its label, and the match has to be exact, trailing spaces and typos included. MCM Lexicon reads labels directly from the mods themselves, so you never have to type one.

</div>

## Check your list before you wreck your list

A recording's page lists its steps, grouped by file, and checks each one against what's actually installed. A step that raises flags says why:

| Flag | What it means |
| --- | --- |
| NOT FOUND | The menu has no page or option by that name (anymore), so the step finds nothing. |
| NOT ALLOWED | The option is there, but the value is outside its range or isn't one of its entries. |
| MOD MISSING | There's no menu by that name in your list, so MCM Recorder skips the step. |
| GRAYED OUT | The menu grays the option out. MCM Recorder plays the step anyway (I checked!), so be aware. |
| DOES NOTHING | The option is there, but the menu doesn't respond to a click on it. |
| UNCHECKED | MCM Lexicon doesn't know all of that menu's options, so this could be a false alarm. |

A step that matches what's installed says nothing at all. The **Files** tab shows each file's changes as they will be written.

<figure class="shot themed" style="--w: 1280px">
  <img class="dark" src="/mcm-lexicon/docs/shots/steps-dark.webp" alt="A recording's page, listing its steps with one selected and its details beside it." loading="lazy" decoding="async" />
  <img class="light" src="/mcm-lexicon/docs/shots/steps-light.webp" alt="A recording's page, listing its steps with one selected and its details beside it." loading="lazy" decoding="async" />
  <figcaption>A recording's steps, grouped by step file. The selected step's details say what the catalog found for it.</figcaption>
</figure>

## Save it

Nothing is written until you save. **Review** (or the Changes page) shows every change as it will be written, compared with what's on disk now. Click **Save** when you're ready.

<figure class="shot themed" style="--w: 1280px">
  <img class="dark" src="/mcm-lexicon/docs/shots/changes-dark.webp" alt="The Changes page, showing a new recording's two files as they will be written." loading="lazy" decoding="async" />
  <img class="light" src="/mcm-lexicon/docs/shots/changes-light.webp" alt="The Changes page, showing a new recording's two files as they will be written." loading="lazy" decoding="async" />
  <figcaption>Before you save, every file as it will be written, next to what's on disk now (nothing, for a new recording).</figcaption>
</figure>

If you changed a recording that came with another mod, the change saves as a copy, in a separate mod folder. The original stays untouched. See [Where things go](../where-things-go/).

**MO2 users:** the first time you save, MO2 lists the recordings folder as a new mod, disabled. Enable it, or MCM Recorder won't see your recording in the game.
