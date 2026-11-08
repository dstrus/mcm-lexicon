---
name: MCM Lexicon (web site)
description: The MCM Lexicon web site, set as one more menu of the game you just closed, larger than the app, with a real menu working in its first viewport.
colors:
  accent: "#7fb8a0"
  on-accent: "#0e1a15"
  ground: "#0b0b0b"
  smoke: "#1c1c1c"
  panel: "#121212"
  select: "rgb(242 239 230 / 0.07)"
  field: "rgb(0 0 0 / 0.38)"
  ink: "#f2efe6"
  muted: "#a8a293"
  rule: "#2a2a2a"
  rule-strong: "#3a3a3a"
  changed: "#d2996b"
  warn: "#d9a441"
  invite: "#f0a3b5"
  on-invite: "#22070f"
  accent-light: "#2a5f86"
  on-accent-light: "#eef4f9"
  ground-light: "#e4e3df"
  smoke-light: "#f3f2ef"
  panel-light: "#eceae6"
  select-light: "rgb(27 26 23 / 0.06)"
  field-light: "rgb(255 255 255 / 0.55)"
  ink-light: "#1b1a17"
  muted-light: "#5d5a52"
  rule-light: "#cdcbc4"
  rule-strong-light: "#b3b0a7"
  changed-light: "#9a4a20"
  warn-light: "#8a5a00"
  invite-light: "#b0325a"
  on-invite-light: "#fff1f5"
typography:
  display:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "clamp(3.5rem, 2rem + 5vw, 6rem)"
    fontWeight: 600
    lineHeight: 0.9
    letterSpacing: "0.06em"
  headline:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "clamp(2.25rem, 1.5rem + 3vw, 3.75rem)"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "0.12em"
  title:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "1.375rem"
    fontWeight: 600
    letterSpacing: "0.12em"
  lead:
    fontFamily: "Jost, system-ui, sans-serif"
    fontSize: "1.375rem"
    fontWeight: 400
    lineHeight: 1.35
  body:
    fontFamily: "Jost, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.55
    fontFeature: "tnum"
  prose:
    fontFamily: "Jost, system-ui, sans-serif"
    fontSize: "1.0625rem"
    fontWeight: 400
    lineHeight: 1.65
  option:
    fontFamily: "Jost, system-ui, sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 400
    lineHeight: 1.3
  file:
    fontFamily: "DM Mono, ui-monospace, monospace"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.55
  button:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "1.125rem"
    fontWeight: 500
    lineHeight: 1.2
    letterSpacing: "0.12em"
  hint:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "1rem"
    fontWeight: 500
    lineHeight: 1.5
    letterSpacing: "0.12em"
  label:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 500
    letterSpacing: "0.12em"
rounded:
  base: "3px"
spacing:
  "1": "0.25rem"
  "2": "0.5rem"
  "3": "0.75rem"
  "4": "1rem"
  "5": "1.5rem"
  "6": "2rem"
  "7": "2.5rem"
  "8": "4rem"
  "9": "clamp(5rem, 3rem + 6vw, 9rem)"
  gutter: "clamp(1rem, 0.25rem + 3.5vw, 3.5rem)"
