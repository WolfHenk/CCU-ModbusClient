// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/config/numeric_direction_test.go
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

package config

import "testing"

func TestNumericDirectionRequiresExplicitWritableHolding(t *testing.T) {
	base := Register{ID: "setpoint", Name: "Sollwert", Enabled: true, Type: "holding", DataType: "int16", Address: 100, Factor: 1, PollSeconds: 10}
	if err := ValidateRegister(base); err != nil {
		t.Fatalf("legacy input: %v", err)
	}
	base.NumericDirection = "input"
	if err := ValidateRegister(base); err != nil {
		t.Fatalf("explicit input: %v", err)
	}
	base.NumericDirection = "output"
	if err := ValidateRegister(base); err != nil {
		t.Fatalf("output: %v", err)
	}
	for _, tc := range []struct {
		typ, dir, datatype string
		factor             float64
	}{
		{"input", "output", "int16", 1},
		{"holding", "unknown", "int16", 1},
		{"holding", "output", "int16", 0},
		{"holding", "output", "int64", 1},
		{"holding", "output", "bool", 1},
	} {
		r := base
		r.Type = tc.typ
		r.NumericDirection = tc.dir
		r.DataType = tc.datatype
		r.Factor = tc.factor
		if err := ValidateRegister(r); err == nil {
			t.Errorf("accepted invalid %+v", r)
		}
	}
}
