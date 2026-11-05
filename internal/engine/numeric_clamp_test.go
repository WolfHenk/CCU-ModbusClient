// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/numeric_clamp_test.go
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

package engine

import (
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"testing"
)

func TestClampNumericOutput(t *testing.T) {
	for _, tc := range []struct {
		typ      string
		in, want float64
	}{
		{"int16", -40000, -32768}, {"int16", 40000, 32767},
		{"uint16", -2, 0}, {"uint16", 80000, 65535},
		{"int32", -3000000000, -2147483648}, {"int32", 3000000000, 2147483647},
		{"uint32", -1, 0}, {"uint32", 5000000000, 4294967295},
	} {
		r := config.Register{DataType: tc.typ, Factor: 1}
		got, clamped := ClampNumericOutput(r, tc.in)
		if !clamped || got != tc.want {
			t.Errorf("%s %v => %v (%t), want %v", tc.typ, tc.in, got, clamped, tc.want)
		}
		if _, err := encodeNumericWrite(r, got); err != nil {
			t.Errorf("%s %v invalid after clamp: %v", tc.typ, got, err)
		}
	}
	r := config.Register{DataType: "uint16", Factor: 2, Offset: 5}
	if got, _ := ClampNumericOutput(r, -4); got != 5 {
		t.Errorf("scaled clamp %v", got)
	}
	if got, clamped := ClampNumericOutput(r, 7); got != 7 || clamped {
		t.Errorf("unneeded clamp %v %t", got, clamped)
	}
}
