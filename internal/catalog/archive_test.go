package catalog

import (
	"testing"

	"lexicon/internal/modlist"
	"lexicon/internal/testcorpus"
)

// Menus whose scripts are packed in archives (PLAN.md §2.6): one whose
// source ships loose beside its archive, as Extended Encounters does, and
// one with only its archive, translations included.
func TestArchivedScripts(t *testing.T) {
	inst := testcorpus.New(t, "MGO", "Default")
	src := "ScriptName EE_MCMScript extends SKI_ConfigBase\nEvent OnConfigInit()\n\tPages = new string[1]\n\tPages[0] = \"$EE_Main\"\nEndEvent\nEvent OnPageReset(string page)\n\tif page == Pages[0]\n\t\tAddToggleOption(\"Ambushes\", true)\n\tendif\nEndEvent\n"
	inst.Mod("Extended Encounters", true, map[string]string{
		"Extended Encounters.esp": testcorpus.Plugin(testcorpus.Quest{EditorID: "EE_MCM", Script: "EE_MCMScript", ModName: "Extended Encounters"}),
		"Extended Encounters.bsa": testcorpus.Archive(testcorpus.ArchiveOptions{}, map[string]string{
			"scripts/ee_mcmscript.pex": testcorpus.Compiled("EE_MCMScript", "SKI_ConfigBase", "$EE_Main", "Ambushes"),
		}),
		"Source/ee_mcmscript.psc": src,
	})
	inst.Mod("Accuracy - MCM menu", true, map[string]string{
		"Accuracy - MCM.esp": testcorpus.Plugin(testcorpus.Quest{EditorID: "ACMainQuest", Script: "ACMainQuestSettingsMCMScript", ModName: "$ACC_Name"}),
		"Accuracy - MCM.bsa": testcorpus.Archive(testcorpus.ArchiveOptions{Compressed: true}, map[string]string{
			"scripts/acmainquestsettingsmcmscript.pex":          testcorpus.Compiled("ACMainQuestSettingsMCMScript", "SKI_ConfigBase"),
			"interface/translations/accuracy - mcm_english.txt": utf16LE("$ACC_Name\tAccuracy\r\n"),
		}),
	})
	inst.Plugins("*Extended Encounters.esp", "*Accuracy - MCM.esp")
	s, err := modlist.Check(inst.Dir, "")
	if err != nil {
		t.Fatal(err)
	}
	l, err := modlist.Read(s)
	if err != nil {
		t.Fatal(err)
	}
	c := Scan(l, Options{})

	ee := menu(t, c, "Extended Encounters")
	if ee.Kind != KindSource || ee.Source.Path != "Source/ee_mcmscript.psc" || ee.Source.Archive != "" || ee.Mod != "Extended Encounters" {
		t.Errorf("Extended Encounters = %s from %+v in %s", ee.Kind, ee.Source, ee.Mod)
	}
	if len(ee.Pages) != 1 || len(ee.Pages[0].Options) != 1 || ee.Pages[0].Options[0].Label != "Ambushes" {
		t.Errorf("Extended Encounters pages = %+v", ee.Pages)
	}
	acc := menu(t, c, "Accuracy")
	if acc.Kind != KindCompiled || acc.Source.Archive != "Accuracy - MCM.bsa" || acc.Source.Path != "scripts/acmainquestsettingsmcmscript.pex" {
		t.Errorf("Accuracy = %s from %+v", acc.Kind, acc.Source)
	}
	if len(c.Skipped) != 0 {
		t.Errorf("skipped %+v", c.Skipped)
	}
}