components:
  button:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    typography: "{typography.button}"
    rounded: "{rounded.base}"
    padding: "0.75rem 1.5rem"
  button-primary:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.on-accent}"
    typography: "{typography.button}"
    rounded: "{rounded.base}"
    padding: "0.75rem 1.5rem"
  theme-option:
    backgroundColor: "transparent"
    textColor: "{colors.muted}"
    typography: "{typography.label}"
    padding: "0.5rem 0.75rem"
  theme-option-selected:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.on-accent}"
  section-link:
    textColor: "{colors.muted}"
    typography: "{typography.button}"
  section-link-current:
    textColor: "{colors.ink}"
  hint-strip:
    backgroundColor: "{colors.ground}"
    textColor: "{colors.muted}"
    typography: "{typography.hint}"
    padding: "0.5rem clamp(1rem, 0.25rem + 3.5vw, 3.5rem)"
  hint-key:
    textColor: "{colors.ink}"
    rounded: "{rounded.base}"
    padding: "0 0.4em"
  demo-frame:
    rounded: "{rounded.base}"
    height: "clamp(15rem, 100svh - 36rem, 30rem)"
  menu-page:
    textColor: "{colors.muted}"
    typography: "{typography.hint}"
    rounded: "{rounded.base}"
  menu-page-current:
    textColor: "{colors.accent}"
  menu-row:
    textColor: "{colors.ink}"
    typography: "{typography.option}"
    padding: "0.2rem 0.5rem"
    height: "2.5rem"
  menu-row-selected:
    backgroundColor: "{colors.select}"
  menu-field:
    backgroundColor: "{colors.field}"
    textColor: "{colors.muted}"
    typography: "{typography.option}"
    rounded: "{rounded.base}"
    padding: "0.2rem 0.5rem"
  menu-field-recorded:
    textColor: "{colors.ink}"
  menu-field-changed:
    textColor: "{colors.changed}"
  menu-switch:
    backgroundColor: "{colors.field}"
    rounded: "{rounded.base}"
    width: "2.25rem"
    height: "1.25rem"
  menu-switch-on:
    borderColor: "{colors.accent}"
  info-area:
    backgroundColor: "{colors.panel}"
    padding: "0.75rem 1.5rem"
  step-file:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    typography: "{typography.file}"
    padding: "0.75rem 1rem"
  screen-frame:
    backgroundColor: "{colors.ground}"
    rounded: "{rounded.base}"
  aside:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.muted}"
    rounded: "{rounded.base}"
    padding: "1rem 1.5rem"
  item-row:
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    padding: "0.75rem"
  item-group-head:
    textColor: "{colors.accent}"
    typography: "{typography.label}"
    padding: "2rem 0.75rem 0.5rem"
  download-row:
    textColor: "{colors.ink}"
    typography: "{typography.headline}"
    padding: "1rem 0.75rem 1rem 2rem"
  download-row-mine:
    backgroundColor: "{colors.select}"
  docs-page-link:
    textColor: "{colors.muted}"
    typography: "{typography.button}"
    padding: "0.5rem 0"
  docs-page-link-current:
    textColor: "{colors.accent}"
---

# Design System: MCM Lexicon (web site)

## Overview

**Creative North Star: "The Game's Own Menus, Set Larger"**

The site is the app's world, the one recorded in the repository's root `DESIGN.md`, set larger for a page that has to persuade. It is a sibling of Hermaeus's site and keeps that site's token set role for role: the same smoked-glass ground, ink, two strengths of hairline, verdigris and frost blue accent, 3px corners, the diamond, and the same two typefaces. From the app it adds the meanings MCM Lexicon needs: DM Mono for anything matched character for character, the File Markup colors, copper on a value that differs from the mod's default, the smoked field, and the small amber diamond on a step not saved yet. A visitor who downloads MCM Lexicon opens a program that looks like the page they just left.

The first viewport is a working menu, not a pitch. The offer runs across the top: the name at display size and the hook at left, and one line of support, the Windows download, and the fine print at right behind a strong hairline. Under it, full width, sits a slice of WeaponThrowVR's real menu, drawn the way MCM Lexicon draws it, beside the step file it writes; every value set in the menu lands in the file, live, spelled as the mod spells it. The rest of the page follows Hermaeus's site: why it exists with "By the way…" asides, real screens with sticky captions and the one in view lit, the app's item table, the safety rules under hairlines, and one download row. The game's button-hint strip runs along the bottom, and every hint is a working shortcut. The docs use the same top bar and footer, with the pages listed down the left the way the app lists a menu's pages.

Where the site departs from the app: a longer type scale (display, headline, and lead are site-only), a 1rem body instead of 15px, larger buttons, a section rhythm with no app equivalent, full-width sections instead of the app's shell, a working menu as the hero, framed screens with scroll dimming, and the diamond widened from "the selected row" to "the selected thing". Inside the hero's menu, the app's own scale returns: 15px option labels, the smoked field, and the game's two-column grid, because that part is the app.

