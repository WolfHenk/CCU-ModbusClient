#!/usr/bin/env python3
# -----------------------------------------------------------------------------
# Projekt:      CCU-ModbusClient
# Datei:        tools/test_self_update.py
# Ersteller:    Wolfram Henkel
# Datum:        10. Nov. 2026
# Version:      0.3.4
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

"""Isolated end-to-end tests for update download, verify, backup, install."""
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import time

SOURCE = Path(__file__).resolve().parents[1] / "packaging/bin/self_update"


def run_case(invalid_checksum=False, unavailable=False):
    with tempfile.TemporaryDirectory(prefix="ccu-modbusclient-updater-test-") as tmp:
        root = Path(tmp)
        fs = root / "root"
        base = fs / "usr/local/addons/ccu-modbusclient"
        conf = fs / "usr/local/etc/config/addons/ccu-modbusclient"
        etc = fs / "usr/local/etc/config"
        rc = etc / "rc.d"
        bin_dir = root / "bin"
        fixtures = root / "fixtures"
        for path in (base, conf, rc, bin_dir, fixtures):
            path.mkdir(parents=True, exist_ok=True)
        (base / "VERSION").write_text("0.3.3\n")
        (base / "placeholder.txt").write_text("prior installed source")
        (conf / "config.json").write_text('{"schema":2,"devices":[{"id":"keep"}]}')
        (etc / "InterfacesList.xml").write_text("<interfaces/>")
        (rc / "ccu-modbusclient").write_text("#!/bin/sh\nexit 0\n")
        original = (conf / "config.json").read_bytes()
        (fixtures / "remote-version").write_text("0.3.4\n")

        archive = fixtures / "archive.tar.gz"
        with tarfile.open(archive, "w:gz") as tar:
            files = {
                "./VERSION": b"0.3.4\n",
                "./source/LICENSE": b"test-only license",
                "./update_script": (
                    "#!/bin/sh\nset -eu\n"
                    + "printf '0.3.4\\n' > "
                    + repr(str(base / "VERSION")) + "\n"
                ).encode(),
            }
            for name, payload in files.items():
                info = tarfile.TarInfo(name)
                info.size = len(payload)
                info.mode = 0o755 if name.endswith("update_script") else 0o644
                tar.addfile(info, io.BytesIO(payload))
        checksum = hashlib.sha256(archive.read_bytes()).hexdigest()
        if invalid_checksum:
            checksum = "0" * 64
        (fixtures / "manifest").write_text(
            f"{checksum}  CCU-ModbusClient-0.3.4-INSTALL.tar.gz\n"
        )
        curl = bin_dir / "curl"
        curl.write_text(
            "#!/bin/sh\n"
            'for url do :; done\n'
            'case "$url" in\n'
            ' *"/main/VERSION") cat "' + str(fixtures / "remote-version") + '" ;;\n'
            ' *".tar.gz.sha256") cat "' + str(fixtures / "manifest") + '" ;;\n'
            ' *".tar.gz") cat "' + str(archive) + '" ;;\n'
            ' *"127.0.0.1:18702/RPC3") '
            "printf '<methodResponse><params><string>system.listMethods</string></params></methodResponse>' ;;\n"
            ' *) exit 22;;\n'
            "esac\n"
        )
        curl.chmod(0o755)
        pidof = bin_dir / "pidof"
        pidof.write_text("#!/bin/sh\nexit 0\n")
        pidof.chmod(0o755)
        worker = root / "self_update"
        code = SOURCE.read_text()
        changes = {
            "BASE=/usr/local/addons/ccu-modbusclient": f"BASE={base}",
            "CONF=/usr/local/etc/config/addons/ccu-modbusclient/config.json": f"CONF={conf / 'config.json'}",
            "ROOT=https://raw.githubusercontent.com/WolfHenk/CCU-ModbusClient": "ROOT=https://testing.invalid/project",
            "STATUS=/usr/local/tmp/ccu-modbusclient-update.status": f"STATUS={root / 'update.status'}",
            "LOCK=/usr/local/tmp/ccu-modbusclient-update.lock": f"LOCK={root / 'update.lock'}",
            "LOG=/usr/local/tmp/ccu-modbusclient-update.log": f"LOG={root / 'update.log'}",
            "dir=$(mktemp -d /usr/local/tmp/ccu-modbusclient-release.XXXXXX)": f"dir=$(mktemp -d {root}/release.XXXXXX)",
            "backupdir=/usr/local/etc/config/addons/ccu-modbusclient/backups": f"backupdir={conf / 'backups'}",
            '''tar -czf "$backup" -C / \\
      usr/local/addons/ccu-modbusclient \\
      usr/local/etc/config/addons/ccu-modbusclient/config.json \\
      usr/local/etc/config/InterfacesList.xml \\
      usr/local/etc/config/rc.d/ccu-modbusclient''': (
                f'tar -czf "$backup" -C "{fs}" \\\n'
                '      usr/local/addons/ccu-modbusclient \\\n'
                '      usr/local/etc/config/addons/ccu-modbusclient/config.json \\\n'
                '      usr/local/etc/config/InterfacesList.xml \\\n'
                '      usr/local/etc/config/rc.d/ccu-modbusclient'
            ),
        }
        for old, new in changes.items():
            assert code.count(old) == 1, (old, code.count(old))
            code = code.replace(old, new)
        worker.write_text(code)
        worker.chmod(0o700)
        env = dict(os.environ, PATH=str(bin_dir) + os.pathsep + os.environ["PATH"])
        if unavailable:
            (fixtures / "remote-version").write_text("0.3.3\n")
        result = subprocess.run([str(worker), "check"], capture_output=True,
                                text=True, env=env, check=True, timeout=20)
        check = json.loads(result.stdout)
        assert check["available"] == (not unavailable), check
        if unavailable:
            print("SELF_UPDATE_NO_NEWER_VERSION_PASS")
            return
        result = subprocess.run([str(worker), "start"], capture_output=True,
                                text=True, env=env, check=True, timeout=15)
        assert json.loads(result.stdout)["started"] is True, result
        end = time.monotonic() + 8
        while time.monotonic() < end:
            if (root / "update.status").exists():
                state = json.loads((root / "update.status").read_text())["state"]
                if state in ("done", "failed"):
                    break
            time.sleep(0.1)
        else:
            raise AssertionError("worker never finished")
        expected_state = "failed" if invalid_checksum else "done"
        assert state == expected_state, ((root / "update.log").read_text(), state)
        assert (conf / "config.json").read_bytes() == original
        expected_version = "0.3.3\n" if invalid_checksum else "0.3.4\n"
        assert (base / "VERSION").read_text() == expected_version
        backups = list((conf / "backups").glob("*.tar.gz")) if (conf / "backups").exists() else []
        assert bool(backups) == (not invalid_checksum), backups
        print("SELF_UPDATE_BAD_CHECKSUM_PASS" if invalid_checksum
              else "SELF_UPDATE_END_TO_END_PASS")


if __name__ == "__main__":
    run_case(unavailable=True)
    run_case(invalid_checksum=True)
    run_case()
