// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/config/coil_direction_test.go
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

func TestCoilDirectionValidation(t *testing.T) {
	base := Register{ID: "c1", Name: "coil", Enabled: true, Type: "coil", Address: 1, DataType: "bool", PollSeconds: 3}
	for _, d := range []string{"", "input", "output"} {
		r := base
		r.CoilDirection = d
		if err := ValidateRegister(r); err != nil {
			t.Errorf("valid direction %q: %v", d, err)
		}
	}
	for _, d := range []string{"invalid", "Input"} {
		r := base
		r.CoilDirection = d
		if err := ValidateRegister(r); err == nil {
			t.Errorf("invalid direction %q accepted", d)
		}
	}
	r := base
	r.Type = "discrete"
	r.CoilDirection = "input"
	if err := ValidateRegister(r); err == nil {
		t.Error("non-coil direction accepted")
	}
}