**Key Characteristics:**
- The app's smoked-glass ground and token set, unchanged, in both themes.
- A real menu, working, as the hero, writing the exact step file beside it.
- Larger steps: a 6rem name and 3.75rem section titles; the app's own scale inside the menu.
- One section rhythm: a caps title, a muted lead, then the material, with large gaps between sections.
- Hairlines, not cards: no section, screen, or row is a filled tile, and nothing lifts on a shadow.
- The bottom hint strip, D, S, and N, each a real shortcut.

## Colors

The app's near-monochrome palette, unchanged: off-white on near-black in dark, near-black on pale smoke in light, with one accent that changes material with the theme, verdigris in dark and frost blue in light. Every value matches the app's role for role; `data-theme` on the root is "dark", "light", or absent to follow the OS.

### Primary
- **Verdigris** (`accent` / `accent-light`): selection and action. The primary download button's fill, the chosen theme option, the switch's knob, edge, and faint wash when on, the slider's diamond knob, the current page in the menu and in the docs list and their diamonds, the selected option's diamond and its outline, header rows in the menu and group headings in the item table, the keys in the step file, the lit screen's title, link underlines (mixed 75% into the ground at rest, full on hover), outline buttons' and fields' hover border, focus rings, text selection, and the caret. The hook's promise, "with Skyrim closed", is set in it as text.
- **Accent Ink** (`on-accent` / `on-accent-light`): text and the diamond on an accent fill.

### Secondary
- **Rose Invitation** (`invite` / `invite-light`): the site's own color, for what the page asks you to touch: the TRY IT and its arrow above the window, the window's outline, the FLIP THIS chip in the menu, and the numbered callouts on the docs' screenshots. Chosen for how it looks beside the rest, not for the world: the page's palette is warm (cream, warm gray, amber, copper) with one cool-leaning green, and rose is the warm hue it lacks and the near-complement of that green, so it pulls the eye and still belongs. A warm pink in dark (`#f0a3b5`) and the same rose deepened to raspberry for pale smoke in light (`#b0325a`). Never a fill larger than a chip, and never on anything the app itself shows.
- **Invitation Ink** (`on-invite` / `on-invite-light`): text and the arrow on an invitation fill.
- **Copper Changed** (`changed` / `changed-light`): in the hero's menu, a recorded value that differs from the mod's default (a slider's number or a menu's chosen entry), as the app shows it. The value only, never a row or a fill.

### Neutral
- **Menu Black** (`ground` / `ground-light`): the outer ground and edge of the radial gradient, the hint strip's opaque fill, the backing of each screen frame, and (mixed 35% with transparency) the demo frame's backing.
- **Smoke Heart** (`smoke` / `smoke-light`): the lighter center of the fixed radial gradient (an ellipse 70% by 60% at 65% 25%, fading to the ground at 75%).
- **Pane Glass** (`panel` / `panel-light`): the app's details-pane material. The step file, the menu's info area, every "By the way…" aside, code blocks in the docs, and a menu field's open list.
- **Smoked Field** (`field` / `field-light`): the backing of the menu's number fields, choice fields, and switch tracks, so a value reads as editable before it's touched.
- **Selected Row Haze** (`select` / `select-light`): the selected option in the menu (and, at 60%, the option under the pointer), the download row for Windows, and the backing of inline file names and code.
- **Parchment Ink** (`ink` / `ink-light`): reading text, titles, key caps, values in the step file, a recorded value in the menu, and links inside leads and asides.
- **Weathered Ink** (`muted` / `muted-light`): leads, captions, the support line and fine print, an unrecorded field's value (the mod's default), punctuation in the step file, the file's path up to its name, page counts, the info area's facts, section links, hints, and theme options at rest, aside text, the footer, and provenance lines.
- **Hairline** (`rule` / `rule-light`): the top bar's underline, menu rows, item rows, download rows, the info area's and the file's inner edges, the footer's top edge, and the hint strip's top edge.
- **Strong Hairline** (`rule-strong` / `rule-strong-light`): the offer's left edge, the demo frame and the file's edge beside the menu, menu header rows, screen frames, the rule above each screen caption and each safety rule, column-head and group-head underlines, the top of the download list, the docs title's underline, button, field, switch, and theme-switch borders, asides, and key caps.

