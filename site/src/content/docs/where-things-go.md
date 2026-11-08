---
title: Where things go
description: Every place MCM Lexicon writes to, and what it writes there.
order: 3
---

MCM Lexicon reads your whole list, but it only writes to a few places. Here they all are.

## Recordings

With Mod Organizer 2, recordings save to a mod folder of their own in your MO2 instance, called `[NoDelete] MCM Lexicon Recordings` unless you change it in Settings. The `[NoDelete] ` prefix keeps a Wabbajack update from deleting it.

<figure class="shot themed" style="--w: 1280px">
  <img class="dark" src="/mcm-lexicon/docs/shots/settings-dark.webp" alt="The Settings page: the mod list in use, and the save folder's name and [NoDelete] prefix." loading="lazy" decoding="async" />
  <img class="light" src="/mcm-lexicon/docs/shots/settings-light.webp" alt="The Settings page: the mod list in use, and the save folder's name and [NoDelete] prefix." loading="lazy" decoding="async" />
  <figcaption>Settings, at the top of the column: change the list MCM Lexicon reads, and name the mod folder recordings save to.</figcaption>
</figure>

MO2 lists that folder as a new mod, disabled, the first time. Enable it so MCM Recorder actually plays your recordings in the game.

With a plain `Data` folder instead, recordings save there.

MCM Lexicon never changes a recording that came with another mod. Changing one saves a copy to that same separate folder, as a recording of its own.

## Backups

When MCM Lexicon replaces or removes a file, a copy goes to a `lexicon-backups` folder beside the list (next to `ModOrganizer.ini` or `Data`) first. A recording's **Backups** tab lists its backups and can restore them.

## Its own settings

Which list was open, the theme, and the save folder's name live in `MCM Lexicon/settings.json` under your user configuration folder. A scan cache goes under your user cache folder.

Nothing else is written anywhere.

## While Skyrim is running

MCM Lexicon won't write anything while the game is running. The top of the page says **Skyrim running · read-only**, and you can still browse all you want.
