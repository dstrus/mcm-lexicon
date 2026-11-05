---
version: 1
slug: "web-src-routes-settings-page-svelte"
primary_target: "web/src/routes/settings/+page.svelte"
related_targets: ["web/src/routes/+layout.svelte","web/src/lib/api/status.ts"]
---

# Settings

**Mode:** Operate. **Scope:** MCM Lexicon's own settings: the mod list it reads (an MO2 instance and profile, a Data folder, or a catalog file), and the mod folder recordings save to in an MO2 instance, with whether "[NoDelete] " goes before its name.

**Audience and job:** the author pointing MCM Lexicon at another list or a catalog, or naming the folder their recordings live in; a Wabbajack user, keeping it through a list update.

**Ranges:** one short name. States: the default, a name of the author's own, the prefix off, a name that can't be a folder's (refused, with the reason), a Data folder or a catalog file open (the setting applies to an MO2 instance).

**Constraints:** Hermaeus's design system (DESIGN.md): a field and a Switch as on a recording's Info tab, kept as they're changed, editing or not, since they change no recording. WCAG AA.

## Direction contract

THESIS: Where MCM Lexicon reads from and where it writes to, as two groups: MOD LIST, whose change is a deliberate form (it rescans and can discard the draft), and SAVE FOLDER, kept as it's changed.

OWN-WORLD: Inherited from Hermaeus.

STORY: The visitor types a name or flips the prefix, and the sentence beneath says the folder that results; notes say what happens to recordings from other mods and to ones already saved.

FIRST VIEWPORT: SETTINGS; MOD LIST with the list in use in a sentence, CHANGE LIST…, and a note; under a hairline, SAVE FOLDER, the name field, the prefix Switch with its note, the resulting folder in a sentence, two notes. CHANGE LIST… opens the list form in place, with BROWSE…'s folder picker.

FORM: The Game's Own Menus, inherited.

FINISH: looked at in both themes at 1440 and 1000 wide and stacked under 60rem.
