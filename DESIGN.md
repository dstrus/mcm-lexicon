---
name: MCM Lexicon
description: A catalog of every MCM option in a mod list, and an editor for MCM Recorder recordings, that looks like one more menu of the game you just closed.
colors:
  accent: "#7fb8a0"
  on-accent: "#0e1a15"
  ground: "#0b0b0b"
  smoke: "#1c1c1c"
  panel: "#121212"
  glass-strong: "rgb(0 0 0 / 0.38)"
  select: "rgb(242 239 230 / 0.07)"
  ink: "#f2efe6"
  muted: "#a8a293"
  rule: "#2a2a2a"
  rule-strong: "#3a3a3a"
  warn: "#d9a441"
  danger: "#e57368"
  changed: "#d2996b"
  accent-light: "#2a5f86"
  on-accent-light: "#eef4f9"
  ground-light: "#e4e3df"
  smoke-light: "#f3f2ef"
  panel-light: "#eceae6"
  glass-strong-light: "rgb(255 255 255 / 0.55)"
  select-light: "rgb(27 26 23 / 0.06)"
  ink-light: "#1b1a17"
  muted-light: "#5d5a52"
  rule-light: "#cdcbc4"
  rule-strong-light: "#b3b0a7"
  warn-light: "#8a5a00"
  danger-light: "#b3261e"
  changed-light: "#9a4a20"
typography:
  display:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "2.5rem"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "0.12em"
  headline:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "1.375rem"
    fontWeight: 600
    letterSpacing: "0.12em"
  title:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "1.125rem"
    fontWeight: 500
    letterSpacing: "0.12em"
  body:
    fontFamily: "Jost, system-ui, sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 400
    lineHeight: 1.5
    fontFeature: "tnum"
  label:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 500
    letterSpacing: "0.12em"
  button:
    fontFamily: "Barlow Condensed, Jost, sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 500
    letterSpacing: "0.12em"
  file:
    fontFamily: "DM Mono, ui-monospace, monospace"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.55
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
components:
  button:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    typography: "{typography.button}"
    rounded: "{rounded.base}"
    padding: "0.25rem 0.75rem"
  button-primary:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.on-accent}"
    typography: "{typography.button}"
    rounded: "{rounded.base}"
    padding: "0.25rem 0.75rem"
  input:
    backgroundColor: "{colors.glass-strong}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.base}"
    padding: "0.5rem 0.75rem"
  table-row:
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    padding: "0.5rem 0.75rem"
  table-row-selected:
    backgroundColor: "{colors.select}"
    textColor: "{colors.ink}"
  column-head:
    textColor: "{colors.muted}"
    typography: "{typography.label}"
    padding: "0.5rem 0.75rem"
  column-head-sorted:
    textColor: "{colors.accent}"
  details-pane:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    padding: "1.5rem"
    width: "clamp(18rem, 26vw, 24rem)"
  hint-bar:
    backgroundColor: "{colors.ground}"
    textColor: "{colors.muted}"
    typography: "{typography.button}"
    padding: "0.5rem 2rem"
  hint-key:
    textColor: "{colors.ink}"
    rounded: "{rounded.base}"
    padding: "0 0.4em"
  tab:
    textColor: "{colors.muted}"
    typography: "{typography.title}"
    padding: "0.5rem 0"
  tab-active:
    textColor: "{colors.ink}"
---

# Design System: MCM Lexicon

## Overview

**Creative North Star: "The Game's Own Menus"**

MCM Lexicon's design system is Hermaeus's, inherited whole: the same tokens, fonts, shell, parts, and rules, with only the meanings its own product needs added (copper, certainty, validation, the File font, the Draft Rule). MCM Lexicon is one more menu of the game you just closed. Every screen is a categories column, a sortable item table, and a full-height details pane, set on smoked glass with hairline rules and driven from the keyboard. It deliberately refuses the admin-dashboard default of gray sidebar, cards, and a blue button: there are no cards, no filled surfaces beyond the details pane, and only one more color, copper, kept for what the player set.

