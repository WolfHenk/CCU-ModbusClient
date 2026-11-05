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
if {![check_session $sid]} {puts {<!doctype html><html><meta charset="utf-8"><p>Sitzung ungueltig. Bitte erneut ueber die OpenCCU anmelden.</p></html>};exit 0}
puts {<!doctype html><html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>CCU-ModbusClient</title><link rel="stylesheet" href="style.css"></head><body>
<header><div class="brand">OpenCCU <span>/</span> ModbusClient</div><div class="header-actions"><span id="status">Konfiguration wird geladen …</span><button class="primary" id="save">Änderungen speichern</button></div></header>
<main><div class="page-heading"><div><h1>Modbus TCP Client</h1><p>Geräte und Register verwalten</p></div><span class="version">Version 0.3.0</span></div>
<div class="notice">Änderungen werden geprüft und gespeichert. Für die Übernahme ist anschließend ein Neustart des Client-Dienstes erforderlich.</div>
<section class="panel"><div class="panel-title"><h2>Geräte</h2><div class="buttons"><button id="addDevice">+ Gerät hinzufügen</button><button class="danger" id="removeDevice">Gerät entfernen</button></div></div>
<div class="panel-content"><div class="fieldrow"><label class="full">Gerät auswählen<select id="device"></select></label></div>
<div class="fieldgrid"><label class="wide">Gerätename<input id="name" placeholder="z. B. Siemens LOGO"></label><label class="wide">IP-Adresse / Host<input id="host" placeholder="192.168.3.207"></label><label>TCP-Port<input id="port" type="number" value="502"></label><label>Unit-ID<input id="unitID" type="number" value="1"></label><label>Timeout (ms)<input id="timeout" type="number" value="1500"></label><label>Wiederholungen<input id="retries" type="number" value="0"></label></div>
<label class="toggle"><input id="enabled" type="checkbox"> Gerät aktiviert</label></div></section>
<section class="panel"><div class="panel-title"><h2>Register</h2><div class="buttons"><button id="addReg">+ Register hinzufügen</button><button class="danger" id="removeReg">Register entfernen</button></div></div>
<div class="panel-content"><label class="full">Register auswählen<select id="register"></select></label><h3>Register-Einstellungen</h3>
<div class="fieldgrid"><label class="wide">Bezeichnung<input id="regName"></label><label>Registertyp<select id="regType"><option value="coil">Coil</option><option value="discrete">Discrete Input</option><option value="holding">Holding Register</option><option value="input">Input Register</option></select></label><label>Adresse<input id="regAddr" type="number" min="0" max="65535"></label><label>Datentyp<select id="regData"><option>bool</option><option>uint8</option><option>int8</option><option>uint16</option><option>int16</option><option>uint32</option><option>int32</option><option>uint64</option><option>int64</option><option>float32</option></select></label><label>Abfrageintervall (s)<input id="poll" type="number" value="1"></label></div>
<details><summary>Weitere Registeroptionen</summary><div class="fieldgrid detail-fields"><label>Numerische Richtung<select id="regDirection"><option value="">Standard</option><option value="input">Eingang</option><option value="output">Ausgang</option></select></label><label>Coil-Richtung<select id="coilDirection"><option value="">Standard</option><option value="input">Eingang</option><option value="output">Ausgang</option></select></label><label>Faktor<input id="factor" type="number" step="any" value="1"></label><label>Offset<input id="offset" type="number" step="any"></label><label>Einheit<input id="regUnit"></label><label>Messgröße<select id="measurement"><option value="">Standard</option><option>generic</option><option>number</option><option>temperature</option><option>humidity</option><option>voltage</option><option>current</option><option>power</option><option>energy</option><option>frequency</option></select></label><label>Raum-ID<input id="roomID" type="number" min="0"></label><label>Gewerk-ID<input id="functionID" type="number" min="0"></label></div><div class="checks"><label><input id="regEnabled" type="checkbox"> Aktiv</label><label><input id="byteSwap" type="checkbox"> Byte-Swap</label><label><input id="wordSwap" type="checkbox"> Word-Swap</label><label><input id="dwordSwap" type="checkbox"> DWord-Swap</label><label><input id="trueMeansOpen" type="checkbox"> True bedeutet offen</label><label><input id="invertInput" type="checkbox"> Eingang invertieren</label></div></details>
</div></section><div id="message" role="status" aria-live="polite"></div><footer>CCU-ModbusClient · Eigenständiges OpenCCU-Add-on · Entwicklungsstand</footer></main>
<script src="app.js"></script></body></html>}
