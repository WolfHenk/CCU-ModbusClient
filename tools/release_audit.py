#!/usr/bin/env python3
# -----------------------------------------------------------------------------
# Projekt:      CCU-ModbusClient
# Datei:        tools/release_audit.py
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

"""Audit license, own file headers and full source and binary release archives."""
from pathlib import Path
import argparse
import hashlib
import tarfile
import sys

ROOT = Path(__file__).resolve().parents[1]
VERSION = "0.3.0"
FOLDERS = ("cmd", "internal", "packaging", "tools", "third_party")
ROOT_FILES = ("go.mod", "go.sum", "VERSION", "Makefile", "README.md",
              "LICENSE", "THIRD_PARTY_NOTICES")
CODE_EXT = (".go", ".js", ".css", ".tcl", ".cgi", ".py")
OTHER_CODE = ("go.mod", "Makefile", "packaging/bin/register_interface",
              "packaging/bin/update_addon", "packaging/bin/register_devtype", "packaging/rc.d/ccu-modbusclient",
              "packaging/update_script")

def expected():
    files = set()
    for folder in FOLDERS:
        files.update(f.relative_to(ROOT).as_posix()
                     for f in (ROOT/folder).rglob("*") if f.is_file())
    files.update(ROOT_FILES)
    return files

def test_sources(names):
    assert (ROOT/"VERSION").read_text().strip() == VERSION
    assert b"Version 1.1" in (ROOT/"LICENSE").read_bytes()
    headers = 0
    for name in sorted(names):
        p = ROOT/name
        assert p.is_file(), f"missing: {name}"
        if not (p.suffix in CODE_EXT or name in OTHER_CODE):
            continue
        lines = p.read_text(encoding="utf-8").splitlines()
        assert any("Projekt:      CCU-ModbusClient" in s for s in lines[:6]), f"missing project header: {name}"
        assert f"Datei:        {name}" in "\n".join(lines[:9]), f"wrong file name: {name}"
        assert "Version:      0.3.0" in "\n".join(lines[:9]), f"wrong version: {name}"
        if lines[0].startswith("#!"):
            assert "CCU-ModbusClient" in "\n".join(lines[1:5]), f"shebang header missing: {name}"
        headers += 1
    assert headers >= 145, f"too few owned code files: {headers}"
    print(f"SOURCE_AUDIT_PASS: {len(names)} files, {headers} licensed own code files")

def entries(path):
    result = {}
    with tarfile.open(path, "r:gz") as ar:
        for entry in ar:
            if not entry.isfile():
                continue
            key = entry.name.removeprefix("./")
            assert key not in result, f"duplicate: {key}"
            result[key] = hashlib.sha256(ar.extractfile(entry).read()).hexdigest()
    return result

def test_packages(names):
    source_archive = ROOT/f"build/CCU-ModbusClient-{VERSION}-SOURCE.tar.gz"
    install_archive = ROOT/f"build/CCU-ModbusClient-{VERSION}-INSTALL.tar.gz"
    assert source_archive.is_file()
    assert install_archive.is_file()
    source = entries(source_archive)
    install = entries(install_archive)
    assert set(source) == names, f"source archive coverage: missing={names-set(source)} extra={set(source)-names}"
    for name in names:
        assert source[name] == hashlib.sha256((ROOT/name).read_bytes()).hexdigest(), f"source differs: {name}"
    embedded = {k.removeprefix("source/"): v for k,v in install.items() if k.startswith("source/")}
    assert embedded == source, f"embedded sources incomplete: {set(source)-set(embedded)}"
    required = ("LICENSE",
                "VERSION", "README.md", "THIRD_PARTY_NOTICES", "update_script",
                "licenses/go-stdlib-LICENSE", "licenses/golang.org-x-net-LICENSE",
                "licenses/golang.org-x-text-LICENSE", "examples/config.json",
                "www/index.cgi", "www/app.js", "www/style.css",
                "www/img/modbus-logo.png", "www/img/modbus-logo-50.png",
                "www/img/modbus-logo-250.png", "bin/register_devtype")
    for name in required:
        assert name in install, f"missing installer file: {name}"
    for arch in ("linux-amd64", "linux-arm64", "linux-arm"):
        for binary in ("ccu-modbusclientd", "ccu-modbusclient-config"):
            assert f"bin/{arch}/{binary}" in install, f"missing {arch}/{binary}"
    for name in list(source) + list(install):
        low = name.lower()
        assert not any(x in low for x in ("lizenzpruefung", "review", "license-apache", "commons-clause")), f"private/obsolete file leaked: {name}"
    print(f"PACKAGE_AUDIT_PASS: {len(source)} source files; {len(install)} installer files; all six binaries")
    for path in (source_archive, install_archive):
        print(f"{path.name}: sha256 {hashlib.sha256(path.read_bytes()).hexdigest()}")

if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--sources", action="store_true")
    ap.add_argument("--package", action="store_true")
    arg = ap.parse_args()
    if not (arg.sources or arg.package):
        ap.error("choose --sources or --package")
    try:
        names = expected()
        test_sources(names)
        if arg.package:
            test_packages(names)
    except (AssertionError, OSError, ValueError) as exc:
        print("AUDIT_FAIL:", exc, file=sys.stderr)
        sys.exit(1)
