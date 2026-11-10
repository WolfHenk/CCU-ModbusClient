# CCU-ModbusClient

**Modbus-TCP-Client für CCU/OpenCCU** · Version 0.3.2

Eigenständiges Add-on, das Modbus-TCP-Geräte (z. B. SPS, I/O-Module, Zähler) in eine
HomeMatic-Zentrale einbindet. Die eingesstellte Software *CCU-Modbus* sowie deren Geräte, Daten
und Kennungen werden nicht übernommen.

| | |
|---|---|
| Version | 0.3.2 |
| Plattformen | Linux AMD64, ARM64, ARMv7 |
| Lizenz | [CCU-ModbusClient-Lizenz 1.1](LICENSE) – kostenlos, auch gewerblich |
| Sprache | Go 1.25.14, statisch gebaut |

---

## Was ist Modbus?

Modbus ist ein Kommunikationsprotokoll nach dem Client/Server-Prinzip. Gould-Modicon entwickelte es
1979 für seine speicherprogrammierbaren Steuerungen. Weil die Spezifikation offen zugänglich
ist, wurde Modbus in der Industrie zum De-facto-Standard. Die TCP/IP-Variante Modbus/TCP ist
seit 2007 Bestandteil der Feldbusnorm IEC 61158.

Heute ist Modbus weit verbreitet in Solaranlagen und Wechselrichtern, Klimaanlagen, Heizungen
und Wärmepumpen.

**Client und Server statt Master und Slave:** Andere Bussysteme sprechen von *Master* (der
steuernde Rechner) und *Slave* (die gesteuerte Baugruppe). Bei Modbus heißt die steuernde
Rechnereinheit **Client**, das angesprochene Gerät **Server**. Die CCU mit CCU-ModbusClient ist
also der Client; Wechselrichter, Wärmepumpe oder I/O-Modul sind die Server.

**CCU-ModbusClient ist damit ein Zusatz zur CCU, der ihr ermöglicht, Modbus-Server
anzusprechen, Daten von ihnen zu lesen und Daten an sie zu schreiben.**

Dabei wurde Wert darauf gelegt, dass in der CCU alles wie ein ganz normales Gerät aussieht:
Schaltaktoren, Kontakte und Messwerte erscheinen mit Kanälen und Datenpunkten wie bei
HomeMatic-Geräten und lassen sich in Programmen und Skripten wie gewohnt verwenden. Die Übersetzung in die Modbus-Sichtweise geschieht vollständig im
CCU-ModbusClient-System.

---

## Modbus-Begriffe einfach erklärt

Ein Modbus-Server stellt seine Daten in vier Bereichen bereit. Jeder Wert hat eine
**Adresse** (eine Nummer), über die der Client ihn anspricht.

| Begriff | Größe | Lesen | Schreiben | Vorstellung | Beispiel |
|---|---|:---:|:---:|---|---|
| **Coil** | 1 Bit (ein/aus) | ✅ | ✅ | Schalter, den man umlegen kann | Relaisausgang, Pumpe ein/aus |
| **Discrete Input** | 1 Bit (ein/aus) | ✅ | – | Kontrolllampe, die man nur ablesen kann | Türkontakt, Störmeldung |
| **Input Register** | 16 Bit (Zahl) | ✅ | – | Anzeige, die man nur ablesen kann | gemessene Temperatur, Leistung |
| **Holding Register** | 16 Bit (Zahl) | ✅ | ✅ | Einstellrad, das man ablesen und verstellen kann | Sollwert, Betriebsart |

Weitere Begriffe:

- **Unit-ID** – Nummer des Geräts hinter einer IP-Adresse. Meist 1; wichtig, wenn ein Gateway
  mehrere Geräte bedient.
- **Datentyp** – ein Register fasst 16 Bit, also Zahlen von 0 bis 65535 (`uint16`) oder
  −32768 bis 32767 (`int16`). Größere Zahlen und Kommazahlen (`int32`, `uint32`, `float32`)
  belegen zwei Register hintereinander.
- **Faktor und Offset** – viele Geräte liefern ganze Zahlen, z. B. `215` für 21,5 °C. Mit
  Faktor `0.1` rechnet CCU-ModbusClient daraus den richtigen Wert.
