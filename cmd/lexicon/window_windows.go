//go:build window

// MCM Lexicon in a window of its own, on Windows: the same pages, shown in
// WebView2 (Edge's engine, part of Windows 10 and 11) instead of the
// browser. Built with -tags window and linked as a GUI program, so no
// console window opens; closing the window stops MCM Lexicon.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"

	"lexicon/internal/server"
)

const windowed = true

// The window's title, which a second copy finds it by, and the window
// class go-webview2 registers.
const windowTitle = "MCM Lexicon"

var windowClass = windows.StringToUTF16Ptr("webview")

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	shell32          = windows.NewLazySystemDLL("shell32.dll")
	findWindow       = user32.NewProc("FindWindowW")
	showWindow       = user32.NewProc("ShowWindow")
	isIconic         = user32.NewProc("IsIconic")
	setForeground    = user32.NewProc("SetForegroundWindow")
	setWindowLongPtr = user32.NewProc("SetWindowLongPtrW")
	callWindowProc   = user32.NewProc("CallWindowProcW")
	sendMessage      = user32.NewProc("SendMessageW")
	systemMetrics    = user32.NewProc("GetSystemMetrics")
	attachConsole    = kernel32.NewProc("AttachConsole")
	extractIconEx    = shell32.NewProc("ExtractIconExW")
)

const (
	gwlpWndProc         = ^uintptr(3) // GWLP_WNDPROC, -4
	wmClose             = 0x0010
	wmSetIcon           = 0x0080
	swRestore           = 9
	smCxScreen          = 0
	smCyScreen          = 1
	attachParentProcess = ^uint32(0) // ATTACH_PARENT_PROCESS, -1
	idYes               = 6
)

// hasConsole is whether there's a console to write to: a terminal this
// copy was started from, or output sent to a file or a pipe.
var hasConsole bool

// The window runs on the thread that made it, which is the main one.
func init() {
	runtime.LockOSThread()
	connectConsole()
}

// connectConsole gives a GUI program's output somewhere to go. Output sent
// to a file or a pipe already has a handle; from a terminal, it's the
// terminal's console. The terminal doesn't wait for a GUI program, so a
// command's output comes after the prompt; `MCMLexicon.exe scan | more`
// waits for it.
func connectConsole() {
	if h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); err == nil && h != 0 && h != windows.InvalidHandle {
		hasConsole = true
		return
	}
	if r, _, _ := attachConsole.Call(uintptr(attachParentProcess)); r == 0 {
		return // started from Explorer: no console, and none wanted
	}
	if out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout, os.Stderr = out, out
		hasConsole = true
	}
	if in, err := os.Open("CONIN$"); err == nil {
		os.Stdin = in
	}
}

// showError reports a startup error in a message box when there's no
// console to report it in, and says whether it did.
func showError(msg string) bool {
	if hasConsole {
		return false
	}
	box(0, msg, windows.MB_OK|windows.MB_ICONERROR)
	return true
}

func box(hwnd uintptr, text string, kind uint32) int32 {
	r, _ := windows.MessageBox(windows.HWND(hwnd), windows.StringToUTF16Ptr(text), windows.StringToUTF16Ptr(windowTitle), kind)
	return r
}

// focusWindow brings a running copy's window to the front, for a second
// copy started while it runs, and says whether there was one.
func focusWindow() bool {
	hwnd, _, _ := findWindow.Call(uintptr(unsafe.Pointer(windowClass)), uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(windowTitle))))
	if hwnd == 0 {
		return false
	}
	if iconic, _, _ := isIconic.Call(hwnd); iconic != 0 {
		showWindow.Call(hwnd, swRestore)
	}
	setForeground.Call(hwnd)
	return true
}

