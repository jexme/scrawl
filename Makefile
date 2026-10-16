# Сборка scrawl. Gio берётся из модульного кэша и патчится скриптом
# scripts/setup-gio.sh, копия Gio в репозиторий не входит.
#
# Для захвата экрана нужны заголовки PipeWire: либо системные
# (sudo apt install libpipewire-0.3-dev), либо распакованные без sudo командой
# scripts/fetch-pipewire-headers.sh в .deps/pipewire-dev (подхватываются сами).
GO ?= go

# cgo отвергает флаг -fno-strict-overflow, который libpipewire.pc добавляет сам.
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

.PHONY: all gio build test notices install uninstall clean

all: build

gio: third_party/gio/.scrawl-patched

third_party/gio/.scrawl-patched: third_party/gio.patch scripts/setup-gio.sh go.mod
	./scripts/setup-gio.sh -f

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null)
LDFLAGS := $(if $(VERSION),-ldflags "-X github.com/jexme/scrawl/internal/version.Version=$(VERSION)")

build: gio
	$(GO) build $(LDFLAGS) -o bin/scrawl ./cmd/scrawl

test: gio
	$(GO) test ./...

# Пересобирает THIRD_PARTY_NOTICES.md из лицензий зависимостей.
notices:
	python3 scripts/gen-notices.py

# Ставит бинарник и .desktop. В Exec подставляется абсолютный путь: portal
# сопоставляет приложение с .desktop-файлом только если Exec существует.
install: build
	install -Dm755 bin/scrawl $(DESTDIR)$(BINDIR)/scrawl
	install -d $(DESTDIR)$(APPDIR)
	sed 's|^Exec=scrawl|Exec=$(BINDIR)/scrawl|' packaging/$(DESKTOP) > $(DESTDIR)$(APPDIR)/$(DESKTOP)
	chmod 644 $(DESTDIR)$(APPDIR)/$(DESKTOP)
	@echo "Установлено в $(BINDIR). Дальше: scrawl -setup, затем scrawl -grant-screenshot"

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/scrawl $(DESTDIR)$(APPDIR)/$(DESKTOP)

clean:
	rm -rf bin