### Status
- **Amber Warning** (`warn` / `warn-light`): the window's RECORDING lamp, its glow, and its count; the small amber diamond on a menu option whose step is in the draft but not saved, and the quoted state "Skyrim running · read-only" in the safety rules, as the app shows them. Never a fill on the site.

### Named Rules
**The Same Token Set Rule.** The site defines one color of its own, Rose Invitation, as a dark and light pair. A color the site needs is a role the app already has, with the app's dark and light values; a new role ships in both, as a pair, or not at all.

**The Changed Rule.** Copper means the player set this, and it differs from the mod's own default. It marks the value itself, never a row, a fill, or a highlight, and never doubles as a warning.

**The Draft Diamond Rule.** Amber on the site is the app's amber: the small diamond on a step not saved yet, and the read-only state. It is never decoration and never a fill.

**The Scarce Accent Rule.** The accent marks what's selected, what's actionable, and group headings inside a list. The largest accent fill is the primary download button. One exception, inherited from Hermaeus's site: the hook's promise is set in the accent, as text and never as a fill.

## Typography

**Display Font:** Barlow Condensed 500 and 600 (with Jost, sans-serif), self-hosted via @fontsource
**Body Font:** Jost 400, 500, and 600 (with system-ui, sans-serif), self-hosted via @fontsource
**File Font:** DM Mono 400 (with ui-monospace, monospace), self-hosted via @fontsource

**Character:** The app's pairing, at the scale of a page: tracked condensed capitals are the game's menu voice, large enough to carry the name and section titles; Jost carries everything that's read. DM Mono, geometric like Jost, carries what MCM Recorder matches character for character.

### Hierarchy
- **Display** (Barlow Condensed 600, up to 6rem, line-height 0.9, 0.06em tracking, uppercase): the name in the hero only. It fills its column and never passes it (capped at 13% of the column's inline size, 15% when stacked), with its tracking halved from the label tracking because it's that large.
- **Headline** (Barlow Condensed 600, up to 3.75rem, line-height 1, label tracking, uppercase): section titles, the docs page title, and Windows in the download row. The menu's name in the demo runs smaller (1.5 to 2rem), as the app's headline caps.
- **Title** (Barlow Condensed 600, 1.375rem, uppercase): screen captions, safety rule names, the name in the top bar, section headings inside a docs page, and the previous and next links at a docs page's foot.
- **Lead** (Jost 400, 1.375rem, line-height 1.35, balanced): the hook under the name, whose promise is in Jost 600 in the accent and never breaks across lines.
- **Body** (Jost 400, 1rem, line-height 1.55, tabular figures): prose, captions, table cells, aside text. Section leads run at 1.125rem in muted ink with a 60ch measure; rule text holds 44ch. The fine print runs at 0.8125rem in muted ink, with nothing set apart in it.
- **Prose** (Jost 400, 1.0625rem, line-height 1.65): the docs' reading column, at most 44rem wide.
- **Option** (Jost 400, 0.9375rem, line-height 1.3): option labels, field values, and help text inside the hero's menu, at the app's own size. A long label wraps rather than lose its end.
- **File** (DM Mono 400, 0.8125rem, line-height 1.55): the step file, its path, and the selector in the info area; code blocks in the docs at 0.875rem, inline code at 0.9em.
- **Button** (Barlow Condensed 500, 1.125rem, label tracking, uppercase; 600 on the primary): buttons, section links, and docs page links.
- **Hint** (Barlow Condensed 500, 1rem, label tracking, uppercase): the hint strip, the menu's page list, and the footer's links.
- **Label** (Barlow Condensed 500, 0.8125rem, 0.12em tracking, uppercase): column heads, group heads, menu header rows, theme options, the Click control, the window's key hints, the docs list's heading, and provenance lines. The credit and the info area's facts run at 0.75rem.

### Named Rules
**The Label Versus Read Rule.** As in the app: anything that labels is tracked condensed caps; anything that's read is Jost in sentence case. A note inside a button (such as "64-bit") switches back to Jost at 0.8125rem with no tracking; a page count beside a caps page name is Jost figures.

**The File Font Rule.** DM Mono is for what MCM Recorder matches exactly: the step file, its path, and a selector. Only color sets a file's parts apart, never weight.

## Layout

Sections are full width, inset by a fluid gutter (1rem to 3.5rem), on the one fixed gradient. The top bar holds the mark and name at left, then the section links and the theme switch at right, over a hairline; under 60rem the links drop to a row of their own under the name and scroll sideways.

The hero has two tiers. Across the top, the offer: a grid of the pitch (name and hook) and a 19 to 28rem column (support, download, fine print) behind a strong-hairline left edge, separated by a column gap of 2 to 6rem and aligned to their foot; under 60rem the columns stack without the edge. Under it, full width, the demo: the menu and the step file side by side (the file 19 to 25rem), the menu's body taking what the window has left (15 to 30rem, the window's height less 36rem), so the offer and the whole menu share the first viewport. The file is as tall as the menu and scrolls inside it. Under 70rem the file moves under the menu (at most 18rem of it before it scrolls); under 48rem the menu's pages wrap into a sideways-scrolling row above the options, which run in one column, the left column's options first.

