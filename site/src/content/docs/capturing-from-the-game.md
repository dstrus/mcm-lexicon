---
title: Capturing from the game
description: The optional capture plugin, which reads every menu in your running game to fill in what can't be discovered otherwise.
order: 3
---

MCM Lexicon reads everything it can from your mods' files, without running the game. That gets us most of the way there, but it does miss some things. Some mods keep their values inside a DLL, some build their labels while the game runs (a follower's name, for example), and a few menus only reveal their pages in-game.

The capture plugin fills those gaps. It's a small SKSE plugin that, when you ask it to, opens every menu in your running game (but not where you can see it) and writes down what each page shows. MCM Lexicon then adds that to its catalog. You never strictly _need_ it. Everything else works without it. But if you want your lexicon to be as close to comprehensive as it can possibly be, this is the way.

## Installing it

[Download the capture plugin](https://github.com/dstrus/mcm-lexicon/releases/latest/download/MCMLexicon-Capture.zip) and install the zip as a mod, the same way you'd install any other. (In MO2: **Install a new mod from an archive**, or just drag the .zip file from File Explorer to MO2's left pane.) It works with Skyrim SE, AE, and VR, and needs SKSE and Address Library (or VR Address Library), which nearly every list already has.

It has no Skyrim plugin, so nothing new shows up in your load order, and it never adds anything to your saves. You can remove it whenever you like, mid-playthrough or not.

## Taking a capture

**If you want to capture the values for a new game:**

1. Launch Skyrim, start a new game, finish character creation, and wait a minute or two for all of the menus to be registered.
2. Save the game.
3. In MCM Lexicon, go to **Settings** and press **Capture in game**.
4. Back in the game, load the save you just made.

_Note that MCM Helper menus always show your saved settings on new games, so unless this is a truly fresh install, some values may not be defaults._

**If you want to capture the current values in a particular playthrough:**

1. In MCM Lexicon, go to **Settings** and press **Capture in game**.
2. Launch Skyrim and load a save (not a new game).

**Either way, this comes next:**

- Stay out of the Journal for a minute or two. About 15 to 30 seconds after the save loads, the plugin starts working through your menus, one page at a time. A large list takes a couple of minutes (MGO's 85 menus take about 100 seconds).

That's it. MCM Lexicon notices the new capture on its own, and Settings shows when it was taken and how much of it matched the menu catalog it already has.

The plugin only captures when you ask. Each request is good for one capture, and it's used up as soon as the capture starts, so even if something went wrong, the next load wouldn't try again.

<div class="btw">

By the way…

Opening your Journal while a capture is running stops it, since the plugin's work would otherwise show up in the real menu. Whatever it had captured by then is kept.

</div>

## What you get

Once there's a capture, MCM Lexicon shows each option's value in your game next to what was pulled from the mod's files. Here, a new game starts Road Encounter Chance at 50% (the mod's default), but it's 25% in the save that was captured:

<figure class="shot themed" style="--w: 1280px">
  <img class="dark" src="/mcm-lexicon/docs/shots/in-your-game-dark.webp" alt="Extended Encounters' World Encounters page with a capture added, its parts numbered." loading="lazy" decoding="async" />
  <img class="light" src="/mcm-lexicon/docs/shots/in-your-game-light.webp" alt="Extended Encounters' World Encounters page with a capture added, its parts numbered." loading="lazy" decoding="async" />
  <figcaption><ol><li>Road Encounter Chance holds your game's value, 25, so that's where a new step starts.</li><li>IN YOUR GAME, next to DEFAULT (the value from the mod's files).</li></ol></figcaption>
</figure>

If you've changed an option in an MCM Helper menu, the value MCM Helper saved still comes first. See [Where the values come from](../making-a-recording/#where-the-values-come-from).

A capture also adds other information that wasn't readable from the mods' files directly: pages it couldn't find, options a script adds in a loop (_e.g._ an option for each active follower), and labels that only exist at runtime. You can record those options like anything else, since their labels match the game.

To see only what came from reading the files again, turn off **Show the capture's values** in Settings.

## Things to know

- **A capture is one game's state.** It shows your values in the save you loaded, not a new game's, and only the options present in that save. An option a setting hides won't be in it.
- **Some menus are skipped.** A few menus only track your progress, with no settings to record, and they are very slow to capture. The LOTD Checklist is the one that has been accounted for so far. Turn on **Include trackers** in Settings if you want them captured anyway.
- **Capturing runs each menu's actual code,** exactly as opening each page in the Journal would. A menu that does something when it's opened or closed (exporting its settings to a file, for example) does that during a capture too.
- **Captures are per game, not per list.** If you play two lists of the same game (_e.g._ two Skyrim SE lists, or two Skyrim VR lists), the newest capture is whichever one you took last. Settings shows how much matched its existing catalog, which makes a capture from the wrong list easy to spot. You can also choose a capture file in Settings, which is how you'd use one from another machine.

The plugin is open source like the rest of MCM Lexicon. Because it's built on CommonLibSSE NG, the plugin itself is licensed under the GPL. Its license and notices come in the zip.
