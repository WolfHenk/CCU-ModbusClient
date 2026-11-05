// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        cmd/ccu-modbusclient-config/test_actions_test.go
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

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/WolfHenk/CCU-ModbusClient/internal/engine"
)

func TestConnectionActionReturnsJSONWithoutWritingConfig(t *testing.T) {
	input := `{"id":"test","name":"Testgerät","enabled":true,"host":"127.0.0.1","port":1,"unit_id":1,"timeout_ms":100,"retries":0,"registers":[]}`
	var out bytes.Buffer
	if err := run([]string{"test_connection"}, strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	var result engine.TestResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Error == "" {
		t.Fatalf("unexpected connection result: %+v", result)
	}
}
func TestRegisterActionRejects64BitOverflowWithoutNetwork(t *testing.T) {
	input := `{"device":{"id":"test","name":"Testgerät","enabled":true,"host":"127.0.0.1","port":502,"unit_id":1,"timeout_ms":100,"retries":0,"registers":[]},"register":{"id":"r","name":"64Bit","enabled":true,"type":"input","address":65535,"datatype":"uint64","factor":1,"poll_seconds":10}}`
	var out bytes.Buffer
	if err := run([]string{"test_register"}, strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	var result engine.TestResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.OK || !strings.Contains(result.Error, "65535") {
		t.Fatalf("64-bit address overflow was not rejected: %+v", result)
	}
}
func TestTestActionsRejectBadJSON(t *testing.T) {
	for _, action := range []string{"test_connection", "test_register"} {
		var out bytes.Buffer
		if err := run([]string{action}, strings.NewReader("{bad"), &out); err == nil {
			t.Fatalf("%s accepted malformed JSON", action)
		}
	}
}
