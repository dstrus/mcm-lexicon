---
version: 1
slug: "web-src-routes-compare-page-svelte"
primary_target: "web/src/routes/compare/+page.svelte"
related_targets: ["web/src/lib/recordings/steps.ts"]
---

# Compare

**Mode:** Operate. **Scope:** the list now against a catalog file saved before: which menus, pages, and options changed, and which recorded steps the changes break.

**Audience and job:** a list maintainer or the author after a list update, asking what changed in the MCMs and whether their recordings still play.

**Ranges:** a catalog of 60 to 80 menus; from no differences to a few hundred; the Changed list's rows as the doorways that filter the table. States: no file chosen, a file that isn't a catalog or was written by a newer version, no differences, differences with and without broken steps.

**Constraints:** Hermaeus's design system (DESIGN.md); the stat-list pane and the Numbers Are Doors Rule; validation words for broken steps. WCAG AA.

## Direction contract

THESIS: Compare reads like a changelog the list never wrote: each menu's additions, removals, and changed options in order, and under them the steps that no longer find their option.

OWN-WORLD: Inherited from Hermaeus.

STORY: The visitor names the older catalog file, presses COMPARE, reads the counts in the pane, clicks one to filter the table, and opens a changed menu where it still exists.

FIRST VIEWPORT: COMPARE, one sentence, the OLDER CATALOG FILE field with COMPARE beside it; after comparing, the table of changes beside the pane's Changed list.

FORM: The Game's Own Menus, inherited.

FINISH: looked at in both themes at 1440 with no file chosen; a comparison of MGO 4 RC 4.5 against an older MGO 4 catalog file (RC 4.2's, saved before its snapshot was removed) is the test case for the result.
