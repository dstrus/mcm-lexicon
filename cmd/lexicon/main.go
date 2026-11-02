// MCM Lexicon catalogs every Mod Configuration Menu option in a mod list and
// edits MCM Recorder recordings, with Skyrim closed. It starts a small web
// server on this machine and opens it in the browser.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"lexicon/internal/gamecheck"
	"lexicon/internal/modlist"
	"lexicon/internal/prefs"
	"lexicon/internal/server"
	"lexicon/web"
)

// version is set when building a release, from the git tag.
var version = "dev"

func main() {
	log.SetFlags(log.Ltime)
	// A command-line mode runs and exits instead of serving.
	if len(os.Args) > 1 {
		if run, ok := commands[os.Args[1]]; ok {
			run(os.Args[2:])
			return
		}
	}
	list := flag.String("list", "", "the mod list: a Mod Organizer 2 instance (the folder holding ModOrganizer.ini), or the game's Data folder")
	profile := flag.String("profile", "", "the Mod Organizer 2 profile to read (default: the one MO2 has selected)")
	port := flag.Int("port", 8766, "port to listen on; if it's taken, the next nine are tried, then any free port")
	noBrowser := flag.Bool("no-browser", false, "don't open MCM Lexicon in a browser")
	showVersion := flag.Bool("version", false, "print MCM Lexicon's version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("MCM Lexicon", version)
		return
	}

	remembered := prefs.Load(prefs.Path())
	saved := remembered.Get()
	var source *modlist.Source
	switch {
	case *list != "":
		s, err := modlist.Check(*list, *profile)
		if err != nil {
			exit("%v", err)
		}
		source = &s
	case saved.ListDir != "":
		// The list chosen last time, if it's still there.
		if s, err := modlist.Check(saved.ListDir, saved.Profile); err == nil {
			source = &s
		}
	default:
		if found := modlist.Find(); found != "" {
			if s, err := modlist.Check(found, ""); err == nil {
				source = &s
			}
		}
	}

	ports := make([]int, 10)
	for i := range ports {
		ports[i] = *port + i
	}
	ln, running, err := server.Listen(ports)
	if running != "" {
		fmt.Printf("MCM Lexicon is already running at %s, so this opens it instead of starting a second copy.\n", running)
		if *list != "" || *profile != "" {
			fmt.Println("To use different options, close that copy first, then start MCM Lexicon again.")
		}
		if !*noBrowser {
			openBrowser(running)
		}
		return
	}
	if err != nil {
		exit("Could not start MCM Lexicon: %v", err)
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port
	handler, err := server.New(server.Options{
		Source:  source,
		Catalog: saved.CatalogFile,
		Prefs:   remembered,
		Running: gamecheck.Running,
		Token:   sessionToken(),
		Port:    actualPort,
		Files:   web.Files(),
		Version: version,
	})
	if err != nil {
		exit("%v", err)
	}
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	url := fmt.Sprintf("http://127.0.0.1:%d/", actualPort)
	fmt.Println()
	if actualPort != *port {
		fmt.Printf("Port %d is in use by another program, so MCM Lexicon is using port %d.\n", *port, actualPort)
	}
	fmt.Printf("MCM Lexicon %s running at %s\n", version, url)
	switch {
	case source != nil && source.Kind == modlist.MO2:
		fmt.Printf("Mod list: %s (profile %s)\n", source.Path, source.Profile)
	case source != nil:
		fmt.Printf("Mod list: %s\n", source.Path)
	case saved.CatalogFile != "":
		fmt.Printf("Catalog:  %s\n", saved.CatalogFile)
	default:
		fmt.Println("Mod list: not found yet; choose it in the browser, or pass -list")
	}
	fmt.Println("Close this window or press Ctrl+C to stop.")
	fmt.Println()
	if !*noBrowser {
		openBrowser(url)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		exit("%v", err)
	}
}

// sessionToken is a new random token for each run. During development the
// Vite dev server has to send the same one, so LEXICON_DEV_TOKEN sets it.
func sessionToken() string {
	if t := os.Getenv("LEXICON_DEV_TOKEN"); t != "" {
		return t
	}
	return rand.Text()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		fmt.Printf("Open %s in your browser.\n", url)
	}
}

// exit reports a startup error. On Windows the console window closes as soon
// as the program ends when it was started by double-clicking, so wait for Enter.
func exit(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	if runtime.GOOS == "windows" {
		fmt.Fprint(os.Stderr, "Press Enter to close.")
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
	os.Exit(1)
}
