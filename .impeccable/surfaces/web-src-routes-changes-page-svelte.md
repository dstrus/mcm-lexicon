---
version: 1
slug: "web-src-routes-changes-page-svelte"
primary_target: "web/src/routes/changes/+page.svelte"
related_targets: ["web/src/routes/+layout.svelte","web/src/lib/ui/LineDiff.svelte"]
---

# Changes

**Mode:** Operate. **Scope:** the draft as a whole, while editing: every recording it changes, deletes, creates, or restores, each with the files Save writes and where, and SAVE ALL and DISCARD ALL.

**Audience and job:** the author about to write to disk, reading exactly what will be written before it is. This is the Draft Rule's proof: nothing is written until it's been seen here or in a recording's Files tab.

**Ranges:** one to a handful of recordings in a draft; a recording's files from 2 to 15, long step files folded to their changed lines. States: nothing to save, a new recording (every file new), a deletion (every file removed), a backup being put back, a save that failed, Skyrim running (Save refused, with the reason).

**Constraints:** Hermaeus's design system (DESIGN.md). The File font and the line diff; amber for Save; the hint strip carries ⌘⏎ and ⌘Z. WCAG AA.

## Direction contract

THESIS: Changes is the receipt before the purchase: one page that says, recording by recording, what Save writes, removes, and backs up, in the words a file reader would use.

OWN-WORLD: Inherited from Hermaeus.

STORY: The visitor arrives from SAVE in the column or REVIEW on a recording, reads the sentence that sums the draft up, scans each recording's files, and presses SAVE ALL, or discards one recording's changes, or all.

FIRST VIEWPORT: CHANGES, the sentence ("Save writes or removes N files for M recordings, backing up the current ones first…"), SAVE ALL and DISCARD ALL, then the first recording with its word (CHANGED, NEW, DELETED, RESTORED), where it's written, its notices, and its files as diffs.

FORM: The Game's Own Menus, inherited.

FINISH: looked at in both themes at 1440 and under 60rem with a draft of one changed recording from another mod.
