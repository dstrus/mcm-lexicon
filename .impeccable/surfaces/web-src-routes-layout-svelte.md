---
version: 1
slug: "web-src-routes-layout-svelte"
primary_target: "web/src/routes/+layout.svelte"
related_targets: ["web/src/lib/source/SourceForm.svelte","web/src/routes/menus/+page.svelte"]
---

# App shell and first run

**Mode:** Operate. **Scope:** the shell (top bar, sections column, content area, hint strip) and the first-run form that chooses a mod list or a catalog file. Every page lives inside it.

**Audience and job:** the author at a desktop monitor between play sessions, pointing MCM Lexicon at a mod list once, then picking menus from the column as they would in the game's own MCM, with Search, Compare, Recordings, and Settings a click away. Which profile is read, whether Skyrim is running, whether editing is on, and which recording is chosen are always visible.

**Ranges:** an MO2 instance has one to a handful of profiles; a catalog's file name can be long. States: no list chosen, a list open, a catalog open instead, Skyrim running (read-only), editing with changes to save, a scan in progress.

**Constraints:** Hermaeus's design system, inherited whole (DESIGN.md, PLAN.md §5.2): same tokens, fonts, shell grid, and breakpoints. Fonts bundled in the program. WCAG AA, visible focus, complete keyboard paths. Holds from 1280 to 1600 px; under 60rem the column becomes a scrolling row.

## Direction contract

THESIS: MCM Lexicon is one more menu of the game you just closed, and it is Hermaeus's shell with MCM Lexicon's sections in it: a categories column, a sortable item table, a details pane, and a button-hint strip backed by real keys. It refuses the admin-dashboard default of gray sidebar, cards, and a blue button.

OWN-WORLD: Smoked-glass ground (near-black in dark, pale smoke in light), hairline rules, off-white or near-black text, one accent only for selection and action: verdigris in dark, frost blue in light. Tracked condensed caps (Barlow Condensed) for the wordmark, sections, headings, and labels; Jost for text; DM Mono for what's matched character for character.

STORY: The visitor names their mod list (or opens a catalog) once, lands on the list's home, and picks a menu from the column, which shows it as the game would; the session at the column's top says whether they're reading or editing and which recording takes their steps.

FIRST VIEWPORT: Top bar: the mark and MCM LEXICON at left; at right the profile select (or the catalog's file name), the scan state, Skyrim's state, and the theme switch. Left, the 18rem column as SkyUI's MCM lays out its left side, the menus starting as high as they can: SETTINGS on its own in full ink after a drawn cog, as Hermaeus's Config; the editing session (READ-ONLY / EDITING, the recording select with a chevron button to its page beside it, CHANGES while editing, which becomes an amber SAVE with REVIEW and its count beside it once the draft holds anything); SEARCH, RECORDINGS, and COMPARE on one row; then the menus under a filter that counts them, every menu by name taking the rest of the height, the one on screen marked with the diamond. Centre: the page; with no menu chosen, the list's summary. Without a list: the form, "Where is your mod list?" in display caps, the folder and profile fields, a primary Use this list, and an Open a catalog file instead link that unfolds the catalog file as the second way in.

FORM: The Game's Own Menus, inherited from Hermaeus rather than chosen anew; the brief pins it.

FINISH: looked at in both themes at 1440 and 1000 wide and stacked under 60rem, with and without a list, read-only and editing; the first-run form at full width.
