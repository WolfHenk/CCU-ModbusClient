// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/install_point.go
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

// InstallTestPoint describes the legacy virtual-device channel action.
// The independent replacement currently exposes metadata only; execution of
// this action is not wired into production and requires separate verification.
func InstallTestPoint() Point {
	zero, one := Boolean(false), Boolean(true)
	return Point{ID: "INSTALL_TEST", Type: "ACTION", Value: zero, Write: true,
		Flags: 3, Default: &zero, Minimum: &zero, Maximum: &one}
}
