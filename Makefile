# Building MCM Lexicon needs Go and Node. Running it needs neither: the
# program it builds has the frontend embedded.

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GOBUILD = CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build windows release web notices types test dev icons clean

# The program, for this machine.
build: web
	$(GOBUILD) -o lexicon ./cmd/lexicon

# The program for 64-bit Windows, as MCMLexicon.exe, from any OS. Its icon and
# file details go into a .syso file that go build links into Windows builds.
windows: web
	go tool go-winres simply --arch amd64 --manifest cli --icon cmd/lexicon/lexicon.ico \
		--product-name "MCM Lexicon" --file-description "MCM Lexicon, a catalog of MCM options and an editor for MCM Recorder recordings" \
		--original-filename MCMLexicon.exe --out cmd/lexicon/rsrc
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o MCMLexicon.exe ./cmd/lexicon

# A release's downloads, into dist/: a zip for Windows, a zip for macOS (one
# program for both Apple silicon and Intel), and a tarball for Linux, each
# with LICENSE and THIRD_PARTY_NOTICES.txt, plus their SHA256SUMS. Their
# names carry no version, so the site can link to the latest release's.
# Needs macOS, for lipo and codesign; the Release workflow runs it when a
# version tag is pushed.
release: windows
	rm -rf dist && mkdir -p dist/windows dist/macos dist/linux
	cp MCMLexicon.exe dist/windows/
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o dist/macos/lexicon-arm64 ./cmd/lexicon
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o dist/macos/lexicon-amd64 ./cmd/lexicon
	lipo -create -output dist/macos/lexicon dist/macos/lexicon-arm64 dist/macos/lexicon-amd64
	rm dist/macos/lexicon-arm64 dist/macos/lexicon-amd64
	codesign --force --sign - dist/macos/lexicon
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o dist/linux/lexicon ./cmd/lexicon
	for os in windows macos linux; do cp LICENSE dist/$$os/LICENSE.txt; cp THIRD_PARTY_NOTICES.txt dist/$$os/; done
	cd dist/windows && zip -qX ../MCMLexicon-windows-x64.zip MCMLexicon.exe LICENSE.txt THIRD_PARTY_NOTICES.txt
	cd dist/macos && zip -qX ../MCMLexicon-macos.zip lexicon LICENSE.txt THIRD_PARTY_NOTICES.txt
	COPYFILE_DISABLE=1 tar -czf dist/MCMLexicon-linux-x64.tar.gz -C dist/linux lexicon LICENSE.txt THIRD_PARTY_NOTICES.txt
	cd dist && shasum -a 256 MCMLexicon-* > SHA256SUMS

# THIRD_PARTY_NOTICES.txt: the license of everything built into the program,
# from its Go modules and the npm packages in its pages, which the frontend's
# build records. Run it after changing dependencies; releases ship it beside
# LICENSE.
notices: web
	go run ./cmd/notices

# The frontend, into web/build, where the program embeds it from.
web: web/node_modules
	npm --prefix web run build

web/node_modules: web/package.json web/package-lock.json
	npm --prefix web ci --ignore-scripts
	@touch $@

# The icon files, from the drawings in assets/: the icon, and a simpler
# drawing for the 16-pixel size, where detail turns to mush.
# Needs ImageMagick; only run when a drawing changes, since the results are
# checked in.
icons:
	convert -background none -density 400 assets/lexicon-icon-small.svg -resize 16x16 /tmp/lexicon-16.png
	for n in 24 32 48 64 128 256; do convert -background none -density 400 assets/lexicon-icon.svg -resize $${n}x$${n} /tmp/lexicon-$$n.png; done
	convert /tmp/lexicon-16.png /tmp/lexicon-32.png /tmp/lexicon-48.png web/static/favicon.ico
	convert /tmp/lexicon-16.png /tmp/lexicon-24.png /tmp/lexicon-32.png /tmp/lexicon-48.png /tmp/lexicon-64.png /tmp/lexicon-128.png /tmp/lexicon-256.png cmd/lexicon/lexicon.ico
	convert -background none -density 400 assets/lexicon-icon.svg -resize 180x180 web/static/apple-touch-icon.png
	cp assets/lexicon-icon.svg web/static/favicon.svg
	rm -f /tmp/lexicon-*.png

# TypeScript types for the API, from the Go types listed in tygo.yaml.
types:
	go tool tygo generate

# Go tests (which don't need the frontend built), then type-check the
# frontend, including that it still matches the generated API types.
test: web/node_modules types
	go test ./internal/...
	npm --prefix web run check

# MCM Lexicon plus Vite's dev server (http://localhost:5174), which reloads
# the page as the frontend changes. MCM Lexicon gets its own port, so a copy
# already running on the usual 8766 doesn't take its place. Both get the same token,
# since the page Vite serves can't carry it. Pass a mod list with LIST=…
# If either one stops (including MCM Lexicon failing to start), both do.
DEV_PORT ?= 8776

dev: web/node_modules
	@test -d web/build || npm --prefix web run build
	@export LEXICON_DEV_TOKEN=$$(openssl rand -hex 16) LEXICON_DEV_PORT=$(DEV_PORT); \
	trap 'kill 0' EXIT; \
	(go run ./cmd/lexicon -no-browser -port $(DEV_PORT) $(if $(LIST),-list "$(LIST)"); kill 0) & \
	npm --prefix web run dev

clean:
	rm -rf lexicon MCMLexicon.exe dist cmd/lexicon/*.syso web/build web/.svelte-kit
