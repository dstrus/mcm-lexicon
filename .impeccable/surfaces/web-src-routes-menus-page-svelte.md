---
version: 1
slug: "web-src-routes-menus-page-svelte"
primary_target: "web/src/routes/menus/+page.svelte"
related_targets: ["web/src/routes/menus/[id]/[[page]]/+page.svelte","web/src/routes/search/+page.svelte","web/src/lib/menus/OptionPane.svelte","web/src/lib/menus/Selector.svelte","web/src/lib/ui/QuickFind.svelte"]
---

# Menus, a menu, and Search

**Mode:** Operate. **Scope:** the catalog's three reading surfaces: every menu (Menus), one menu's pages and options (a menu), and options across every menu (Search), each an item table with the selected row in the details pane, plus Quick Find. Adding an option to a recording comes later and is not on these pages yet.

**Audience and job:** the author between play sessions, finding the exact label, page, and allowed values of an option, to write or fix a recording without launching Skyrim. The question is always "what exactly does a recording have to say", so the raw selector, its index among duplicates, and its allowed values are the payload.

**Ranges:** 12 menus in MGO today (about 70 once Papyrus menus are read), up to 12 pages a menu, up to about 140 options a menu, a few hundred search results at most (cut at 300). States: first scan running, scan failed, no menus, no match, a menu's page with nothing recordable, an inactive menu, duplicate labels, labels with spaces at an end, $KEY labels with and without a translation, saved values that differ from the default.

**Constraints:** Hermaeus's design system, inherited whole (DESIGN.md). Certainty and validation show as words, not color; copper only for a saved value that differs from the default; raw selectors in DM Mono beneath their text only when they differ or have an edge space. WCAG AA, complete keyboard paths with real hints.

## Direction contract

THESIS: The catalog reads like the game's own menus, one level up: the list of MCMs is a categories table, a menu's pages are tabs, and its options are an inventory table with a details pane that says exactly what a recording must hold.

OWN-WORLD: Inherited from Hermaeus: smoked glass, hairlines, tracked condensed caps for labels, Jost for reading, DM Mono for what's matched character for character, one accent for selection, action, and group headings.

STORY: The visitor opens Menus, picks a menu (or presses / to find one), steps through its pages as tabs, arrows down the options, and reads the selector, type, range, choices, default, and saved value in the pane; or searches every menu at once and opens the menu at the option found.

FIRST VIEWPORT: Menus: MENUS with All / Active and the count, where the list is, a line saying how many Papyrus menus aren't read yet, then the table (menu, mod, kind, pages, options) beside the pane. A menu: Up to Menus, its name, mod · plugin · kind, the page tabs with option counts, then the option table (option with raw selector and "1 of 2" for duplicates, type, allowed, default, saved) beside the option pane.

FORM: The Game's Own Menus, inherited; the brief pins it.

FINISH: looked at in both themes against the MGO snapshot, at 1440 wide; the narrow layout is Hermaeus's CSS, not yet looked at.
