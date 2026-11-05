#!/bin/sh
# -----------------------------------------------------------------------------
# Projekt:      CCU-ModbusClient
# Datei:        packaging/www/update-check.cgi
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

# Private GitHub repositories must not be probed anonymously for releases.
printf 'Content-Type: text/plain; charset=utf-8\r\nCache-Control: no-store\r\n\r\nn/a'
