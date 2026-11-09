# Developing MCM Lexicon

For using it, see the [README](README.md).

## Building

You need Go 1.26 or later and Node 22.17 or later.

```sh
make build              # the frontend, then the program, as ./lexicon
make windows            # the program for 64-bit Windows, in its own WebView2 window, as ./MCMLexicon.exe
make windows-console    # the same as a console program that opens the browser, as ./MCMLexicon-console.exe
make release            # the Windows download, into dist/
make release-notes VERSION=v0.2.0   # that version's release notes, from CHANGELOG.md, into dist/
make notices            # THIRD_PARTY_NOTICES.txt, from the licenses of everything built into the program
make test               # Go tests, then type-checking the frontend
make dev LIST=/path/to/instance   # MCM Lexicon plus Vite's dev server at http://localhost:5174
make types              # regenerate the API's TypeScript types after changing them in Go
make icons              # the icon files, from the drawings in assets/
```

`go test ./internal/...` works without Node. Building the program, or `go vet ./...`, needs the frontend built first, since it's embedded from `web/build`. The tests that read real mod lists run only when `snapshots/` holds some (or `LEXICON_SNAPSHOTS` points at them); without any they're skipped, not failed.

To release a version, add its section to [CHANGELOG.md](CHANGELOG.md) (`make release-notes VERSION=v0.2.0` shows the notes it makes), then push a tag such as `v0.2.0`. The Release workflow tests, builds the Windows download and the capture plugin's, and attaches them to a draft release with those notes, which goes live when you publish it.

### The capture plugin

`skse/` is `MCMLexicon.dll`, the SKSE plugin that captures what each MCM shows in a running game ([PLAN.md](PLAN.md) §11). It's C++ on CommonLibSSE NG. Builds you install or release are made on Windows, with Visual Studio 2022's C++ tools, CMake, Ninja, and vcpkg:

```powershell
cd skse
$env:VCPKG_ROOT = 'C:\path\to\vcpkg'
cmake --preset release     # from a Developer PowerShell, so cl is found
cmake --build --preset release   # skse/build/MCMLexicon.dll
```

On macOS or Linux, `cmake --preset cross` checks that it compiles, with clang-cl and lld-link against a Windows SDK and C runtime downloaded by [xwin](https://github.com/Jake-Shadle/xwin). The DLL it makes is for checking only: clang-cl doesn't promise MSVC's layout for the game's classes, so what you install or release comes from MSVC. Set up once (on macOS; on Linux, your distribution's `clang`, `lld`, `cmake`, `ninja`, and `rustup`):

```sh
brew install llvm lld cmake ninja rustup
/opt/homebrew/opt/rustup/bin/rustup default stable
cargo install xwin --locked     # ~/.cargo/bin/xwin
xwin --accept-license --cache-dir ~/xwin-cache splat --output ~/xwin-out \
  --include-debug-libs --use-winsysroot-style --preserve-ms-arch-notation   # accepts Microsoft's license for the SDK and runtime
git clone https://github.com/microsoft/vcpkg ~/.local/share/vcpkg && ~/.local/share/vcpkg/bootstrap-vcpkg.sh -disableMetrics
```

Then, from `skse/`:

```sh
export VCPKG_ROOT=~/.local/share/vcpkg   # and XWIN_SYSROOT, if the SDK isn't in ~/xwin-out
cmake --preset cross && cmake --build --preset cross   # skse/build-cross/MCMLexicon.dll
```

The first build compiles CommonLibSSE NG from source, about 15 minutes; after that, a change to the plugin builds in seconds. `skse/cross/` holds the toolchain, vcpkg's triplet for it, and a stand-in for vcpkg's `directxtk` that skips its shaders, which need the Windows SDK's `fxc.exe`.

The Capture plugin workflow builds it on every push to `main` that changes `skse/` (or by hand, from the Actions tab) and keeps `MCMLexicon-Capture.zip` as the run's artifact. The DLL as distributed is GPL-3.0-or-later, since CommonLibSSE NG is; see [skse/LICENSE.md](skse/LICENSE.md).

## Layout

| Path | Purpose |
| --- | --- |
| `cmd/lexicon/` | Flags, startup, opening the browser, and the `scan`, `diff`, and `snapshot` commands |
| `internal/server/` | The web server: host and token checks, serving the frontend, mounting each domain's routes |
| `internal/httpapi/` | Turning typed Go functions into JSON handlers |
| `internal/apperr/` | Errors meant for the user, by kind; the server maps kinds to HTTP statuses |
| `internal/modlist/` | Finding and reading a mod list: an MO2 instance's profile and mod order, or a `Data` folder, as the virtual file order the game sees |
| `internal/bsa/` | Reading Bethesda archives, so MCMs packed in them are read too |
| `internal/plugin/` | Reading plugins as far as their quests and the script properties attached to them |
| `internal/papyrus/` | Papyrus source (`source`) and compiled scripts (`pex`) lowered into one intermediate form (`ir`), and the analysis that reads a menu out of a script (`mcm`) |
| `internal/mcmhelper/` | MCM Helper's `config.json` menus, and the settings files that hold their saved values |
| `internal/translations/` | The game's translation files, for the text behind `$KEY` labels |
| `internal/catalog/` | What MCM Lexicon knows about a list: the scan that builds it, its cache, search, catalog files, and comparing two |
| `internal/recordings/` | MCM Recorder's recordings: reading, checking against the catalog, the draft, saving with backups, and restoring |
| `internal/snapshot/` | Copying everything MCM Lexicon reads from a list, for testing |
| `internal/prefs/` | MCM Lexicon's own remembered settings |
| `internal/gamecheck/` | Whether Skyrim is running, per OS |
| `internal/textfile/` | How files the game reads are changed: atomic writes, backups, line endings kept |
| `internal/testcorpus/` | Small mod lists built in temporary folders, for tests, and the snapshots when there are any |
| `assets/` | The drawings behind the app icon and the mark in the top bar; `make icons` renders the icon files from them |
| `web/src/routes/` | The app's pages; each folder is a URL |
| `web/src/lib/copy/` | The words the app shows, in `en.yaml`, grouped by page; `web/copy.ts` turns them into a typed module (`messages.ts`, generated) as the frontend builds |
| `web/src/lib/ui/` | The look and its shared parts: theme tokens (`theme.css`), the keyboard-shortcut hint strip, sortable heads, row selection, the diff |
| `web/src/lib/<domain>/` | Each domain's API calls and components, mirroring `internal/<domain>/` |
| `web/src/lib/api/` | The API client, and TypeScript types generated from Go (`types/`, don't edit) |
| `tygo.yaml` | Which Go types become TypeScript types |
| `skse/` | The capture plugin, `MCMLexicon.dll`: C++ on CommonLibSSE NG, built by its own workflow |

Domain packages own their logic, request and response types, routes, and tests. Only `textfile` writes files the game reads. The visual system is recorded in [DESIGN.md](DESIGN.md), and the plan it was built from in [PLAN.md](PLAN.md).
