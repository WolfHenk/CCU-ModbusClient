#!/bin/tclsh
# -----------------------------------------------------------------------------
# Projekt:      CCU-ModbusClient
# Datei:        packaging/www/index.cgi
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

source /usr/local/addons/ccu-modbusclient/lib/session.tcl
set sid [query_value sid]
puts "Content-Type: text/html; charset=utf-8\r"
puts "Cache-Control: no-store\r\n"
if {![check_session $sid]} {
 puts {<!doctype html><html><meta charset="utf-8"><p>Sitzung ungueltig. Bitte erneut ueber die OpenCCU anmelden.</p></html>}
 exit 0
}
puts {<!doctype html>}
puts {<html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">}
puts {<title>CCU-ModbusClient</title><link rel="stylesheet" href="style.css?v=0.3.3-ui1"></head><body>}
puts {<header class="top"><div><img src="img/modbus-logo.png" class="logo" alt="Modbus"></div><div><h1>CCU-ModbusClient</h1><div class="sub">Modbus TCP Client für CCU/OpenCCU</div></div><div class="version">0.3.3</div></header>}
puts {<main><div id="notice" class="notice hidden"></div>}
puts {<section class="toolbar"><div><h2>Modbus-TCP-Geräte</h2><div class="hint">Geräte und Register werden lokal auf der CCU gespeichert.</div></div><button id="addDevice" class="primary">+ Gerät hinzufügen</button></section>}
puts {<div id="devices" class="devices"><div class="empty">Lade Konfiguration…</div></div>}
puts {<section class="footerbar"><button id="saveAll" class="primary">Änderungen speichern</button><span id="dirtyText" class="muted">Keine ungespeicherten Änderungen.</span></section>}
puts {<footer class="copyright">© 2026 Wolfram Henkel · CCU-ModbusClient-Lizenz 1.1</footer>}
puts {</main>}
puts {<dialog id="deviceDialog"><form method="dialog" id="deviceForm"><div class="dialoghead"><h2 id="deviceTitle">Gerät</h2><button value="cancel" class="iconbtn">×</button></div>}
puts {<div class="grid"><label>Name<input id="dName" required></label><label>IP-Adresse / Host<input id="dHost" required></label><label>Port<input id="dPort" type="number" min="1" max="65535" value="502"></label><label>Unit-ID<input id="dUnit" type="number" min="0" max="255" value="1"></label></div>}
puts {<details><summary>Erweitert</summary><div class="grid advanced"><label>Timeout (ms)<input id="dTimeout" type="number" min="100" max="30000" value="1500"></label><label>Retries<input id="dRetries" type="number" min="0" max="2" value="1"></label><label class="check"><input id="dEnabled" type="checkbox" checked> Gerät aktiv</label></div></details>}
puts {<div class="testline"><button type="button" id="testConnection">Verbindung testen</button><span id="connectionResult" role="status"></span></div>}
puts {<div class="registerHead"><h3>Register</h3><button type="button" id="addRegister">+ Register hinzufügen</button></div>}
puts {<div class="register-nav"><button type="button" onclick="document.querySelector('#deviceDialog .tablewrap').scrollTop -= 130">&#9650; Vorherige Register</button><button type="button" onclick="document.querySelector('#deviceDialog .tablewrap').scrollTop += 130">&#9660; Weitere Register</button></div>}
puts {<div class="tablewrap"><table><thead><tr><th>Name</th><th>Typ</th><th>Adresse</th><th>Datentyp</th><th>Intervall</th><th></th></tr></thead><tbody id="registerRows"></tbody></table></div>}
puts {<div class="dialogactions"><button type="button" id="deleteDevice" class="danger">Gerät löschen</button><span class="spacer"></span><button value="cancel">Abbrechen</button><button type="button" id="applyDevice" class="primary">Übernehmen</button></div></form></dialog>}
puts {<dialog id="registerDialog"><form method="dialog"><div class="dialoghead"><h2 id="registerTitle">Register</h2><button value="cancel" class="iconbtn">×</button></div>}
puts {<div class="grid"><label>Bezeichnung<input id="rName" required></label><label class="check"><input id="rEnabled" type="checkbox" checked> Register aktiv</label><label>Registertyp<select id="rType"><option value="holding">Holding Register</option><option value="input">Input Register</option><option value="coil">Coil</option><option value="discrete">Discrete Input</option></select></label><label id="rCoilDirectionWrap" style="display:none">Coil-Verwendung<select id="rCoilDirection"><option value="output">Ausgang (schalten und lesen)</option><option value="input">Eingang (nur lesen)</option></select></label><label>Registernummer<input id="rAddress" type="number" min="0" max="65535" value="0"></label><label>Datentyp<select id="rDatatype"><option>INT8</option><option>UINT8</option><option>INT16</option><option>UINT16</option><option>INT32</option><option>UINT32</option><option>INT64</option><option>UINT64</option><option>FLOAT32</option><option>BOOL</option></select></label><label>Einheit<input id="rUnit" placeholder="z.B. °C"></label><label>Numerische Richtung<select id="rNumericDirection"><option value="">Standard / Eingang</option><option value="input">Eingang</option><option value="output">Ausgang (schreibbar)</option></select></label><label>Messgröße<select id="rMeasurement"><option value="">Standard</option><option value="generic">Allgemein</option><option value="number">Zahl</option><option value="temperature">Temperatur</option><option value="humidity">Feuchte</option><option value="voltage">Spannung</option><option value="current">Strom</option><option value="power">Leistung</option><option value="energy">Energie</option><option value="frequency">Frequenz</option></select></label><label>Faktor<input id="rFactor" value="1"></label><label>Offset<input id="rOffset" value="0"></label><label>Polling (s)<input id="rPoll" type="number" min="1" max="86400" value="10"></label></div>}
puts {<p class="precision-hint" id="precisionHint" hidden>INT64 / UINT64 werden als exakte Dezimaltexte an die CCU übertragen. Faktor oder Offset ungleich 1 / 0 können Rundung verursachen.</p>}
puts {<div class="seriesbox"><label class="check"><input id="rSeries" type="checkbox"> Registerserie anlegen</label><label id="rSeriesCountWrap" style="display:none">Gesamtanzahl<input id="rSeriesCount" type="number" min="2" max="256" value="2"></label></div><details><summary>Erweitert</summary><div class="grid advanced"><label class="check"><input id="rByteSwap" type="checkbox"> Byte-Swap</label><label class="check"><input id="rWordSwap" type="checkbox"> Word-Swap (16-Bit-Wörter)</label><label class="check"><input id="rDWordSwap" type="checkbox"> DWord-Swap (32-Bit-Hälften, 64 Bit)</label><label class="check"><input id="rTrueMeansOpen" type="checkbox"> True bedeutet offen</label><label class="check" id="rInvertInputWrap" style="display:none"><input id="rInvertInput" type="checkbox"> Eingangszustand in der CCU umkehren (1 ↔ 0)</label></div></details>}
puts {<div class="testline"><button type="button" id="testRegister">Register lesen / Wert testen</button><span id="registerResult" role="status"></span></div>}
puts {<div class="dialogactions"><button type="button" id="deleteRegister" class="danger">Register löschen</button><span class="spacer"></span><button value="cancel">Abbrechen</button><button type="button" id="applyRegister" class="primary">Übernehmen</button></div></form></dialog>}
puts {<script src="app.js?v=0.3.3-ui1"></script></body></html>}
