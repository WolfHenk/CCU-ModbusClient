// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        cmd/ccu-modbusclientd/main_test.go
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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckOnlyAndRejectMissingEndpoint(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "client.json")
	data := `{"schema":2,"devices":[{"id":"a","name":"Test","enabled":true,"host":"192.168.3.207","port":502,"unit_id":1,"timeout_ms":1500,"retries":0,"registers":[]}]}`
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), file, "", true, 32220, 32229); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), file, "", false, 32220, 32229)
	if err == nil || !strings.Contains(err.Error(), "endpoint required") {
		t.Fatalf("unsafe start accepted: %v", err)
	}
}
func TestLegacySerialRequiresExplicitMigration(t *testing.T) {
	file := filepath.Join(t.TempDir(), "old.json")
	data := `{"schema":2,"devices":[{"id":"a","name":"Old","enabled":true,"host":"192.168.3.207","port":502,"unit_id":1,"timeout_ms":1500,"ccu_address":"MB003207502001","registers":[]}]}`
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), file, "", true, 32220, 32229)
	if err == nil || !strings.Contains(err.Error(), "foreign or incompatible CCU identity") {
		t.Fatalf("legacy serial was accepted: %v", err)
	}
}
