// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/output_display_range_test.go
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

import (
	"math"
	"testing"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
)

func TestOutputDisplayRange(t *testing.T) {
	cases := []struct {
		name, typ              string
		factor, offset, lo, hi float64
		ok                     bool
	}{
		{"unsigned16", "uint16", 2, 10, 10, 131080, true},
		{"negative-scale", "int16", -2, 10, -65524, 65546, true},
		{"unsigned8", "uint8", 1, 0, 0, 255, true},
		{"signed32", "int32", 1, 0, -2147483648, 2147483647, true},
		{"zero-scale", "uint16", 0, 10, 10, 10, true},
		{"float", "float32", 1, 0, 0, 0, false},
		{"nonfinite", "uint16", math.Inf(1), 0, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lo, hi, ok := outputDisplayRange(config.Register{DataType: tc.typ, Factor: tc.factor, Offset: tc.offset})
			if ok != tc.ok || (ok && (lo != tc.lo || hi != tc.hi)) {
				t.Fatalf("got lo=%v hi=%v ok=%v", lo, hi, ok)
			}
		})
	}
}