The world is dense and quiet. Text is Jost at a compact 15px with tabular figures so counts, times, and importance scores line up in columns; anything that labels rather than reads (categories, headings, column heads, keys, and buttons) is set in tracked, uppercase Barlow Condensed. The accent is scarce and means "this is selected" or "this is what you can do". Dark mode is the game's near-black menus; light mode is the same world in pale smoke, built from one token set with the same roles.

The signature is the bottom button-hint strip: every hint is a real keyboard shortcut, and clicking the hint does the same thing. The SVG diamond marks the selected row, as it does in the game.

The mark isn't chosen yet; the top bar carries the wordmark alone, and the icon is the game's diamond on near-black as a stand-in (`assets/`). When drawn, the mark follows Hermaeus's in every way but subject: in the top bar before the wordmark in the text color, with its central shape cut out so the ground shows through; as the app icon and favicon in verdigris on the game's near-black, in a rounded square, with a simpler drawing at 16 pixels.

**Key Characteristics:**
- Smoked-glass ground: a fixed radial gradient from a lighter smoke heart to the ground color.
- Hairline 1px rules (two strengths) instead of boxes, cards, or shadows.
- Tracked condensed caps for labels; Jost for reading text; tabular figures everywhere.
- One accent for selection and action, and for the group headings that sort lists into sections.
- Item table plus a sticky, full-height details pane that sizes itself to end at the hint strip.
- A fixed bottom hint strip of live keyboard shortcuts, right-aligned like the game's button hints.

## Colors

A near-monochrome palette, off-white on near-black (or near-black on pale smoke), with one accent role that changes material with the theme: verdigris, the green of old bronze, in dark; frost blue, Skyrim's cold light, in light. Each role has a dark value and a `-light` counterpart; the light set applies under `data-theme="light"` or when the theme is "system" and the OS prefers light.

### Primary
- **Verdigris** (`accent` / `accent-light`): the selection and action color: verdigris in dark, frost blue in light. The diamond marker, the primary button fill, the sorted column head and its hover, the current section in the sections column, the active tab's underline, focus rings, text selection, the caret, checkbox accents, inline text buttons, link underlines (mixed 75% into the ground at rest so a page of links stays quiet but clears 3:1, full on hover), the stat bars (at 55% opacity), and the group headings in tables and stat lists (a menu in search results, a step file in a recording, Compare's Changed list). Both pass WCAG AA on their worst-case ground.
- **Accent Ink** (`on-accent` / `on-accent-light`): text on an accent fill, used for the primary button and selected text.

### Neutral
- **Menu Black** (`ground` / `ground-light`): the outer ground, the edge of the radial gradient, and the hint strip's opaque background.
- **Smoke Heart** (`smoke` / `smoke-light`): the lighter center of the smoked-glass gradient (an ellipse at 65% 25%), and the quick-find dialog's surface.
- **Pane Glass** (`panel` / `panel-light`): the opaque fill of the details pane, so table content doesn't show through while the pane sticks.
- **Smoked Field** (`glass-strong` / `glass-strong-light`): the translucent fill of inputs, selects, and text areas.
- **Selected Row Haze** (`select` / `select-light`): the faint ink wash on the selected table row, the selected quick-find result, and the checked theme option.
- **Parchment Ink** (`ink` / `ink-light`): body text, current navigation, key caps.
- **Weathered Ink** (`muted` / `muted-light`): secondary text, column heads, definition terms, unbuilt sections, hints at rest, placeholders.
- **Hairline** (`rule` / `rule-light`): row dividers and structural borders between shell zones.
- **Strong Hairline** (`rule-strong` / `rule-strong-light`): column-head underlines, group-heading underlines, control borders, the pane's left edge.

### Secondary
- **Copper Changed** (`changed` / `changed-light`): the bronze under the verdigris, for what the player set, where the mod's default is known: a saved MCM Helper value that differs from the config's default, or a recording step's value when the option's default is known and the step sets something else. A value in copper, never a row or a fill. Passes AA on both grounds in both themes.

### Status
- **Amber Warning** (`warn` / `warn-light`): the SAVE button while the draft holds unsaved changes (its one fill), "Skyrim running · read-only", a recording step that won't replay, an option read with less than exact certainty (in the pane's notice), and warning notices or flags.
- **Ember Error** (`danger` / `danger-light`): error notices only.

