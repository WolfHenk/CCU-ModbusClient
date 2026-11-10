#!/bin/tclsh
# -----------------------------------------------------------------------------
# Projekt:      CCU-ModbusClient
# Datei:        packaging/www/update-check.cgi
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
source /usr/local/addons/ccu-modbusclient/lib/session.tcl
puts "Content-Type: application/json; charset=utf-8\r"
puts "Cache-Control: no-store\r"
set sid [query_value sid]
if {![check_session $sid]} {
    puts "Status: 403 Forbidden\r\n\r"
    puts {{"ok":false,"error":"Sitzung ungültig"}}
    exit 0
}
set action [query_value action]
set method [expr {[info exists env(REQUEST_METHOD)] ? $env(REQUEST_METHOD) : "GET"}]
if {$action eq "check" || $action eq "status"} {
    if {$method ne "GET"} {
        puts "Status: 405 Method Not Allowed\r\n\r"
        puts {{"ok":false,"error":"Nur GET zulässig"}}
        exit 0
    }
} elseif {$action eq "install"} {
    if {$method ne "POST"} {
        puts "Status: 405 Method Not Allowed\r\n\r"
        puts {{"ok":false,"error":"Nur POST zulässig"}}
        exit 0
    }
    # A non-simple same-origin AJAX header blocks cross-site form submissions.
    if {![info exists env(HTTP_X_REQUESTED_WITH)] || $env(HTTP_X_REQUESTED_WITH) ne "XMLHttpRequest"} {
        puts "Status: 403 Forbidden\r\n\r"
        puts {{"ok":false,"error":"Ungültige Anforderung"}}
        exit 0
    }
    # Reject explicit cross-site requests even with a valid session ID.
    if {[info exists env(HTTP_ORIGIN)] && $env(HTTP_ORIGIN) ne ""} {
        set origin $env(HTTP_ORIGIN)
        set host [expr {[info exists env(HTTP_HOST)] ? $env(HTTP_HOST) : ""}]
        if {$origin ne "http://$host" && $origin ne "https://$host"} {
            puts "Status: 403 Forbidden\r\n\r"
            puts {{"ok":false,"error":"Ungültiger Ursprung"}}
            exit 0
        }
    }
} else {
    puts "Status: 400 Bad Request\r\n\r"
    puts {{"ok":false,"error":"Aktion unbekannt"}}
    exit 0
}
set cmd /usr/local/addons/ccu-modbusclient/bin/self_update
if {$action eq "install"} {set mode start} else {set mode $action}
if {[catch {set reply [exec -- $cmd $mode]}]} {
    puts "Status: 503 Service Unavailable\r\n\r"
    puts {{"ok":false,"error":"Aktualisierungsdienst nicht verfügbar oder bereits aktiv"}}
    exit 0
}
puts "\r"
puts $reply
