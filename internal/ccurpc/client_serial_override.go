// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_serial_override.go
// Ersteller:    Wolfram Henkel
// Datum:        05. Nov. 2026
// Version:      0.3.0
//
// Copyright (c) 2026 Wolfram Henkel.
//
// Lizenz:       CCU-ModbusClient-Lizenz für kostenfreie Nutzung und
//               Weitergabe, Version 1.1.
//
// Private und gewerbliche Nutzung sind erlaubt.
// Die Software und Änderungen dürfen nur kostenlos unter
// derselben Lizenz und mit vollständigem Quellcode weitergegeben werden.
//
// Vollständiger Lizenztext: LICENSE
// -----------------------------------------------------------------------------

package ccurpc

import "strings"

// IsClientSerialForEndpoint accepts a stable 13-character normal address or
// a persistently assigned 14-character address with one alphanumeric character immediately after MBC.
func IsClientSerialForEndpoint(actual, base string) bool {
	if actual == base {
		return true
	}
	if len(base) != 13 || len(actual) != 14 || actual[:3] != "MBC" || actual[4:] != base[3:] {
		return false
	}
	c := actual[3]
	return c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func normalizedClientSerial(actual, base string) string {
	actual = strings.TrimSpace(actual)
	if actual == "" {
		return base
	}
	return actual
}