### Named Rules
**The Changed Rule.** Copper means one thing: the player set this, and it differs from the mod's own default. It marks the value itself, never a row, a fill, or the action on it, and never doubles as a warning; a warning stays amber. A step that sets an option to its own default is therefore not copper, which is useful: it's a step that does nothing.

**The Certainty Rule.** Certainty shows as words, not color. An `exact` option shows nothing extra. `pattern` and `unknown` show as a muted label-caps word after the option's text in tables (PATTERN, UNKNOWN), and in the pane as an amber notice saying what it means: the label is built at runtime, so a recording can only match it with a `*wildcard*`, or can't match it at all. Certainty is never a column of its own.

**The Validation Rule.** A step that won't replay is amber: a label-caps word in the step table (NOT FOUND, NOT ALLOWED, MOD MISSING) and, in the pane, a sentence that says what the catalog has instead, with the suggestion as a text button that applies it. A step that can't be checked is muted (UNCHECKED); a step that matches is silent. A recording that couldn't be read at all is an error.

**The Scarce Accent Rule.** The accent marks selection, action, and group headings that sort a list into sections. A resting choice beside one that acts (READ-ONLY beside EDITING) doesn't take it even when chosen: Segmented's quiet option shows it in ink on the selected haze, so color means the app can change something. It never fills a surface larger than a button and never decorates. If something is in the accent color, it is selected, actionable, or a heading inside a list.

**The One Token Set Rule.** Light mode is the same roles with different values, never a different design. A new color ships as a dark/light pair or not at all.

## Typography

**Display Font:** Barlow Condensed 500 and 600 (with Jost, sans-serif), self-hosted via @fontsource
**Body Font:** Jost 400 and 500 (with system-ui, sans-serif), self-hosted via @fontsource
**File Font:** DM Mono 400 (with ui-monospace, monospace), self-hosted via @fontsource

**Character:** Barlow Condensed in tracked capitals is the game's menu voice, crisp and narrow; Jost is a calm geometric text face that stays legible at small sizes. DM Mono, geometric like Jost, is for anything matched character for character: raw selectors, page names and menu entries as a recording holds them, the files the editor previews, and catalog and recording paths. The scale is short (five steps) because the interface is a menu, not a document.

### Hierarchy
- **Display** (Barlow Condensed 600, 2.5rem, line-height 1, uppercase): the page title only (Menus, a menu's name, Search, Recordings, a recording's name, Compare).
- **Headline** (Barlow Condensed 600, 1.375rem, uppercase): the selected item's name in the details pane, and the app name in the top bar.
- **Title** (Barlow Condensed 500, 1.125rem, uppercase): category links in the sections column and the tabs under a character.
- **Body** (Jost 400, 0.9375rem, line-height 1.5, tabular figures): table cells, values, prose. Long text in the pane runs at line-height 1.6 with a 65ch measure.
- **Label** (Barlow Condensed 500, 0.8125rem, 0.12em tracking, uppercase): column heads, group headings, definition terms in caps, the profile, recording, scan-state, and game-state labels, and the theme switch.
- **Button** (Barlow Condensed 500, 0.9375rem, 0.12em tracking, uppercase): buttons and the hint strip.
- **File** (DM Mono 400, 0.8125rem, line-height 1.55): a raw selector beside its text, a catalog's file name, and a whole file as it will be written, in the editor's Files tab.

### Named Rules
**The Label Versus Read Rule.** Anything that labels is tracked condensed caps; anything that is read is Jost in sentence case. Counts beside a label switch back to Jost with no tracking.

**The Lined-Up Figures Rule.** Tabular figures are on at the body level. Numeric columns align right.

**The Raw Selector Rule.** The translated text is for reading, in Jost; the raw selector sits beneath it, muted, in DM Mono, only when the two differ or when the selector has a space at either end. A trailing space is invisible in any font, so copy names it: "Ends with a space. MCM Recorder matches it exactly."

## Layout

