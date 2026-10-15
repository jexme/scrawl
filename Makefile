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

.PHONY: all gio build test clean

all: build

gio: third_party/gio/.screen-editor-patched

third_party/gio/.screen-editor-patched: third_party/gio.patch scripts/setup-gio.sh go.mod
	./scripts/setup-gio.sh -f

build: gio
	$(GO) build -o bin/screen-editor ./cmd/screen-editor

test: gio
	$(GO) test ./...

clean:
	rm -rf bin