**The Section Rhythm Rule.** Every section after the hero is the same: the largest spacing step above it (5 to 9rem), a caps headline with a muted lead under it (up to 52rem wide), a 4rem gap, then the material. Sections don't get backgrounds or borders of their own to set them apart; the rhythm does that.

**The Scroll Inside Rule.** The demo's menu, page list, and file scroll inside themselves, never the page: the selected option starts in view on a whole row, a written step scrolls the file to it, and the options don't pass their scroll on to the page.

Why sets its story (up to 52rem) beside a 16 to 24rem column of asides, stacking under 60rem. The screens list stacks each screen with a 5 to 9rem gap, each a caption column (14 to 20rem) beside the frame, the caption sticky 2rem from the top; under 60rem the caption sits above and stops sticking. The item table caps at 64rem; under 30rem its rows become stacked name-and-description pairs. The safety rules flow in an auto-fit grid of columns at least 22rem wide, up to 80rem. The download row is a single 48rem list.

The docs are a 16rem page list, sticky 2rem from the top, beside a 44rem reading column, up to 84rem in all; under 60rem the list becomes a sideways-scrolling row above the page, scrolled to the current one.

The hint strip is fixed along the bottom. It hides under 60rem and on coarse pointers, since its shortcuts are for keyboards; its tip hides under 75rem, where it would be cut short. The footer leaves 6rem beneath itself so the strip never covers it.

## Elevation & Depth

The site is flat, like the app. Depth comes from the smoked-glass gradient, from hairlines, from Pane Glass against the translucent ground, and from the screens, which are opaque images in hairline frames. Nothing lifts on a shadow: not screens, buttons, asides, or the demo.

The one depth effect is focus by light: while the visitor scrolls the screens, the screen nearest the middle of the window is lit and the others dim (opacity 0.38) and desaturate (0.4); their captions never dim, easing over 600ms. The dimming turns on only when script can observe the scroll, switches off under 60rem, and drops its transition under reduced motion.

### Named Rules
**The No Lift Rule.** Nothing on the page floats except the window, which carries the app's modal lift (`0 1.5rem 3rem rgb(0 0 0 / 0.45)`) because it's a program in front of the page; nothing else has a drop shadow. Emphasis comes from light (the lit screen), the accent, the haze, and the diamond. The selected option's 1px accent outline is drawn inside its row and is a line, not a lift, and the window's Rose Invitation outline is the same line at the window's scale, in the invitation's hue.

