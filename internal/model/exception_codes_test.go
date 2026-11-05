// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/model/exception_codes_test.go
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

import (
	"strings"
	"testing"
)

func TestStandardExceptionCodesV11b3(t *testing.T) {
	expected := map[byte]string{
		0x01: "Illegal Function",
		0x02: "Illegal Data Address",
		0x03: "Illegal Data Value",
		0x04: "Server Device Failure",
		0x05: "Acknowledge",
		0x06: "Server Device Busy",
		0x08: "Memory Parity Error",
		0x0A: "Gateway Path Unavailable",
		0x0B: "Gateway Target Device Failed to Respond",
	}
	for code, name := range expected {
		if got := ExceptionCodeLabel(code); got != name {
			t.Errorf("code %02X: got %q, want %q", code, got, name)
		}
	}
	// These values are not assigned by the current specification.
	for _, code := range []byte{0x00, 0x07, 0x09, 0x0C, 0xFF} {
		if got := ExceptionCodeLabel(code); got != "" {
			t.Errorf("unassigned code %02X incorrectly labelled %q", code, got)
		}
		if got := ExceptionMessage(0x03, code); !strings.HasPrefix(got, "83 ") || !strings.Contains(got, "Unknown/Manufacturer-specific") {
			t.Errorf("unassigned code %02X not preserved: %q", code, got)
		}
	}
}

func TestExceptionMessageRetainsOriginalFunctionAndCode(t *testing.T) {
	for _, tc := range []struct {
		function, code byte
		want           string
	}{
		{0x01, 0x01, "81 01 Illegal Function"},
		{0x02, 0x06, "82 06 Server Device Busy"},
		{0x03, 0x02, "83 02 Illegal Data Address"},
		{0x04, 0x0B, "84 0B Gateway Target Device Failed to Respond"},
		{0x05, 0x04, "85 04 Server Device Failure"},
		{0x10, 0x0A, "90 0A Gateway Path Unavailable"},
	} {
		if got := ExceptionMessage(tc.function, tc.code); got != tc.want {
			t.Errorf("FC%02X Code%02X got %q, want %q", tc.function, tc.code, got, tc.want)
		}
	}
}
