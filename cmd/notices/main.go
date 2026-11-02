// Command notices writes THIRD_PARTY_NOTICES.txt: the license of everything
// built into MCM Lexicon, which releases ship beside LICENSE. That's the Go
// standard library, the Go modules compiled into the program for each OS it
// ships for, and the npm packages in its pages, as web/third-party.ts
// records them during the frontend's build. Run it with `make notices`,
// which builds the frontend first.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	out      = "THIRD_PARTY_NOTICES.txt"
	webList  = "web/.svelte-kit/third-party.json"
	program  = "./cmd/lexicon"
	ruleLine = "================================================================================"
)

// The operating systems MCM Lexicon ships for; each pulls in some code of its own.
var targets = []string{"windows", "darwin", "linux"}

// A module's license files: its license, any for code it carries from
// elsewhere (LICENSE-SQLITE), a NOTICE that Apache's license asks to travel
// with it, and the AUTHORS a license credits.
var licenseFile = regexp.MustCompile(`(?i)^(licen[cs]e|copying|notice|authors)(\.|-|$)`)

type notice struct {
	name, version, license, text string
}

func main() {
	log.SetFlags(0)
	var notices []notice

	goVersion := strings.TrimSpace(run(nil, "go", "env", "GOVERSION"))
	goRoot := strings.TrimSpace(run(nil, "go", "env", "GOROOT"))
	// Go keeps its license in GOROOT; Homebrew's Go keeps it a folder above.
	std, err := os.ReadFile(filepath.Join(goRoot, "LICENSE"))
	if err != nil {
		std, err = os.ReadFile(filepath.Join(goRoot, "..", "LICENSE"))
	}
	if err != nil {
		log.Fatalf("Go's own license: %v", err)
	}
	notices = append(notices, notice{"The Go standard library", goVersion, "BSD-3-Clause", string(std)})

	mods, err := goModules()
	if err != nil {
		log.Fatal(err)
	}
	notices = append(notices, mods...)

	web, err := webPackages()
	if err != nil {
		log.Fatal(err)
	}
	notices = append(notices, web...)

	if err := os.WriteFile(out, []byte(render(notices)), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Wrote %s: %d notices\n", out, len(notices))
}

// goModules lists the modules compiled into the program for any target,
// each with its license file.
func goModules() ([]notice, error) {
	type module struct{ Path, Version, Dir string }
	found := map[string]module{}
	for _, goos := range targets {
		env := append(os.Environ(), "GOOS="+goos, "GOARCH=amd64")
		dec := json.NewDecoder(strings.NewReader(run(env, "go", "list", "-deps", "-json", program)))
		for dec.More() {
			var pkg struct {
				Standard bool
				Module   *module
			}
			if err := dec.Decode(&pkg); err != nil {
				return nil, err
			}
			if pkg.Standard || pkg.Module == nil || pkg.Module.Path == "lexicon" {
				continue
			}
			found[pkg.Module.Path] = *pkg.Module
		}
	}
	var out []notice
	for _, m := range found {
		text, err := licenseIn(m.Dir)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", m.Path, err)
		}
		out = append(out, notice{name: m.Path, version: m.Version, text: text})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

func licenseIn(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && licenseFile.MatchString(e.Name()) {
			files = append(files, e.Name())
		}
	}
	// The license itself first, then what goes with it.
	sort.SliceStable(files, func(i, j int) bool { return rank(files[i]) < rank(files[j]) })
	var b strings.Builder
	for i, f := range files {
		text, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return "", err
		}
		if len(files) > 1 {
			if i > 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "--- %s ---\n\n", f)
		}
		b.WriteString(strings.TrimRight(strings.ReplaceAll(string(text), "\r\n", "\n"), "\n") + "\n")
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("no license file in %s", dir)
	}
	return b.String(), nil
}

func rank(file string) int {
	switch strings.ToUpper(file) {
	case "LICENSE", "LICENSE.TXT", "LICENSE.MD", "COPYING":
		return 0
	}
	return 1
}

// webPackages reads the packages the frontend's build recorded.
func webPackages() ([]notice, error) {
	b, err := os.ReadFile(webList)
	if err != nil {
		return nil, fmt.Errorf("%v (build the frontend first: make web)", err)
	}
	var pkgs []struct{ Name, Version, License, Text string }
	if err := json.Unmarshal(b, &pkgs); err != nil {
		return nil, err
	}
	var out []notice
	for _, p := range pkgs {
		if strings.TrimSpace(p.Text) == "" {
			return nil, fmt.Errorf("%s has no license file", p.Name)
		}
		out = append(out, notice{p.Name, p.Version, p.License, p.Text})
	}
	return out, nil
}

func render(notices []notice) string {
	var b strings.Builder
	b.WriteString("Third-party software in MCM Lexicon\n\n")
	b.WriteString("MCM Lexicon is released under the MIT License (see LICENSE). It includes the\n")
	b.WriteString("software below, each under its own license, reproduced here as its authors\n")
	b.WriteString("distribute it. The fonts are licensed under the SIL Open Font License 1.1.\n\n")
	for _, n := range notices {
		line := n.name
		if n.version != "" {
			line += " " + n.version
		}
		if n.license != "" {
			line += " (" + n.license + ")"
		}
		fmt.Fprintf(&b, "  %s\n", line)
	}
	for _, n := range notices {
		fmt.Fprintf(&b, "\n%s\n%s %s\n%s\n\n", ruleLine, n.name, n.version, ruleLine)
		text := strings.ReplaceAll(n.text, "\r\n", "\n")
		b.WriteString(strings.TrimRight(text, "\n") + "\n")
	}
	return b.String()
}

func run(env []string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		log.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, stderr.String())
	}
	return string(b)
}