The shell is a CSS grid: a full-width top bar, then an 18rem column beside the main area, with the hint strip fixed along the bottom. The top bar holds the name at left and, at right, only what describes the list and the game: the source (the profile select for an MO2 instance, or the catalog's file name), the scan state, Skyrim's state, and the theme switch. The column is laid out as SkyUI's MCM lays out its left side: the editing session at the top, the sections (Search, Compare, Recordings) under it, and then the menus, which take the rest of the height, each band set off by a hairline. Main content has 1.5rem top padding, 2rem side padding and 5rem bottom padding to clear the hint strip.

List screens (Recordings, a recording's steps, Compare, Search) split into the item table and a details pane of `clamp(18rem, 26vw, 24rem)`. The pane sticks to the top and is sized by script to end exactly at the hint strip, so a form's footer stays in view. A filter bar above the table stops at the pane's edge. A menu is the exception, drawn as SkyUI draws it (Components, The MCM Screen); its home, with no menu chosen, is the list's summary. Without a mod list there is no column, and main is the form that chooses one, at 64rem.

Spacing follows a seven-step scale from 0.25rem to 2.5rem; table cells use 0.5rem by 0.75rem, gaps between zones use 1.5rem, and section gaps on overview grids use 2.5rem.

Responsive behavior: under 70rem, wide columns drop. Under 60rem, the column becomes rows under the top bar (the session as one row, the sections as one, the menus as their filter with a short list under it), the pane stacks below the table with a top rule instead of a left one, less important columns hide, and a menu's page list wraps above its options.

## Elevation & Depth

The world is flat. Depth comes from the smoked-glass gradient behind everything, from hairline rules, and from the one opaque surface (the details pane) against the translucent ground. Rows, sections, and the shell have no shadows.

### Shadow Vocabulary
- **Modal lift** (`box-shadow: 0 1.5rem 3rem rgb(0 0 0 / 0.45)`, backdrop `rgb(0 0 0 / 0.45)`): only for the quick-find dialog, which floats over the page.
- **Tip lift** (`box-shadow: 0 0.5rem 1.25rem rgb(0 0 0 / 0.35)`): only for the tip that shows clipped text whole, which floats over whatever is beneath it.

### Named Rules
**The Clipped Text Rule.** Text cut short with an ellipsis shows its whole self in a tip beneath it when the pointer rests there: the pane's material with a hairline edge, lifted by a soft shadow, never the browser's own tooltip. Text that fits shows nothing.

**The Hairline Rule.** Separate with a 1px rule, never a box or a shadow. Use `rule` between rows and shell zones, `rule-strong` under column heads and group headings and around controls.

## Shapes

Corners are nearly square: one radius (3px) on buttons, inputs, key caps, the theme switch, and the dialog. Tables, the pane, and the shell have square edges. The only drawn geometry is the 10-unit SVG diamond (rendered at 9px, filled with the accent) that marks the selected row, and an 8px triangle on the sorted column head that flips for ascending order. A color option's value shows a 9px square of its color beside its hex.

## Components

### Buttons
Quiet outlined caps that light up in the accent on hover.
- **Shape:** gently squared (3px radius), 1px `rule-strong` border, padding 0.25rem by 0.75rem.
- **Default:** transparent fill, ink text, button typography. Hover turns the border the accent color.
- **Primary:** accent fill and border, accent-ink text. Used once per action group (Save, confirm).
- **Disabled:** 50% opacity, not-allowed cursor.
- **Text button:** accent-colored, underlined, no border, for actions inside a sentence.
- **Focus:** a 2px accent outline offset by 2px, across all controls.

### Inputs / Fields
- **Style:** smoked-field fill, 1px `rule-strong` border, 3px radius, 0.5rem by 0.75rem padding, muted placeholder, accent caret.
- **Labels:** the field's label sits above it in muted text at the label size.
- **Checkboxes and radios:** native, with the accent color.

### Item Table
The core of every list screen.
- Collapsed borders, 1px `rule` under every row, cells padded 0.5rem by 0.75rem.
- A narrow first column (1.5rem) holds the diamond for the selected row.
- The selected row gets the `select` haze; inactive rows drop to muted text.
- Column heads are sort buttons in label caps, muted, accent-colored when sorted or hovered, with a `rule-strong` underline. Counts and dates sort largest or newest first.
- Group heading rows (a menu in search results, a step file in a recording) are label caps in the accent with generous top padding (1.5rem) and a `rule-strong` underline.
- A selectable row tints `glass` under the pointer and `select` when selected; heading rows don't react.
- When the selection changes, the details pane's content enters with a short fly-in (6px rise, 180ms); the rows themselves don't move.

### Details Pane
- Opaque `panel` fill, a `rule-strong` left edge, 1.5rem padding, sticky at the top, sized to end at the hint strip.
- The item's name in headline caps, then a definition grid (7 to 7.5rem term column, muted terms at the label size), then actions.
- Forms inside the pane keep their buttons in a sticky footer with a `rule` top edge.
- Every list has one. Steps are edited in the pane, never inline: the form swaps in with `?mode=edit` in the address, and its footer says the change is kept in the draft until the recording is saved.

### The MCM Screen
A menu is drawn as SkyUI draws it, so what the player reads here is what they'll see in the headset. The menu's name in headline caps with the mod, plugin, and kind beneath. A 13rem page list down the left, each page a title-caps row with the diamond on the current one and its option count at the right edge in muted figures. The options to its right on the game's own grid: 64 rows of two columns, each option in the row and column SkyUI's cursor gave it (read from the scripts' cursor calls, or from MCM Helper's fill mode and positions), so the two columns line up row for row and a slot nothing fills stays blank, as in the game. A row is the option's text at the left and its value at the right edge, on a hairline: a toggle's switch, a slider's number in its format, a menu's chosen entry, a key's name, a color's swatch, a text option's value. A header is a group heading row in the accent over a strong hairline; an empty option is a blank row. The row's value is what the game would show: while editing with a recording chosen, the recorded value, edited in place (the Switch, or the compact editor on click or E) in a cell drawn as a field at rest, the same glass and hairline as an input, so it reads as editable before it's clicked, with a dash in it where the option isn't recorded yet (a Switch is dimmed there instead); a recorded option carries a small hollow diamond before its value, and one whose step the draft hasn't saved yet a small amber diamond, the Save button's color, so a row says at a glance whether it's in the recording and whether that's on disk; otherwise the menu's known value, copper where the player's saved value differs from the default, and a dash where the catalog doesn't know it. The selected row takes the diamond and the selected haze. Under the grid, where SkyUI shows its info text, on Pane Glass out to main's edges and, on a page taller than the window, stuck above the hint strip so the selected option and its details are always in view together (a row scrolled to stops above it): the selected option's help in Jost, then one muted line of label caps and raw text with the selector, index, type, what it allows, default, saved value, where it's defined, and its step in the recording, with the certainty notice in amber when the label isn't exact. There is no details pane on this screen.

### Navigation
- **The column:** three bands over hairlines. The session first (below). Then the sections, SEARCH, COMPARE, and RECORDINGS, title-size caps in muted ink; hover turns ink, and the current one is the selected item, in the accent at weight 600. Then MENUS with its count as a label-caps link to the list's home, a filter field, and every menu by display text in SkyUI's alphabetical order, each a title-caps row with a slot for the diamond, which marks the one on screen (in the accent); inactive menus, which the game wouldn't list, sink below an INACTIVE label at 55%. The list scrolls on its own.
- **The editing session,** at the column's top, always: the Segmented READ-ONLY / EDITING, where only EDITING takes the accent fill and a chosen READ-ONLY rests in ink on the selected haze, so the switch carries color exactly when changes can be staged; RECORDING (ADDING TO while editing) in label caps over the select of the recording to look at or add steps to (its last entry, while editing, NEW RECORDING…, which names one at once); the chosen recording's name under it as a title-caps link to its page with a hollow diamond before it, in full ink while it has unsaved changes, in amber when its mod is disabled, and in the accent when it's the current page; and, while editing, CHANGES with the draft's count at its right edge. Once the draft holds anything, that row gives way to SAVE in an amber fill (Pane Glass text, weight 600), the one amber fill in the app, so unsaved work reads from across the room as the warning it is; it rises in when the draft starts and again as each change lands, then breathes a soft amber light (2.8s, none under reduced motion). REVIEW sits beside it as a muted text link to the Changes page and a muted label-size line under it saying how many recordings have changes not yet saved.
- **Tabs:** title-size caps in muted ink, spaced 1.5rem apart over a `rule` baseline; the current tab is ink at weight 600 with a 2px accent underline. A tab's count follows in muted Jost: a menu's pages carry their option counts, a recording's Steps and Files their counts.
- **Theme switch, and any segmented choice:** a 3px-radius outlined group of caps options; the chosen one fills with the accent, in accent-ink text at weight 600. The game's menus mark the chosen item plainly, and so do these.
- **Switch (on or off):** a Boolean is a sliding switch, not Hermaeus's check box: a squared track (the one 3px radius, never a pill) with a square knob that slides right as the track fills with the accent; off, a hairline track on smoked field with a muted knob. The knob eases over 140ms with an exponential ease-out, and not at all under reduced motion. A switch that shows a value not yet set (an option not recorded yet) is drawn at 45% until hovered. A choice between named modes (READ-ONLY / EDITING) stays a Segmented, and a toggle whose value isn't known is chosen outright, ON or OFF.
- **Range (a slider's value):** when a slider's range is known, its value is set on a slider as well as typed: a 1px Strong Hairline track with the diamond as its knob, 12px, in the accent, turning to ink on keyboard focus; no fill, no ticks. The number field sits at its right, 4.5rem in a table cell and 6rem in the pane, in tabular figures, for an exact value. The two are one editor: the knob applies as the pointer lets go, the field on Enter, and moving between them isn't leaving. Without a range there's only the field.

### Button-Hint Strip (signature)
Fixed along the bottom of the viewport, opaque ground, `rule` top edge, hints right-aligned with 1.5rem gaps. Each hint is a button: a key cap (1px `rule-strong` border, 3px radius, ink, min 1.6em wide) followed by the action in muted caps; hover turns it ink, unavailable hints fade to 45%. Every hint is backed by a real shortcut registered in the shortcut registry, and the strip shows only what the current screen offers. Enter on a focused link or button is never a shortcut: it does what that link or button does, and a page's Enter shortcut runs only when focus is elsewhere.

### Stat List
A name-and-count list under a label-caps heading in the accent. Each row carries a 2px accent bar at 55% opacity under the name, scaled to the count; counts are muted and right-aligned.

**The Numbers Are Doors Rule.** When a stat list's rows are the page's navigation (Compare's Changed list, whose rows filter the change table), each name is a title-size caps link, muted at rest, beside a 1.25rem marker column. The diamond and the `select` haze mark the row the arrow keys reach; hover and focus move them, and Enter applies it, with the hint naming the row.

### Quick Find
A dialog (28rem max, 18vh from the top) on the smoke surface with a strong hairline, 3px radius and the modal lift, opened with `/` from any page; it lists menus by display text, with the diamond on the selected one, and Enter opens the menu. The dialog that asks which recording a step goes to, when none is open, is the same surface.

### The Draft Rule
Nothing is written until Save. A recording with unsaved changes says so in a muted line under its title, naming the folder Save will write to, and the hint strip's Save hint is enabled only then. Every notice passes Hermaeus's test: name the file, the backup, and the consequence.

## Do's and Don'ts

### Do:
- **Do** put every list in the item table with a diamond column and a details pane beside it.
- **Do** register a real keyboard shortcut for every primary action and let the hint strip show it.
- **Do** set labels in tracked Barlow Condensed caps and reading text in Jost, with tabular figures.
- **Do** separate with 1px hairlines (`rule`, `rule-strong`) and keep corners at 3px.
- **Do** define every new color as a dark and light pair in the same token set.
- **Do** keep the details pane opaque and sized to end at the hint strip.
- **Do** show a raw selector in DM Mono beside its text when they differ, and say in words when it ends with a space.

### Don't:
- **Don't** use cards, gray sidebars, filled tiles, or a blue (or any second) accent color.
- **Don't** use the accent for decoration or large fills; it means selection, action, or a list's group heading.
- **Don't** add shadows to rows, panes, or the shell; the only shadows are under things that float: the quick-find dialog and the tip for clipped text.
- **Don't** show a hint in the strip that isn't backed by a working shortcut.
- **Don't** make a group label as bright as the current section.
- **Don't** add a column, color, or badge for certainty; it's a word after the label, and a sentence in the pane.
