---
version: 1
slug: "site-src-pages-index-astro"
primary_target: "site/src/pages/index.astro"
related_targets: ["site/src/components/MenuDemo.astro","site/src/layouts/Docs.astro"]
---

# Landing page (site/src/pages/index.astro)

Mode: Persuade. Code-led (no image generation in this harness). The docs under site/src/pages/docs are a Read surface framed by the same world.

Audience: Skyrim VR players and mod list authors who use MCM Recorder. Job: understand that MCM Lexicon writes MCM Recorder recordings without launching Skyrim, trust it with their list, and download the Windows build. Proof: WeaponThrowVR's real menu (shown with Shizof's permission), real screens from the Synergy list, and the exact file it writes. No invented claims.

## Direction contract

THESIS: The menu is the pitch. The first viewport is a working slice of WeaponThrowVR's real menu, drawn as MCM Lexicon draws it, and every value the visitor sets writes the exact step MCM Recorder reads, live, beside it. It refuses the category's headline, screenshot, and feature-card hero.

OWN-WORLD: Hermaeus's site world, inherited and expanded (the user released the site from treating it as sacred on 2026-11-05; it has a larger audience): smoked-glass ground, 1px hairlines in two strengths, verdigris (dark) and frost blue (light), Barlow Condensed tracked caps to label, Jost to read, 3px corners, the diamond on the selected thing, and the bottom hint strip of real shortcuts. Added from the app: DM Mono for the step file and selectors, the File Markup colors (keys in the accent, values in ink, punctuation muted), copper on a value that differs from the mod's default, and an amber diamond on a step not yet saved. New to the site: the app's shell as a component (top bar, menu screen, pane, hint strip) placed in the page as a working window; a status lamp (RECORDING, amber, Save's breathing glow, a count that lands with Save's rise); and a contextual hint strip, the page's stepping aside for the window's.

STORY: The visitor recognizes a SkyUI menu, sets a value, and watches the step file change; they understand nobody had to launch the game. Then why it exists, real screens, what else it does, the safety rules, and the download.

FIRST VIEWPORT: Across the top, the offer: MCM LEXICON at display size with the hook "Create and edit MCM Recorder recordings, with Skyrim closed." under it at left; at right, behind a strong hairline (26 to 28rem), one line of support, the primary Download for Windows button, and the fine print. Under it, full width, MCM Lexicon's own window, running, so it reads as a program and not another screenshot: the app's top bar (the mark and name; LIST synergy-rc37, PROFILE Synergy VR - ENB, SKYRIM CLOSED; and a RECORDING lamp in Save's amber, breathing its glow, counting the draft's steps), then the menu at left, opened on General with "Allow Greanade and Other Throw" selected and in view (WEAPONTHROWVR, the credit to Shizof in its header, its 13 pages down the left, two option columns on the game's grid scrolling inside, SkyUI's info area with help and selector at its foot), and the step file at right (19 to 25rem, Pane Glass, DM Mono, its path above), as tall as the menu and scrolling inside it, and along the window's bottom the app's hint strip: "Go ahead, flip something." at left and, at right, the keys this screen offers (↑↓ MOVE, then SPACE ON/OFF, SPACE CLICK, or ENTER APPLY by the selected option's kind, and R START OVER), every one a real key and a button. The site's own strip steps aside while the window is on screen and returns once it's scrolled past. The offer's support line points at the window. The file's foot says every label is spelled as the mod spells it and nothing here touches your files. Revised during the build from "file docked under the menu, offer at right": that stack didn't fit a 900px-tall window, and side by side makes "set here, see there" one glance.

FORM: Hermaeus's site structure, with the diary entry replaced by a working menu (the user chose "a live menu, writing JSON" over a static file and a screenshot). seed:e89249373b33dce4 (rolled for the record; the user pinned the structure, which beats the roll, so its hand was not presented)

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
