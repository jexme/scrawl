# Building scrawl. Gio is taken from the module cache and patched by the script
# scripts/setup-gio.sh; a copy of Gio is not part of the repository.
#
# Screen capture needs the PipeWire headers: either the system ones
# (sudo apt install libpipewire-0.3-dev), or ones unpacked without sudo by
# scripts/fetch-pipewire-headers.sh into .deps/pipewire-dev (picked up automatically).
GO ?= go

# cgo rejects the -fno-strict-overflow flag that libpipewire.pc adds on its own.
export CGO_CFLAGS_ALLOW := -fno-strict-overflow

PW_LOCAL := $(CURDIR)/.deps/pipewire-dev
ifneq ($(wildcard $(PW_LOCAL)/usr/lib/x86_64-linux-gnu/pkgconfig),)
export PKG_CONFIG_SYSROOT_DIR := $(PW_LOCAL)
export PKG_CONFIG_PATH := $(PW_LOCAL)/usr/lib/x86_64-linux-gnu/pkgconfig
endif

PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
APPDIR ?= $(PREFIX)/share/applications
DESKTOP := io.github.jexme.Scrawl.desktop

.PHONY: all gio build test notices install uninstall dist deb hooks clean

all: build

gio: third_party/gio/.scrawl-patched

third_party/gio/.scrawl-patched: third_party/gio.patch scripts/setup-gio.sh go.mod
	./scripts/setup-gio.sh -f

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null)
# Release targets (dist, deb) strip the symbol table and DWARF (-s -w): a smaller binary and
# package. `make build` keeps them for debugging.
STRIP :=
dist deb: STRIP := -s -w
LDFLAGS = $(if $(or $(STRIP),$(VERSION)),-ldflags "$(STRIP) $(if $(VERSION),-X github.com/jexme/scrawl/internal/version.Version=$(VERSION))")

build: gio
	$(GO) build $(LDFLAGS) -o bin/scrawl ./cmd/scrawl

test: gio
	$(GO) test ./...

# Regenerates THIRD_PARTY_NOTICES.md from the dependency licenses.
notices:
	python3 scripts/gen-notices.py

# Installs the binary and the .desktop file. An absolute path is put into Exec: the portal
# matches an application to a .desktop file only if Exec exists.
install: build
	install -Dm755 bin/scrawl $(DESTDIR)$(BINDIR)/scrawl
	install -d $(DESTDIR)$(APPDIR)
	sed 's|^Exec=scrawl|Exec=$(BINDIR)/scrawl|' packaging/$(DESKTOP) > $(DESTDIR)$(APPDIR)/$(DESKTOP)
	chmod 644 $(DESTDIR)$(APPDIR)/$(DESKTOP)
	@sh scripts/gnome-hotkey.sh Print $(BINDIR)/scrawl --auto; case $$? in 0) echo "Клавиша Print назначена на Scrawl.";; 12) echo "Print занята другим ярлыком: назначьте свою в scrawl -settings.";; esac
	@echo "Установлено в $(BINDIR). Дальше: scrawl -setup, затем scrawl -grant-screenshot"

# Removes the binary, the .desktop file, the GNOME shortcut and the autostart entry;
# `make uninstall PURGE=1` also deletes the settings and the cache.
uninstall:
	PREFIX="$(PREFIX)" BINDIR="$(BINDIR)" APPDIR="$(APPDIR)" DESTDIR="$(DESTDIR)" sh packaging/uninstall.sh $(if $(PURGE),--purge)

# Release archive: binary, .desktop, installer, licenses and README, plus SHA-256.
# The version is taken from git describe or from VERSION=v1.2.3.
DIST := dist
PKG = scrawl-$(VERSION)-linux-$(shell $(GO) env GOARCH)

dist: build
	rm -rf $(DIST)/$(PKG) $(DIST)/$(PKG).tar.gz $(DIST)/$(PKG).tar.gz.sha256
	mkdir -p $(DIST)/$(PKG)
	cp bin/scrawl packaging/$(DESKTOP) packaging/install.sh packaging/uninstall.sh scripts/gnome-hotkey.sh LICENSE THIRD_PARTY_NOTICES.md README.md docs/translations/README.ru.md $(DIST)/$(PKG)/
	tar -C $(DIST) --owner=0 --group=0 -czf $(DIST)/$(PKG).tar.gz $(PKG)
	cd $(DIST) && sha256sum $(PKG).tar.gz > $(PKG).tar.gz.sha256
	@echo "Готово: $(DIST)/$(PKG).tar.gz"

# Package for Debian and Ubuntu: dist/scrawl_VERSION_ARCH.deb.
deb: build
	packaging/deb.sh "$(VERSION)" "$(shell $(GO) env GOARCH)" $(DIST)

# Enables checks before commit and push: a personal email must not get into the history.
hooks:
	git config core.hooksPath .githooks
	@echo "Хуки включены: адрес автора проверяется перед коммитом и перед push."

clean:
	rm -rf bin $(DIST)