- **Byte- und Wortreihenfolge** – Hersteller legen die Teile einer Zahl unterschiedlich ab.
  Zeigt ein Wert Unsinn, hilft meist das Tauschen der Bytes oder der Worte (`byte_swap`,
  `word_swap`).
- **Abfrageintervall** – wie oft (in Sekunden) ein Wert vom Gerät gelesen wird.

Welche Adressen, Datentypen und Faktoren ein Gerät verwendet, steht in seiner
Modbus-Dokumentation (oft „Registerliste“ oder „Modbus-Map“ genannt).

---

## Lizenz in Kürze

- ✅ Private und gewerbliche Nutzung erlaubt
- ✅ Ändern und Weitergeben erlaubt – **nur kostenlos**, mit vollständigem Quellcode und unter
  denselben Bedingungen
- ✅ Arbeitsleistung (Installation, Einrichtung, Support) und Hardware dürfen berechnet werden
- ❌ Kein Entgelt für die Software selbst, auch kein Aufschlag auf Preise oder Stundensätze

Maßgeblich ist der vollständige Text in [LICENSE](LICENSE).
Rechte Dritter: [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES) und [third_party/licenses](third_party/licenses).

---

## Änderungen in 0.3.2

- **Kanaltypen mit Beschriftung**: Zahlenwerte erscheinen in der CCU unter
  vorhandenen Kanaltypen, deren Datenpunkte in der WebUI beschriftet sind –
  ohne Änderung an OpenCCU-Dateien. Istwerte je Messgröße: allgemein
  „Eingangswert“, Temperatur, Rel. Luftfeuchte, Leistung, Energie-Zähler, Strom,
  Spannung, Frequenz. Sollwerte (Holding-Register mit Richtung Ausgang):
  „Ausgangspegel“ mit der eingestellten Einheit.
  Datenpunkte heißen jetzt z. B. `…:5.VALUE`, `.TEMPERATURE` oder `.LEVEL`
  statt `.VOLTAGE`/`.VALUE`. **Bestehende Geräte** übernimmt die CCU beim Update
  selbst mit den neuen Kanaltypen; Geräte- und Kanalnamen können dabei auf die
  Standardnamen zurückfallen. Programme und Skripte mit den alten
  Datenpunktnamen sind anzupassen.
