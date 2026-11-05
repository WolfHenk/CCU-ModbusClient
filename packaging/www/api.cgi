#!/bin/tclsh
# -----------------------------------------------------------------------------
# Projekt:      CCU-ModbusClient
# Datei:        packaging/www/api.cgi
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
set action [query_value action]
puts "Content-Type: application/json; charset=utf-8\r"
puts "Cache-Control: no-store\r"
if {![check_session $sid]} {
 puts "Status: 403 Forbidden\r\n\r"
 puts {{"ok":false,"error":"Sitzung ungueltig"}}
 exit 0
}
set method [expr {[info exists env(REQUEST_METHOD)] ? $env(REQUEST_METHOD) : "GET"}]
if {($action eq "read" && $method ne "GET") || ($action eq "write" && $method ne "POST") || ($action ne "read" && $action ne "write")} {
 puts "Status: 405 Method Not Allowed\r\n\r"
 puts {{"ok":false,"error":"Aktion/Methode ungueltig"}}
 exit 0
}
set cmd /usr/local/addons/ccu-modbusclient/bin/ccu-modbusclient-config
set config /usr/local/etc/config/addons/ccu-modbusclient/config.json
if {$action eq "read"} {
 if {[catch {set result [exec $cmd read $config]} err]} {
  puts "Status: 500 Internal Server Error\r\n\r"
  puts {{"ok":false,"error":"Konfiguration nicht lesbar"}}
 } else {puts "\r";puts $result}
 exit 0
}
set len 0
if {[info exists env(CONTENT_LENGTH)] && [string is integer -strict $env(CONTENT_LENGTH)]} {set len $env(CONTENT_LENGTH)}
if {$len <= 0 || $len > 1048576} {
 puts "Status: 413 Payload Too Large\r\n\r"
 puts {{"ok":false,"error":"Ungueltige Datengroesse"}}
 exit 0
}
set body [read stdin $len]
if {[string length $body] != $len} {
 puts "Status: 400 Bad Request\r\n\r"
 puts {{"ok":false,"error":"Unvollstaendige Anfrage"}}
 exit 0
}
set tmp /usr/local/tmp/ccu-modbusclient-[pid]-[clock clicks].json
if {[catch {
 set f [open $tmp {WRONLY CREAT EXCL}]
 fconfigure $f -translation binary
 puts -nonewline $f $body
 close $f
 set result [exec $cmd write $config < $tmp]
} err]} {
 catch {close $f}
 catch {file delete -- $tmp}
 puts "Status: 400 Bad Request\r\n\r"
 puts {{"ok":false,"error":"Konfiguration ungueltig oder Speichern fehlgeschlagen"}}
 exit 0
}
file delete -- $tmp
puts "\r"
puts {{"ok":true,"restart_required":true}}
