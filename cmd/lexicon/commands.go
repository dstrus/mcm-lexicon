package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lexicon/internal/catalog"
	"lexicon/internal/modlist"
	"lexicon/internal/snapshot"
)

// The command-line modes, which run and exit instead of serving:
// `lexicon scan` and `lexicon snapshot`. Without one, lexicon serves.
var commands = map[string]func(args []string){
	"scan":     runScan,
	"snapshot": runSnapshot,
	"diff":     runDiff,
}

// listFlags adds the flags that name a mod list to fs, and returns a
// function that reads it once the flags are parsed.
func listFlags(fs *flag.FlagSet) func() *modlist.List {
	list := fs.String("list", "", "the mod list: a Mod Organizer 2 instance, or the game's Data folder (default: the one next to the program)")
	profile := fs.String("profile", "", "the Mod Organizer 2 profile to read (default: the one MO2 has selected)")
	return func() *modlist.List {
		path := *list
		if path == "" {
			if path = modlist.Find(); path == "" {
				exit("No mod list next to the program; pass -list.")
			}
		}
		src, err := modlist.Check(path, *profile)
		if err != nil {
			exit("%v", err)
		}
		l, err := modlist.Read(src)
		if err != nil {
			exit("%v", err)
		}
		return l
	}
}

// runScan reads a list and prints its mods with the files the game sees
// from each, within what MCM Lexicon reads; or, with -json, writes the
// list's catalog, with a capture from the game added if -capture names one.
func runScan(args []string) {
	fs := flag.NewFlagSet("lexicon scan", flag.ExitOnError)
	read := listFlags(fs)
	all := fs.Bool("all", false, "list every enabled mod, not only those with files MCM Lexicon reads")
	asJSON := fs.Bool("json", false, "write the list's catalog as JSON, to -o or to standard output")
	out := fs.String("o", "", "with -json, the catalog file to write (a .json file)")
	language := fs.String("language", "ENGLISH", "the translations to read, as their file names spell the language")
	cache := fs.Bool("cache", true, "with -json, use and update the cache the app keeps of the list's scripts")
	capture := fs.String("capture", "", "with -json, a capture.json from the capture plugin to add, saying what it matched")
	fs.Parse(args)
	l := read()
	if *asJSON {
		o := catalog.Options{Language: *language, Version: version}
		if *cache {
			o.CacheDir = catalog.DefaultCacheDir()
		}
		c := catalog.Scan(l, o)
		if *capture != "" {
			f, err := catalog.ReadCapture(*capture)
			if err != nil {
				exit("%v", err)
			}
			c = catalog.ApplyCapture(c, f, *capture)
			fmt.Fprintf(os.Stderr, "Capture: %s\n", c.Capture.Report)
		}
		if *out == "" {
			b, err := catalog.Marshal(c)
			if err != nil {
				exit("%v", err)
			}
			os.Stdout.Write(b)
			return
		}
		if err := catalog.WriteFile(*out, c); err != nil {
			exit("%v", err)
		}
		fmt.Printf("Wrote %s: %d menus, %d not read\n", *out, len(c.Menus), len(c.Skipped))
		return
	}
	l.Index(modlist.MCMScope)

	fmt.Printf("%s", describe(l))
	counts := map[string]int{}
	for _, f := range l.Files("") {
		counts[f.Mod]++
	}
	var mods, withFiles, disabled, total int
	for i := len(l.Mods) - 1; i >= 0; i-- {
		m := l.Mods[i]
		if !m.Enabled {
			disabled++
			continue
		}
		mods++
		n := counts[m.Name]
		total += n
		if n > 0 {
			withFiles++
		}
		if n > 0 || *all {
			fmt.Printf("%5d  %-60s %5d\n", m.Priority, m.Name, n)
		}
	}
	fmt.Printf("\n%d enabled mods (%d disabled), %d with files MCM Lexicon reads, %d files the game sees\n", mods, disabled, withFiles, total)
}

// runSnapshot writes a snapshot of a list (PLAN.md §10).
func runSnapshot(args []string) {
	fs := flag.NewFlagSet("lexicon snapshot", flag.ExitOnError)
	read := listFlags(fs)
	languages := fs.String("languages", "ENGLISH", "translation languages to include, as their file names spell them, separated by commas")
	out := fs.String("o", "", "where to write the snapshot: a .zip, or a new folder (default: snapshot-<list>-<time>.zip here)")
	fs.Parse(args)
	l := read()
	if *out == "" {
		*out = fmt.Sprintf("snapshot-%s-%s.zip", filepath.Base(l.Source.Path), time.Now().Format("20060102-1504"))
	}
	var langs []string
	for _, lang := range strings.Split(*languages, ",") {
		if lang = strings.TrimSpace(lang); lang != "" {
			langs = append(langs, strings.ToUpper(lang))
		}
	}
	sum, err := snapshot.Write(l, *out, snapshot.Options{Languages: langs, Tool: "lexicon snapshot " + version})
	if err != nil {
		exit("%v", err)
	}
	fmt.Printf("Snapshot of %s (%s): %d mods, %d files, %d recorded steps\n", l.Source.Path, l.Source.Profile, sum.Mods, sum.Files, sum.Steps)
	fmt.Printf("Wrote %s (%.1f MB)\n", sum.Out, float64(sum.Bytes)/(1024*1024))
}

// runDiff compares two catalog files, older first.
func runDiff(args []string) {
	fs := flag.NewFlagSet("lexicon diff", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: lexicon diff old.json new.json")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if fs.NArg() != 2 {
		fs.Usage()
		exit("Name two catalog files, the older first.")
	}
	old, err := catalog.ReadFile(fs.Arg(0))
	if err != nil {
		exit("%v", err)
	}
	new, err := catalog.ReadFile(fs.Arg(1))
	if err != nil {
		exit("%v", err)
	}
	fmt.Print(catalog.Report(catalog.Compare(old, new)))
}

// describe is the list in a few lines: where it is and what was read.
func describe(l *modlist.List) string {
	var b strings.Builder
	switch l.Source.Kind {
	case modlist.MO2:
		fmt.Fprintf(&b, "Mod Organizer 2 instance: %s\nProfile: %s", l.Source.Path, l.Source.Profile)
		if l.Source.Game != "" {
			fmt.Fprintf(&b, " (%s)", l.Source.Game)
		}
		b.WriteString("\n")
		switch {
		case l.DataDir != "":
			fmt.Fprintf(&b, "Game Data folder: %s\n", l.DataDir)
		case l.DataPath != "":
			fmt.Fprintf(&b, "Game Data folder: %s (not here, so the game's own files aren't read)\n", l.DataPath)
		default:
			b.WriteString("Game Data folder: not named in ModOrganizer.ini\n")
		}
		if l.Plugins() == nil {
			b.WriteString("No plugins.txt in this profile, so every plugin counts as active\n")
		}
	default:
		fmt.Fprintf(&b, "Data folder: %s\n", l.Source.Path)
	}
	b.WriteString("\n")
	return b.String()
}