- **Einheiten**: Sonderzeichen wie „°“ werden im ISO-8859-1-Zeichensatz der CCU
  übertragen und auch in Skripten korrekt geliefert (vorher „Â°C“ bzw. „&#xB0;C“).
- **Bereichsprüfung**: Sollwerte außerhalb des Datentyps werden abgelehnt statt
  auf den Grenzwert begrenzt; der Grund steht in der Fehlermeldung.
  Werte mit Faktor 0,1 (z. B. 45,3 °C) werden nicht mehr fälschlich abgewiesen.
- **Restlose Deinstallation**: Beim Entfernen über *Einstellungen →
  Systemsteuerung → Zusatzsoftware* löscht die CCU alle CCU-ModbusClient-Geräte
  samt Kanälen, Datenpunkten sowie Raum- und Gewerkzuordnungen und anschließend
  die Schnittstelle. Auch die Konfiguration (`config.json`) wird entfernt –
  vorher bei Bedarf sichern. Programme, die Datenpunkte der Geräte verwenden,
  bleiben bestehen und sind anzupassen. Gelingt das Löschen in der CCU nicht,
  bricht die Deinstallation ab, ohne etwas zu entfernen, und kann wiederholt
  werden.

**Vor jeder Deinstallation:** Die Geräte- und Kanalnamen, Datenpunktadressen
und die Datei `config.json` bei Bedarf sichern. Alle CCU-Programme, Direkt-
verknüpfungen und Skripte auf Verweise auf `CCU-ModbusClient`-Geräte bzw.
`MBC...`-Adressen prüfen und solche Verweise vorher entfernen oder ersetzen.
**Die Deinstallation löscht keine fremden Programme und repariert keine
verbleibenden Skriptreferenzen.** Ein vollständiger Rückbau des Add-ons ist
nicht gleichbedeutend mit einer automatischen Reparatur benutzereigener Logik.

## Änderungen in 0.3.1

- **XML-RPC `setValue`**: Schreibzugriffe der CCU auf schaltbare Coils und
  schreibbare Holding-Register gelangen jetzt über die tatsächlich registrierte
  XML-RPC-Schnittstelle `/RPC3` in die Modbus-Schreibwarteschlange.
- **XML-RPC `getValue`**: ReGa und CCU-Skripte können einzelne Werte gezielt
  abfragen. Zurückgegeben wird der letzte gültige Wert aus dem periodisch
  aktualisierten Wertecache; die Abfrage löst keinen zusätzlichen Modbus-Lesevorgang aus.
  Eingänge und 64-Bit-Zahlen bleiben lesbar (64 Bit als Dezimaltext).
- **Fehlertexte**: Abgelehnte Schreibversuche melden erkennbare Eingabefehler,
  unbekannte und schreibgeschützte Datenpunkte. Unbekannte interne Fehler werden
  weiterhin nicht preisgegeben.
- **Bedeutung eines erfolgreichen Schreibaufrufs**: Er bestätigt die Aufnahme
  in die Schreibwarteschlange, **nicht** schon die erfolgreiche Modbus-Quittung.
- Bewährte Geräteadressen und die getrennte Konfiguration des bisherigen
  CCU-Modbus bleiben unverändert.

---

## Projektstand

> **⚠️ Hinweis:** Bau und lokale Tests ersetzen keinen Installationstest auf einer echten
> OpenCCU. Verändern von Registern auf einer funktionierenden Anlage kann die Anlageneinstellungen ändern und zu Schäden führen

Quellcode, Tests, Dokumentation und Binaries für alle Plattformen liegen in diesem Repository
bzw. in den Release-Archiven.

---

## Quellcode

| Pfad | Inhalt |
|---|---|
| `cmd/` | Programme: Daemon `ccu-modbusclientd`, Werkzeug `ccu-modbusclient-config` |
| `internal/` | Programmbibliotheken und Tests |
| `packaging/` | WebUI, Skripte, Startdatei, Konfigurationsvorlage `examples/config.json` |
| `go.mod`, `go.sum` | Abhängigkeiten mit genauen Versionen |
| `Makefile`, `VERSION` | Bau und Versionsnummer |
| `third_party/licenses/` | Lizenztexte der Fremdbibliotheken |

---

## Bauen (Linux)

**Voraussetzungen:** Go 1.25.14, GNU make, Node.js, POSIX-Shell.

Im Projektverzeichnis, mit dem eigenen Pfad zum Go-Compiler:

```sh
make GO=/pfad/zur/go1.25.14/bin/go test      # Tests
make GO=/pfad/zur/go1.25.14/bin/go build     # Binaries
make GO=/pfad/zur/go1.25.14/bin/go package   # Installations- und Quellarchiv
```

- `make build` erzeugt Daemon und Konfigurationswerkzeug für AMD64, ARM64 und ARMv7
  (`CGO_ENABLED=0`, `-trimpath`).
- `make package` prüft die Quellen und erstellt beide Archive.
- Kontrolle: `ccu-modbusclientd -version` und `ccu-modbusclient-config -version` melden `0.3.2`.

---

## Pakete

| Archiv | Inhalt |
|---|---|
| `CCU-ModbusClient-0.3.2-INSTALL.tar.gz` | Binaries, WebUI, Skripte, Konfigurationsvorlage, README, Lizenzen, kompletter Quellcode unter `source/` |
| `CCU-ModbusClient-0.3.2-SOURCE.tar.gz` | Quellcode mit Bauanleitung und Lizenztexten |

Fertige Archive liegen unter `releases/`, selbst gebaute unter `build/`.

---

## Installation

Installation als Zusatzsoftware über die WebUI der OpenCCU; die passende Architektur wählen.

---

## Marken

Modbus ist eine Marke der jeweiligen Rechteinhaber. HomeMatic, CCU und OpenCCU sind Marken bzw.
Bezeichnungen ihrer jeweiligen Inhaber. Dieses Projekt ist von ihnen weder zertifiziert noch
unterstützt.

---

© 2026 Wolfram Henkel
