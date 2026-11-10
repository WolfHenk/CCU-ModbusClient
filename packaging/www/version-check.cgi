#!/bin/tclsh
# -----------------------------------------------------------------------------
# Projekt:      CCU-ModbusClient
# Datei:        packaging/www/version-check.cgi
# Ersteller:    Wolfram Henkel
# Datum:        10. Nov. 2026
# Version:      0.3.6
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
# Version query for OpenCCU's page "Zusatzsoftware" (rc.d info "Update:").
# cp_software.cgi calls it without a session:
#   ?cmd=check_version  -> latest published version as plain text, or "n/a"
#   ?cmd=download       -> redirect to the installer archive of that version
# Read-only: nothing is installed here; the URL is fixed, no input is used
# except the command name.

set cmd check_version
if {[info exists env(QUERY_STRING)]} {
    foreach pair [split $env(QUERY_STRING) &] {
        if {$pair eq "cmd=download"} { set cmd download }
    }
}
set latest ""
if {![catch {exec /usr/local/addons/ccu-modbusclient/bin/self_update check} reply]} {
    regexp {"latest":"([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})"} $reply all latest
}
if {$cmd eq "download" && $latest ne ""} {
    set url "https://github.com/WolfHenk/CCU-ModbusClient/raw/v$latest/releases/CCU-ModbusClient-$latest-INSTALL.tar.gz"
    puts -nonewline "Content-Type: text/html; charset=utf-8\r\n\r\n"
    puts -nonewline "<html><head><meta http-equiv='refresh' content='0; url=$url' /></head><body><a href='$url'>CCU-ModbusClient $latest</a></body></html>"
} else {
    puts -nonewline "Content-Type: text/plain; charset=iso-8859-1\r\nCache-Control: no-store\r\n\r\n"
    if {$latest ne ""} { puts -nonewline $latest } else { puts -nonewline "n/a" }
}
