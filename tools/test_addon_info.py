#!/usr/bin/env python3
# -----------------------------------------------------------------------------
# Projekt:      CCU-ModbusClient
# Datei:        tools/test_addon_info.py
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

"""Regression: the native OpenCCU addon page merges info.de/en with info."""
import subprocess
from pathlib import Path

SCRIPT = Path(__file__).resolve().parents[1] / "packaging/rc.d/ccu-modbusclient"


def call(argument):
    result = subprocess.run(["sh", str(SCRIPT), argument], capture_output=True,
                            text=True, check=True, timeout=8)
    assert result.stderr == "", (argument, result.stderr)
    return result.stdout


base = call("info")
assert base.count("Name: CCU-ModbusClient") == 1
assert base.count("Version: ") == 1
assert base.count("Info: <img src=") == 1
assert base.count("Config-Url: ") == 1
assert base.count("Operations: ") == 1

for language in ("de", "en"):
    localized = call(f"info.{language}")
    combined = localized + base
    assert localized == "", f"info.{language} must not repeat the common fields"
    assert combined.count("Name: CCU-ModbusClient") == 1
    assert combined.count("Version: ") == 1
    assert combined.count("Info: <img src=") == 1

print("ADDON_INFO_REGRESSION_PASS: German and English, no duplicate name/version/logo")
