---
version: 1
slug: "web-src-routes-menus-page-svelte"
primary_target: "web/src/routes/menus/+page.svelte"
related_targets: ["web/src/routes/menus/[id]/[[page]]/+page.svelte","web/src/routes/search/+page.svelte","web/src/lib/menus/OptionPane.svelte","web/src/lib/menus/Selector.svelte","web/src/lib/ui/QuickFind.svelte"]
---

# Menus, a menu, and Search

**Mode:** Operate. **Scope:** the catalog's three reading surfaces: the list's home (Menus), one menu drawn as SkyUI draws it (a menu), and options across every menu (Search), plus Quick Find; and, while editing with a recording chosen, the menu page as the place a recording is built, with the game's own grid as the form.

**Audience and job:** the author between play sessions, finding the exact label, page, and allowed values of an option, to write or fix a recording without launching Skyrim. The question is always "what exactly does a recording have to say", so the raw selector, its index among duplicates, and its allowed values are the payload.

**Ranges:** 80 menus in MGO, up to 12 pages a menu, up to about 140 options a menu, a few hundred search results at most (cut at 300). States: first scan running, scan failed, no menus, no match, a menu's page with nothing recordable, an inactive menu, a menu read with a guess (its plugin, its name, its pages) or from a compiled script, duplicate labels, labels with spaces at an end, $KEY labels with and without a translation, saved values that differ from the default, options grayed out or inert, steppers, and, while editing, options recorded, unsaved, or not yet recorded.

**Constraints:** Hermaeus's design system, inherited whole (DESIGN.md). Certainty and validation show as words, not color; copper only for a saved value that differs from the default; raw selectors in DM Mono beneath their text only when they differ or have an edge space. WCAG AA, complete keyboard paths with real hints.

## Direction contract

THESIS: The catalog reads like the game's own menus: the column lists the MCMs as SkyUI does, a menu is drawn on the game's own two-column grid with its page list down the left and its info text beneath, and Search is the one place the game never had, every option in every menu in one table with a details pane that says exactly what a recording must hold.

OWN-WORLD: Inherited from Hermaeus: smoked glass, hairlines, tracked condensed caps for labels, Jost for reading, DM Mono for what's matched character for character, one accent for selection, action, and group headings.

STORY: The visitor picks a menu from the column (or presses / to find one), steps through its pages, arrows across the grid as in the game, and reads the help, selector, type, allowed values, default, and saved value in the info area; while editing, changes a value in its cell and sees the diamond mark it as recorded. Or they search every menu at once and open the menu at the option found.

FIRST VIEWPORT: Menus: MENUS with the count, where the list is, one line inviting a choice from the column, and the scripts that aren't menus, folded. A menu: its name, mod · plugin · kind, any note on how it was read, the page list with option counts down the left, the grid of headers and options with values at the right edge, and the info area beneath. Search: SEARCH, one line, the box with the count beside it, the results grouped under their menus beside the option pane.

FORM: The Game's Own Menus, inherited; the brief pins it.

FINISH: looked at in both themes against the MGO snapshot at 1440 and 1000 wide and stacked under 60rem, read-only and editing, with Quick Find open; the info area stays in view on a tall page, and long paths break between folders.
