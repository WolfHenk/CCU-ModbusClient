// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/model/exception_codes.go
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

package model

import "fmt"

// ExceptionCodeLabel uses the standard code table in MODBUS Application Protocol
// Specification V1.1b3, section 7 (exception responses), table on page 48.
// Gaps are deliberately not invented or given standardized meanings.
func ExceptionCodeLabel(code byte) string {
	switch code {
	case 0x01:
		return "Illegal Function"
	case 0x02:
		return "Illegal Data Address"
	case 0x03:
		return "Illegal Data Value"
	case 0x04:
		return "Server Device Failure"
	case 0x05:
		return "Acknowledge"
	case 0x06:
		return "Server Device Busy"
	case 0x08:
		return "Memory Parity Error"
	case 0x0A:
		return "Gateway Path Unavailable"
	case 0x0B:
		return "Gateway Target Device Failed to Respond"
	default:
		return ""
	}
}

// ExceptionMessage preserves function and code in their original on-wire
// representation. For example a holding-register exception is "83 02 ...".
// Unknown or manufacturer-specific codes are retained, never silently lost.
func ExceptionMessage(function, code byte) string {
	name := ExceptionCodeLabel(code)
	if name == "" {
		name = "Unknown/Manufacturer-specific Exception"
	}
	return fmt.Sprintf("%02X %02X %s", function|0x80, code, name)
}
