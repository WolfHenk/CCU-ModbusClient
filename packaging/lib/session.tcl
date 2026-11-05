# -----------------------------------------------------------------------------
# Projekt:      CCU-ModbusClient
# Datei:        packaging/lib/session.tcl
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

load tclrega.so

proc url_decode {s} {
    set out ""
    set i 0
    set n [string length $s]
    while {$i < $n} {
        set c [string index $s $i]
        if {$c == "+"} {
            append out " "
            incr i
        } elseif {$c == "%" && [expr {$i + 2}] < $n} {
            set hex [string range $s [expr {$i + 1}] [expr {$i + 2}]]
            set value 0
            if {[scan $hex %x value] == 1} {
                append out [format %c $value]
                incr i 3
            } else {
                append out $c
                incr i
            }
        } else {
            append out $c
            incr i
        }
    }
    return $out
}

proc check_session {sid} {
    if {[regexp {^@([0-9a-zA-Z]{10})@$} $sid all sidnr]} {
        set res [lindex [rega_script "Write(system.GetSessionVarStr('$sidnr'));"] 1]
        if {$res != ""} { return 1 }
    }
    return 0
}

proc query_value {name} {
    if {![info exists ::env(QUERY_STRING)]} { return "" }
    foreach part [split $::env(QUERY_STRING) "&"] {
        set p [string first "=" $part]
        if {$p < 0} {
            set key [url_decode $part]
            set val ""
        } else {
            set key [url_decode [string range $part 0 [expr {$p-1}]]]
            set val [url_decode [string range $part [expr {$p+1}] end]]
        }
        if {$key == $name} { return $val }
    }
    return ""
}
