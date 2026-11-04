---
version: 1
slug: "web-src-routes-recordings-page-svelte"
primary_target: "web/src/routes/recordings/+page.svelte"
related_targets: ["web/src/routes/recordings/[name]/+layout.svelte","web/src/routes/recordings/[name]/+page.svelte","web/src/routes/recordings/[name]/files/+page.svelte","web/src/routes/recordings/[name]/info/+page.svelte","web/src/routes/recordings/[name]/backups/+page.svelte","web/src/lib/ui/LineDiff.svelte","web/src/lib/ui/CodeArea.svelte"]
---

# Recordings and a recording

**Mode:** Operate. **Scope:** the list of MCM Recorder recordings the game sees (Recordings), and one recording: its header (where it is, what Save would write, and where), and its tabs: Steps, Files, Info, and Backups.

**Audience and job:** the author between play sessions, checking that a recording still matches the mods installed, changing a value, moving or removing a step, pulling another recording's steps into theirs, typing a file by hand when the table can't say it, or putting a backup back. Every change goes to one draft; nothing is written until Save.

**Ranges:** 6 recordings in the author's list, 4 to 20 steps each, 1 to 13 step files; a recording from another mod, one from the save folder, and new ones not on disk yet. States: read-only, editing with and without an open recording, a draft with changes, a deletion in the draft, a backup chosen to restore, a recording whose files are covered by empty ones, a recording with no backups, a recording from another mod untouched (nothing to save) or changed (saves to the save folder).

**Constraints:** Hermaeus's design system, inherited whole (DESIGN.md). The File font for files; a diff marked as Hermaeus marks an override against its base; copper only for a value that differs from a known default; validation as words, not color. WCAG AA, complete keyboard paths with real hints: arrows, E, U, D, Backspace, Esc, ⌘⏎ to save, ⌘Z to undo.

## Direction contract

THESIS: A recording reads as the game will play it: its steps in replay order under their file headings, each checked against the catalog in words, and the files exactly as Save writes them, with the draft's changes always one line away from being understood.

OWN-WORLD: Inherited from Hermaeus: smoked glass, hairlines, tracked condensed caps, Jost for reading, DM Mono for files and selectors, one accent for selection and action, amber only for Save and warnings.

STORY: The visitor opens Recordings, picks one, reads its check in the pane, opens it, changes a value in the table or the pane, sees the header say what Save will write and where, reviews the files, and saves from the column or the Changes page; or finds it wrong and discards.

FIRST VIEWPORT: Recordings: RECORDINGS with the count and a filter, the table (recording with its title beneath, where it is, steps, menus, autorun, issues) beside the pane with author, version, where, the info file, files, steps, its check in a sentence, and OPEN (DELETE and NEW while editing). A recording: Up, its name, title · where it is · steps in files, the draft line with REVIEW and DISCARD, SAVE TO, then the tabs; Steps as a table beside the step pane.

FORM: The Game's Own Menus, inherited; the brief pins it.

FINISH: looked at in both themes at 1440 and 1000 wide and stacked under 60rem, against the MGO snapshot, with a draft on a recording from another mod: the Files tab reads it against that mod's copy, so the diff shows the change and not the file's spelling.
