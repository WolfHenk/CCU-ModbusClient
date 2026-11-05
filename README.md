# CCU-ModbusClient 0.3.0

Eigenstaendiger Modbus-TCP-Client fuer CCU/OpenCCU. 

Das Modbus-Protokoll ist ein Kommunikationsprotokoll, das auf einer Client/Server-Architektur basiert. 
Es wurde 1979 von Gould-Modicon für die Kommunikation mit seinen speicherprogrammierbaren Steuerungen ins Leben gerufen.
In der Industrie hat sich der Modbus zu einem De-facto-Standard entwickelt, da es sich um ein offenes Protokoll handelt. 
Seit 2007 ist die Version Modbus/TCP Teil der Norm IEC 61158.

## Stand und Schutzgrenzen

Version 0.3.0. Vollstaendiger Quellcode, Tests, Dokumentation und
plattformuebergreifende Binaries liegen in diesem Repository bzw. den
Release-Archiven vor. Der Bau und die lokalen Tests sind getrennt von einem
vollstaendigen nativen Installationstest auf einer OpenCCU zu bewerten.
Keine ungepruefte Installation auf produktiven CCU-Systemen.

## Lizenz

CCU-ModbusClient-Lizenz fuer kostenfreie Nutzung und Weitergabe, Version 1.1
(siehe [LICENSE](LICENSE)). Private und gewerbliche Nutzung erlaubt.
Software und weitergegebene Aenderungen duerfen nur kostenlos und mit
vollstaendigem zugehoerigen Quellcode unter denselben Bedingungen
weitergegeben werden. Entgeltliche eigenstaendige Arbeitsleistungen und
Hardware sind zulaessig, ohne Softwareaufschlaege. Drittanbieterhinweise:
[THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES) und [third_party/licenses](third_party/licenses).

## Vollstaendiger Quellcode

Quellen unter `cmd/`, `internal/`, `packaging/`; Go-Moduldateien
`go.mod`, `go.sum`; `Makefile` und `VERSION`, alle Tests und
Konfigurationsvorlage `packaging/examples/config.json`. Die genauen
Abhaengigkeitsversionen stehen in `go.mod` und `go.sum`.
Fremdlizenzen liegen unter `third_party/licenses/`. Die eigenstaendige
Bezugsdatei `releases/CCU-ModbusClient-0.3.0-SOURCE.tar.gz`
enthaelt diese Dateien gemeinsam mit dem Lizenztext.

## Bauanleitung (Linux)

Voraussetzungen: Go 1.25.14, GNU make, Node.js und POSIX shell.
Den Beispielpfad zum Go-Compiler durch den eigenen Installationspfad ersetzen.
Benoetigte Go-Module lassen sich gemaess `go.mod` und `go.sum` beziehen.
Im Projektverzeichnis:

```sh
make GO=/pfad/zur/go1.25.14/bin/go test
make GO=/pfad/zur/go1.25.14/bin/go build
make GO=/pfad/zur/go1.25.14/bin/go package
```

`make build` baut je zwei Linux-Binaries (Daemon und Konfigurationswerkzeug)
fuer AMD64, ARM64 und ARMv7. Alle Binaries sind mit `CGO_ENABLED=0`
und `-trimpath` erstellt. `make package` prueft die Quellen und erstellt
ein Installationsarchiv sowie ein gesondertes Quellarchiv. Das
Installationsarchiv fuehrt Quellen unter `source/` mit und stellt
drittlizenzierte Bestandteile mit allen Texten bereit.

Zur Verifizierung: `ccu-modbusclientd -version` und
`ccu-modbusclient-config -version` geben `0.3.0` aus.
Fuer die CCU muss die passende Architektur gewaehlt werden.

## Paket und Installation

`build/CCU-ModbusClient-0.3.0-INSTALL.tar.gz` beinhaltet Binaries, WebUI,
Skripte, Konfigurationsvorlage, vollstaendigen Projektquellcode, README,
Lizenz und Drittanbieterlizenzen. `build/CCU-ModbusClient-0.3.0-SOURCE.tar.gz`
liefert die relevanten Quellen mit Bauanleitung gesondert.

Das Installationsskript erfordert eine kontrollierte OpenCCU-Umgebung.
Die native Ende-zu-Ende-Abnahme umfasst WebUI-Sitzung, ReGa-Integration,
Konfiguration, Schreibpfade und Installations-/Deinstallationsablauf.
Ein erfolgreicher Build allein ist kein Nachweis dieser Abnahme.
