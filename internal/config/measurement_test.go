// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/config/measurement_test.go
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

func TestMeasurementAllowlist(t *testing.T) {
	r := Register{ID: "r", Name: "r", Enabled: true, Type: "input", DataType: "float32", PollSeconds: 10}
	for _, m := range []string{"", "generic", "number", "temperature", "humidity", "voltage", "current", "power", "energy", "frequency"} {
		r.Measurement = m
		if err := ValidateRegister(r); err != nil {
			t.Errorf("allowed %q: %v", m, err)
		}
	}
	for _, m := range []string{"window", "motion", "gas", "temperature_sensor"} {
		r.Measurement = m
		if err := ValidateRegister(r); err == nil {
			t.Errorf("accepted unsupported %q", m)
		}
	}
	r.Measurement = "temperature"
	r.Type = "discrete"
	r.DataType = "bool"
	if ValidateRegister(r) == nil {
		t.Fatal("accepted measurement on digital input")
	}
	r.Type = "input"
	r.DataType = "uint64"
	if ValidateRegister(r) == nil {
		t.Fatal("accepted 64-bit measurement without tested representation")
	}
}
