# -----------------------------------------------------------------------------
# Projekt:      CCU-ModbusClient
# Datei:        Makefile
# Ersteller:    Wolfram Henkel
# Datum:        05. Nov. 2026
# Version:      0.3.0
#
# Copyright (c) 2026 Wolfram Henkel.
#
# Lizenz:       CCU-ModbusClient-Lizenz für kostenfreie Nutzung und
#               Weitergabe, Version 1.1.
#
# Private und gewerbliche Nutzung sind erlaubt.
# Die Software und Änderungen dürfen nur kostenlos unter
# derselben Lizenz und mit vollständigem Quellcode weitergegeben werden.
#
# Vollständiger Lizenztext: LICENSE
# -----------------------------------------------------------------------------

GO ?= go
VERSION := $(shell cat VERSION)
BIN_FLAGS := -trimpath -ldflags "-X main.version=$(VERSION)"
SOURCE_FILES := go.mod go.sum Makefile VERSION README.md LICENSE THIRD_PARTY_NOTICES

.PHONY: test build source package verify-release

test:
	$(GO) test -count=1 ./...
	$(GO) vet ./...
	sh -n packaging/update_script packaging/rc.d/ccu-modbusclient packaging/www/update-check.cgi packaging/bin/register_interface
	node --check packaging/www/app.js
	python3 tools/release_audit.py --sources

build:
	mkdir -p build/bin/linux-amd64 build/bin/linux-arm64 build/bin/linux-arm
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(BIN_FLAGS) -o build/bin/linux-amd64/ccu-modbusclientd ./cmd/ccu-modbusclientd
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(BIN_FLAGS) -o build/bin/linux-amd64/ccu-modbusclient-config ./cmd/ccu-modbusclient-config
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(BIN_FLAGS) -o build/bin/linux-arm64/ccu-modbusclientd ./cmd/ccu-modbusclientd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(BIN_FLAGS) -o build/bin/linux-arm64/ccu-modbusclient-config ./cmd/ccu-modbusclient-config
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(BIN_FLAGS) -o build/bin/linux-arm/ccu-modbusclientd ./cmd/ccu-modbusclientd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(BIN_FLAGS) -o build/bin/linux-arm/ccu-modbusclient-config ./cmd/ccu-modbusclient-config

source:
	rm -rf build/source
	mkdir -p build/source
	cp -a cmd internal packaging tools third_party build/source/
	cp -a $(SOURCE_FILES) build/source/
	cd build/source && tar -czf ../CCU-ModbusClient-$(VERSION)-SOURCE.tar.gz .

package: test build source
	rm -rf build/pkg
	mkdir -p build/pkg/bin build/pkg/rc.d build/pkg/www build/pkg/licenses build/pkg/examples build/pkg/lib
	cp -a build/bin/linux-* build/pkg/bin/
	cp packaging/bin/update_addon packaging/bin/register_interface build/pkg/bin/
	cp packaging/update_script build/pkg/
	cp packaging/rc.d/ccu-modbusclient build/pkg/rc.d/
	cp packaging/www/*.cgi packaging/www/app.js packaging/www/style.css build/pkg/www/
	cp packaging/lib/session.tcl build/pkg/lib/session.tcl
	cp packaging/examples/config.json build/pkg/examples/
	cp VERSION LICENSE THIRD_PARTY_NOTICES README.md build/pkg/
	cp third_party/licenses/* build/pkg/licenses/
	cp -a build/source build/pkg/source
	chmod 755 build/pkg/update_script build/pkg/bin/update_addon build/pkg/bin/register_interface build/pkg/www/*.cgi build/pkg/rc.d/ccu-modbusclient
	cd build/pkg && tar -czf ../CCU-ModbusClient-$(VERSION)-INSTALL.tar.gz .
	python3 tools/release_audit.py --package

verify-release:
	python3 tools/release_audit.py --sources --package
