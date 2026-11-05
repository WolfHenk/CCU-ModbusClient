// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        cmd/ccu-modbusclient-config/main_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadWriteAndStableSerial(t *testing.T) {
	t.Setenv("CCU_MODBUSCLIENT_CONFIG_TEST", "1")
	path := filepath.Join(t.TempDir(), "config.json")
	original := `{"schema":2,"devices":[]}`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	body := `{"schema":2,"devices":[{"id":"test","name":"Test","enabled":true,"host":"192.168.3.207","port":502,"unit_id":1,"timeout_ms":1500,"registers":[]}]}`
	var out bytes.Buffer
	if err := run([]string{"write", path}, strings.NewReader(body), &out); err != nil {
		t.Fatal(err)
	}
	saved, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Devices[0].CCUAddress != "MBC03CF01F601" {
		t.Fatalf("identity: %q", saved.Devices[0].CCUAddress)
	}
	if saved.Listen != "127.0.0.1:18702" {
		t.Fatalf("wrong default port: %q", saved.Listen)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"write", path}, strings.NewReader("{invalid"), &out); err == nil {
		t.Fatal("invalid JSON accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("invalid write modified configuration")
	}
	changed := strings.ReplaceAll(string(before), `"host": "192.168.3.207"`, `"host": "192.168.4.207"`)
	if err := run([]string{"write", path}, strings.NewReader(changed), &out); err == nil {
		t.Fatal("identity-changing endpoint accepted")
	}
}
func TestRejectSymlinkConfigFile(t *testing.T) {
	t.Setenv("CCU_MODBUSCLIENT_CONFIG_TEST", "1")
	folder := t.TempDir()
	real := filepath.Join(folder, "real.json")
	original := []byte(`{"schema":2,"devices":[]}`)
	if err := os.WriteFile(real, original, 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(folder, "alias.json")
	if err := os.Symlink(real, alias); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"write", alias}, strings.NewReader(string(original)), &bytes.Buffer{}); err == nil {
		t.Fatal("symlink accepted")
	}
	after, err := os.ReadFile(real)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("redirected file modified")
	}
}
