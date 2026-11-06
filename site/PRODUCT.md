# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Astro, building static HTML for GitHub Pages, as Hermaeus's site is built. User documentation is written as Markdown in a content collection; interactive parts are plain TypeScript in Astro components. The site lives in `site/` of the MCM Lexicon repository, with its own `package.json`, apart from the app's build: the Go program embeds `web/build` only, and no Go files go under `site/`.

Served at `https://daveystrus.com/mcm-lexicon`. Astro's `site` is `https://daveystrus.com` and its `base` is `/mcm-lexicon`, so every link and asset path carries the base.

## Users

Primary: Skyrim VR players and mod list authors who use MCM Recorder. VR players feel the cost of launching the game most (a headset, minutes of loading, menus that are awkward in VR), and list authors maintain recordings across releases of a list.

Secondary: anyone using MCM Recorder on Skyrim SE or AE with Mod Organizer 2, Vortex, or a plain `Data` folder; and people who already run MCM Lexicon and need the manual.

## Product Purpose

The public web site for MCM Lexicon: a landing page that lets an MCM Recorder user understand what MCM Lexicon does and decide to download it, and user documentation. Success means a visitor knows within a minute that MCM Lexicon writes MCM Recorder recordings without launching Skyrim, that it's unofficial and careful with their files, and where to get it; and an installed user finds how to do what they came for.

MCM Lexicon itself reads a mod list from disk, with the game closed, catalogs every Mod Configuration Menu in it, and creates and edits MCM Recorder recordings. The app's own product record is the repository's root `PRODUCT.md`.

## Positioning

MCM Recorder records clicks in game and replays them. Writing or fixing a recording by hand means knowing every label exactly, and no file lists them. MCM Lexicon is that list, read from the mods' own files, and an editor that writes what MCM Recorder reads. If you're happy making recordings in game, MCM Recorder alone is the right tool, and the site says so.

## Operating Context

- Downloads are builds attached to GitHub Releases at `https://github.com/dstrus/mcm-lexicon`. Releases are Windows only, one file whose name carries no version: `releases/latest/download/MCMLexicon-windows-x64.zip`. macOS and Linux users can build it themselves (`DEVELOPING.md`).
- No Nexus Mods page exists; until one does, the site doesn't mention or link one.

## Capabilities and Constraints

- Static hosting only: no server code, no accounts, no forms that need a back end.
- The site describes only what the app actually does, in its terms (menu, page, option, selector, recording, step, step file, catalog, list, profile).
- The site never describes MCM Lexicon as a rewrite, port, or successor of anything.
- Screens and the hero's menu show other authors' menus and labels. They come from the Synergy list. WeaponThrowVR, by Shizof, is shown with the author's express permission, and the site credits it. Snapshots of a list are never committed or shared.
- MCM Lexicon is released under the MIT License.

## Brand Commitments

- Name: MCM Lexicon. The mark (the face of a Dwemer lexicon with the game's diamond at its dial's center) is in the repository's `assets/`.
- Voice: Davey's own, as in the README and the MGO tutorial site: first person, conversational, honest about limits (including that many people should just use MCM Recorder). Dry humor in asides, parentheticals, and leads, never in download steps or safety rules, which stay plain and exact. No hype.
- The unofficial-status and trademark statements stay as the README words them: MCM Lexicon works with MCM Recorder but isn't part of it, isn't affiliated with MCM Recorder, SkyUI, MCM Helper, or Mod Organizer, and includes none of their files; The Elder Scrolls V: Skyrim is a trademark of ZeniMax Media Inc.
- A sibling of Hermaeus's site (`../hermaeus/site`): the same world, set larger than the app.
- American English spelling and the serial comma.

## Evidence on Hand

- The app runs against the Synergy list's snapshot, so real screens can be taken.
- WeaponThrowVR's real menu, read by MCM Lexicon, drives the hero.
- No testimonials, users, download counts, reviews, or press; don't invent any.

## Product Principles

1. **Show the real thing.** The real menu, the real labels, and the exact file MCM Lexicon writes beat any claim about it.
2. **Earn trust on safety.** Say plainly what it writes, where, and what it refuses to do, before asking anyone to point it at their list.
3. **Unofficial, and says so.**
4. **Docs answer the real setup.** Mod Organizer 2 first, then Vortex and plain `Data` folders, on Windows.

## Accessibility & Inclusion

WCAG AA in both themes, visible focus, complete keyboard paths (the hero's menu included), and documentation that reads well with a screen reader.