## Shapes

Nearly square corners, as in the app: one 3px radius on buttons, fields, switches, the theme switch, key caps, asides, the demo frame, screen frames, inline code, and a step's highlight in the file; tables, rows, and sections are square, and the switch's knob is 2px. The only drawn geometry is the app's diamond, a 10-unit SVG drawn at 9px, and the mark. The diamond marks the selected thing: the current page and the selected option in the menu, the current docs page, the lit screen's title, the Windows download row (in a column of its own, as in the app's tables), and inside the primary download button in accent ink. The same shape, at 7px in amber, marks a step not saved yet, and at 12px in the accent it is the slider's knob on its track. It is hidden everywhere else.

## Components

### Buttons
Outlined caps that light up in the accent, larger than the app's.
- **Shape:** gently squared (3px radius), 1px strong-hairline border, padding 0.75rem by 1.5rem.
- **Default:** transparent, ink text, button typography; hover turns the border the accent.
- **Primary:** accent fill and border, accent-ink text at weight 600, with the diamond, and a short Jost note ("64-bit") at its far end. One per group: Download for Windows in the hero, full width of its column, and Download in the download row.
- **Focus:** a 2px accent outline offset by 2px, on every control.

### Theme Switch
The app's segmented choice: a 3px outlined group of label caps (Auto, Dark, and Light) in muted ink; the chosen one fills with the accent in accent ink at weight 600. The choice is remembered and applied before first paint.

### Navigation
Section links in the top bar use button caps in muted ink, turning ink on hover; the current one is ink at weight 600. A skip link in the accent appears on focus. The footer's links (Docs, Source, Releases, Issues) are hint-size caps beside the mark at 2.5rem and the fine print.

### Working Menu (signature)
**The Real Menu Rule.** The hero's menu is WeaponThrowVR's, as MCM Lexicon reads it from the Synergy list, shown with Shizof's permission and credited in its header. Its pages, options, positions, help, ranges, and defaults are the catalog's; its labels are spelled as the mod spells them. Nothing in it is invented.

Drawn as the app draws a menu, inside a 1px strong-hairline frame with 3px corners: the menu's name in headline caps with the mod, plugin, and "Source" in muted Jost beneath, and the credit in muted label caps at the right. A 12rem page list down the left, each page a caps row with its option count in muted Jost figures, the current one in the accent at weight 600 with the diamond. The options to its right on the game's two-column grid, each option in the row and column SkyUI's cursor gave it, each row at least 2.5rem on a hairline; header rows are label caps in the accent over a strong hairline. Hovering a row lays the haze at 60%; the selected row takes the full haze, the diamond, and a 1px accent outline inside it.

Values sit at a row's right edge in the smoked field with a strong hairline, its border turning the accent on hover. Not recorded yet, a field shows the mod's default in muted ink and a switch is dimmed to 45% (full on hover and focus); recorded, the value turns ink, copper where it differs from the default, and the small amber diamond appears before it. The switch is the app's: a squared track whose knob slides right (140ms) and lights in the accent, the track taking an accent edge and a 22% accent wash over the field; off, the knob is muted and the edge turns ink on hover. The knob is the light: a filled track with a dark knob read as off inverted. A slider's number is a field; while its row is being edited, a diamond knob on a 1px track in muted ink at 80% (a rule measures under 2:1, and a control's line needs 3:1) appears beside it (hidden under 48rem). A text option is a CLICK control in outlined label caps.

Under the grid, where SkyUI shows its info text, an info area on Pane Glass over a hairline: the selected option's help in Jost (two lines at most, whole under 48rem), then one muted label-caps line with its selector in DM Mono and ink, its type, and what it allows. Up and down arrows move between options in a column, every control is reachable by keyboard, and each written step is announced politely to screen readers.

### The Window (signature)
The hero is MCM Lexicon's own window, running, in front of the page: the one thing the screenshots further down can't be. It's outlined (1px) in Rose Invitation, the line the selected row wears inside it in the accent, because it's the thing to touch, and lifted with the app's modal lift (`0 1.5rem 3rem rgb(0 0 0 / 0.45)`), the site's one shadow. Above it, at the right end of the hook's line (from 60rem up) or right-aligned just above the window (below), TRY IT in Rose Invitation: label caps at weight 600 from 1.375 to 2rem with a drawn down arrow at 1em that nudges 0.3rem toward the window three times a second after the page loads (none under reduced motion) and dips 2px on hover; it's a link to the window. Its top bar is the app's, at the app's size: the mark and name at left; at right, LIST and PROFILE in muted label caps with their values (the list's name in the File font), SKYRIM CLOSED in muted label caps, and the lamp. Under the bar, the menu screen beside the details pane; along the bottom, the hint strip. The site's own strip steps aside (slides down and fades over 360ms, none under reduced motion) while at least a third of the window is on screen, so the page never shows two strips, and returns once the window is scrolled past.

**The Invitation.** In the row of one option not recorded yet (Allow Axe Throw, two rows under the selected one at the top of General), between its label and its switch, a Rose Invitation chip: FLIP THIS and a drawn arrow pointing at the switch, label caps at weight 600 in invitation ink, 3px corners. It arrives 700ms after the page (6px slide, 320ms), then nudges 5px toward the switch three times (1.4s each), none under reduced motion. Clicking it flips the switch, so the visitor's first act adds a step and the lamp counts up. The first pointer or key anywhere in the window sends it away (320ms fade) for good. Between 70 and 86rem, where the menu's columns are narrow beside the pane, it reads FLIP → so the label beside it keeps its lines.

**The Lamp.** RECORDING in Save's amber (`warn`), label caps at weight 600, with the 9px diamond before it carrying Save's glow (a drop shadow in amber, breathing between 3px and 7px over 2.8s, none under reduced motion) and "· N STEPS" after it, the count in tabular figures. Amber here means what it means in the app: a draft that isn't on disk. The count rises in (4px, 320ms) each time a step lands, as Save does in the app. The lamp is the site's only repeating motion.

**The Window's Hint Strip.** Along the window's bottom over a hairline, as the app's: right-aligned, the keys this screen offers, each a key cap (1px strong hairline, 3px corners, ink) and a muted caps label, and each a real key and a button that does the same. ↑↓ MOVE and R START OVER always; then by the selected option's kind, SPACE ON/OFF for a toggle, SPACE CLICK for a text option, and ENTER APPLY for a slider. A key cap presses in (2px dip, the accent) whichever way it ran. Keys are for keyboards: under 60rem or on a coarse pointer the window has no strip.

Rows a click acts on (toggles and text options) show a pointer.

### Step File (signature)
The file the menu writes, beside it on Pane Glass behind a strong hairline: its path above in DM Mono (the folders muted, the file name in ink) the file, and a muted foot saying every label is spelled as the mod spells it. The file is written as MCM Lexicon writes it (lowercase keys in JContainers' order, two-space indents, a slider's value always with a decimal point). A step just written flashes the accent at 22% behind it and fades over 1.6s, unless reduced motion is set.

**The File Markup Rule.** As in the app: keys in the accent, since they're the words MCM Recorder reads; values in ink, since they're what it fills in; punctuation muted.

### Screens
A screen of the app in both themes, showing the one that matches the page, in a 1px strong-hairline frame with 3px corners and no shadow. Each has a caption with a title-step caps name over a strong hairline and a muted description; the screen dims until it's lit, its caption never does; when lit, its name turns the accent and shows the diamond. A provenance line in muted label caps follows the list.

### Docs Figure
A screenshot in the docs is the app in a strong-hairline frame with 3px corners on the game's ground, as the landing page frames its screens, with its caption in muted Jost under it. Both themes' captures ship and the page shows the one matching its theme. A figure may run past the reading column toward the page's edge, up to its own 1x size (a whole window at 1280px, a crop at its own width) and never enlarged, since a window at the column's 44rem is unreadable. Callouts are numbered badges in Rose Invitation drawn into the capture; their caption is a numbered list in two columns, each number in the same rose badge, so the list reads with the picture.

### By the Way… Aside
Tangents, caveats, and jokes pulled out of the main flow, on the landing page and in the docs alike: Pane Glass, a strong hairline, 3px corners, padding 1rem by 1.5rem, a short title in ink in 1.125rem caps at weight 600, then muted Jost. Links inside are ink. Load-bearing text (download steps, safety rules) never goes in an aside.

### Item Table
The app's item table at page scale: muted label-caps column heads over a strong hairline, group heads in the accent with a strong hairline, names in button-size caps, descriptions in Jost, and a hairline under each row. No row hover or selection; the table is read, not used. The docs' tables use the same heads and hairlines at body size.

### Download Row
One row for the one system there's a build for: Windows in headline caps with the diamond in a column of its own, "64-bit" in muted Jost, and the primary button, on the haze, between a strong hairline above and a hairline below. A muted line after it says where macOS and Linux builds come from.

### Docs Page List
As the app lists a menu's pages: a muted label-caps heading over a strong hairline, then each page in button caps in muted ink, turning ink on hover; the current page in the accent at weight 600 with the diamond in a 1.25rem column of its own. At a page's foot, previous and next as title caps over a strong hairline, each under a muted label ("Previous", "Next"), turning the accent on hover.

### Inline Code
A file name or command in running text is DM Mono at 0.9em, in the color of the text around it, on the haze with 3px corners, on the landing page and in the docs alike. Code blocks in the docs are DM Mono on Pane Glass in a strong-hairline frame.

### Button-Hint Strip (signature)
As in the app: fixed along the bottom on opaque ground, a hairline above, hints at the right. Each hint is a button, a key cap (1px strong-hairline border, 3px radius, ink, at least 1.6em wide) then the action in muted caps, turning ink on hover. The site has three: D jumps to Download and focuses its button, S jumps to Safety, and N moves to the next section, then back to the top. Every hint is a real shortcut, ignored while a modifier is held or focus is in a field, and clicking it does the same thing. Its key cap presses in: it dips 2px and lights in the accent for a moment (220ms), unless reduced motion is set.

**Loading-screen tips.** The strip's left end shows a tip, as Skyrim's loading screens do: muted Jost at 0.8125rem, in a random order per visit, the next on a click or whenever a new section reaches the middle of the window, fading in from a slight blur (360ms). None repeats until every one has been shown. Every tip is a true fact about MCM Lexicon, short enough to fit on one line at 75rem.

## Do's and Don'ts

### Do:
- **Do** use the app's tokens role for role, in both themes, and add a new color only as a dark and light pair.
- **Do** give every section the same rhythm: the largest spacing step above it, a caps headline and muted lead, then the material.
- **Do** show the real thing: the real menu from the catalog, real screens in hairline frames in both themes, and the exact file MCM Lexicon writes, with provenance lines for where they came from.
- **Do** put the diamond on the selected thing, and only there; amber diamonds only on steps not saved yet.
- **Do** set anything MCM Recorder matches exactly (step files, paths, selectors) in DM Mono, with keys in the accent, values in ink, and punctuation muted.
- **Do** keep the demo's scrolling inside the demo; it never scrolls the page.
- **Do** back every hint in the strip with a working shortcut, and hide the strip where there's no keyboard.

### Don't:
- **Don't** use cards, filled tiles, or section backgrounds; separate with hairlines and the section rhythm. Pane Glass is for the file, the info area, asides, and code blocks.
- **Don't** put a drop shadow on anything, screens included.
- **Don't** use copper anywhere the app wouldn't, or as a highlight; it marks a value that differs from the mod's default.
- **Don't** add a second accent, or use the accent for decoration.
- **Don't** invent or tidy a menu's labels in the demo; they're spelled as the mod spells them.
