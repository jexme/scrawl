# Сборка screen-editor. Gio берётся из модульного кэша и патчится скриптом
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
DESKTOP := io.github.jexme.ScreenEditor.desktop

.PHONY: all gio build test notices install uninstall clean

all: build

gio: third_party/gio/.screen-editor-patched

third_party/gio/.screen-editor-patched: third_party/gio.patch scripts/setup-gio.sh go.mod
	./scripts/setup-gio.sh -f

build: gio
	$(GO) build -o bin/screen-editor ./cmd/screen-editor

test: gio
	$(GO) test ./...

# Пересобирает THIRD_PARTY_NOTICES.md из лицензий зависимостей.
notices:
	python3 scripts/gen-notices.py

# Ставит бинарник и .desktop. В Exec подставляется абсолютный путь: portal
# сопоставляет приложение с .desktop-файлом только если Exec существует.
install: build
	install -Dm755 bin/screen-editor $(DESTDIR)$(BINDIR)/screen-editor
	install -d $(DESTDIR)$(APPDIR)
	sed 's|^Exec=screen-editor|Exec=$(BINDIR)/screen-editor|' packaging/$(DESKTOP) > $(DESTDIR)$(APPDIR)/$(DESKTOP)
	chmod 644 $(DESTDIR)$(APPDIR)/$(DESKTOP)
	@echo "Установлено в $(BINDIR). Дальше: screen-editor -setup, затем screen-editor -grant-screenshot"

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/screen-editor $(DESTDIR)$(APPDIR)/$(DESKTOP)

clean:
	rm -rf bin