// runWindow shows MCM Lexicon at url in its window, and returns when the
// window is closed. Without WebView2 it opens the browser instead, and a
// message box stands in for the window: closing it stops MCM Lexicon.
func runWindow(address, token string) {
	data := ""
	if dir, err := os.UserCacheDir(); err == nil {
		// WebView2 keeps its profile here, not beside the program, which is
		// in the mod list's folder.
		data = filepath.Join(dir, "MCM Lexicon", "WebView2")
	}
	sw, _, _ := systemMetrics.Call(smCxScreen)
	sh, _, _ := systemMetrics.Call(smCyScreen)
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		DataPath:  data,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  windowTitle,
			Width:  uint(min(1600, sw*4/5)),
			Height: uint(min(1000, sh*17/20)),
			Center: true,
		},
	})
	if w == nil {
		openBrowser(address)
		box(0, fmt.Sprintf("MCM Lexicon is open in your browser at %s, because this PC doesn't have Microsoft Edge WebView2.\n\nClick OK to stop MCM Lexicon.", address), windows.MB_OK|windows.MB_ICONINFORMATION)
		return
	}
	defer w.Destroy()
	hwnd := uintptr(w.Window())
	setIcon(hwnd)
	askBeforeClosing(hwnd, address, token)

	// A link to another site opens in the browser, not in the window.
	w.Bind("lexiconOpen", func(link string) {
		if u, err := url.Parse(link); err == nil && (u.Scheme == "https" || u.Scheme == "http") {
			openBrowser(u.String())
		}
	})
	w.Init(windowScript)
	w.Navigate(address)
	w.Run()
}

// windowScript runs in every page before the app's own scripts. The
// browser's own keys mean nothing in this window, so they're kept from it:
// reload, print, find, and save. The app still sees them (a key box can
// take F5). Links to other sites go to the browser.
const windowScript = `(() => {
	addEventListener('keydown', (e) => {
		const k = e.key.toLowerCase();
		if (e.key === 'F5' || e.key === 'F3' || e.key === 'F7' || (e.ctrlKey && ['r', 'p', 'f', 'g', 's', 'u', 'o', 'n'].includes(k))) e.preventDefault();
	}, true);
	const away = (href) => {
		try {
			const u = new URL(href, location.href);
			if (u.origin === location.origin) return false;
			window.lexiconOpen(u.href);
			return true;
		} catch {
			return false;
		}
	};
	addEventListener('click', (e) => {
		const a = e.target instanceof Element && e.target.closest('a[href]');
		if (a && away(a.href)) e.preventDefault();
	}, true);
	window.open = (href) => {
		if (href) away(String(href));
		return null;
	};
})();`

// setIcon gives the window the program's own icon, for its title bar and
// the taskbar: the first in the .exe, whatever its resource ID.
func setIcon(hwnd uintptr) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	var large, small uintptr
	if n, _, _ := extractIconEx.Call(uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(exe))), 0, uintptr(unsafe.Pointer(&large)), uintptr(unsafe.Pointer(&small)), 1); n == 0 {
		return
	}
	if large != 0 {
		sendMessage.Call(hwnd, wmSetIcon, 1, large) // ICON_BIG
	}
	if small != 0 {
		sendMessage.Call(hwnd, wmSetIcon, 0, small) // ICON_SMALL
	}
}

// askBeforeClosing asks before the window closes with changes not saved:
// the draft lives only while MCM Lexicon runs, so closing loses it.
func askBeforeClosing(hwnd uintptr, address, token string) {
	var previous uintptr
	proc := windows.NewCallback(func(h, msg, wp, lp uintptr) uintptr {
		if msg == wmClose {
			if n := unsaved(address, token); n > 0 {
				text := fmt.Sprintf("%d recordings have changes that aren't saved yet. Close MCM Lexicon and lose them?", n)
				if n == 1 {
					text = "A recording has changes that aren't saved yet. Close MCM Lexicon and lose them?"
				}
				if box(h, text, windows.MB_YESNO|windows.MB_ICONWARNING|windows.MB_DEFBUTTON2) != idYes {
					return 0
				}
			}
		}
		r, _, _ := callWindowProc.Call(previous, h, msg, wp, lp)
		return r
	})
	previous, _, _ = setWindowLongPtr.Call(hwnd, gwlpWndProc, proc)
}

// unsaved is how many recordings the draft changes, asked of this copy's
// own server; 0 when it can't say, so the window can always close.
func unsaved(address, token string) int {
	req, err := http.NewRequest("GET", address+"api/status", nil)
	if err != nil {
		return 0
	}
	req.Header.Set(server.TokenHeader, token)
	res, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err != nil {
		return 0
	}
	defer res.Body.Close()
	var s server.Status
	if json.NewDecoder(res.Body).Decode(&s) != nil {
		return 0
	}
	return s.Changes
}
