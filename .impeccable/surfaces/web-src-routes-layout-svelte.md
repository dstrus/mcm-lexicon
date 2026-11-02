---
version: 1
slug: "web-src-routes-layout-svelte"
primary_target: "web/src/routes/+layout.svelte"
related_targets: ["web/src/lib/source/SourceForm.svelte","web/src/routes/menus/+page.svelte"]
---

# App shell and first run

**Mode:** Operate. **Scope:** the shell (top bar, sections column, content area, hint strip) and the first-run form that chooses a mod list or a catalog file. Every page lives inside it.

**Audience and job:** the author at a desktop monitor between play sessions, pointing MCM Lexicon at a mod list once, then moving between Menus, Search, Recordings, and Compare. Which profile is read, whether Skyrim is running, and which recording is open are always visible.

**Ranges:** an MO2 instance has one to a handful of profiles; a catalog's file name can be long. States: no list chosen, a list open, a catalog open instead, Skyrim running (read-only), a section not built yet.

**Constraints:** Hermaeus's design system, inherited whole (DESIGN.md, PLAN.md §5.2): same tokens, fonts, shell grid, and breakpoints. Fonts bundled in the program. WCAG AA, visible focus, complete keyboard paths. Holds from 1280 to 1600 px; under 60rem the column becomes a scrolling row.

## Direction contract

THESIS: MCM Lexicon is one more menu of the game you just closed, and it is Hermaeus's shell with MCM Lexicon's sections in it: a categories column, a sortable item table, a details pane, and a button-hint strip backed by real keys. It refuses the admin-dashboard default of gray sidebar, cards, and a blue button.

OWN-WORLD: Smoked-glass ground (near-black in dark, pale smoke in light), hairline rules, off-white or near-black text, one accent only for selection and action: verdigris in dark, frost blue in light. Tracked condensed caps (Barlow Condensed) for the wordmark, sections, headings, and labels; Jost for text; DM Mono for what's matched character for character.

STORY: The visitor names their mod list (or opens a catalog) once, lands on Menus, and sees in the top bar which profile is read and whether Skyrim is running; the column takes them to Search, Recordings, and Compare as those are built.

FIRST VIEWPORT: Top bar: MCM LEXICON at left; at right the profile select (or the catalog's file name), Skyrim's state, and the theme switch. Left: the sections column, with unbuilt sections under a muted "Not built yet" label as plain text. Centre: the page. Without a list: the form, "Where is your mod list?" in display caps, the folder and profile fields, a primary Use this list, and under a hairline the catalog file as the second way in.

FORM: The Game's Own Menus, inherited from Hermaeus rather than chosen anew; the brief pins it.

FINISH: unreviewed and undocumented is unfinished; each phase ends with the pages looked at in both themes at 1280 to 1600 and stacked under 60rem (PLAN.md §5.6).
