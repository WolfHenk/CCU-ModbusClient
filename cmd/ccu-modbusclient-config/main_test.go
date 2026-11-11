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

func deviceJSON(regs ...string) string {
	return `{"schema":2,"devices":[{"id":"h","name":"Haus","enabled":true,"host":"192.168.3.200","port":502,"unit_id":1,"timeout_ms":1500,"registers":[` + strings.Join(regs, ",") + `]}]}`
}

func regJSON(id, typ, dt string, extra string) string {
	return `{"id":"` + id + `","name":"` + id + `","enabled":true,"type":"` + typ + `","address":0,"datatype":"` + dt + `","factor":1,"offset":0,"poll_seconds":10,"unit":""` + extra + `}`
}

func channelsOf(t *testing.T, path string) map[string]int {
	t.Helper()
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]int{}
	for _, r := range c.Devices[0].Registers {
		m[r.ID] = r.Channel
	}
	return m
}

// The page may drop or invent ccu_channel values: the saved file decides.
func TestWriteKeepsChannelsFromFile(t *testing.T) {
	t.Setenv("CCU_MODBUSCLIENT_CONFIG_TEST", "1")
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"schema":2,"devices":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	first := deviceJSON(regJSON("c1", "coil", "bool", ""), regJSON("t1", "input", "float32", ""))
	if err := run([]string{"write", path}, strings.NewReader(first), &out); err != nil {
		t.Fatal(err)
	}
	// numeric first now, bogus number for c1, a new discrete input appended
	second := deviceJSON(regJSON("t1", "input", "float32", `,"ccu_channel":1`), regJSON("c1", "coil", "bool", `,"ccu_channel":7`), regJSON("d1", "discrete", "bool", ""))
	if err := run([]string{"write", path}, strings.NewReader(second), &out); err != nil {
		t.Fatal(err)
	}
	if c := channelsOf(t, path); c["c1"] != 1 || c["t1"] != 2 || c["d1"] != 3 {
		t.Fatalf("channels %v", c)
	}
}

// Repair of the test CCU: the 42-register file without numbers would be
// migrated with the broken order; adopt-channels takes the numbers of the
// 36-register backup and appends the six new registers.
func TestAdoptChannelsFromBackup(t *testing.T) {
	t.Setenv("CCU_MODBUSCLIENT_CONFIG_TEST", "1")
	dir := t.TempDir()
	var regs36 []string
	for i := 0; i < 3; i++ {
		regs36 = append(regs36, regJSON("coil"+string(rune('0'+i)), "coil", "bool", ""))
	}
	for i := 0; i < 4; i++ {
		regs36 = append(regs36, regJSON("ir"+string(rune('0'+i)), "input", "float32", ""))
	}
	for i := 0; i < 29; i++ {
		regs36 = append(regs36, regJSON("hr"+strings.Repeat("x", i), "holding", "float32", ""))
	}
	regs42 := append(append([]string(nil), regs36...), regJSON("di37", "discrete", "bool", ""), regJSON("coil38", "coil", "bool", ""),
		regJSON("n39", "input", "float32", ""), regJSON("n40", "input", "float32", ""), regJSON("n41", "input", "int32", ""), regJSON("n42", "input", "float32", ""))
	ref := filepath.Join(dir, "before.json")
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(ref, []byte(deviceJSON(regs36...)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(deviceJSON(regs42...)), 0600); err != nil {
		t.Fatal(err)
	}
	if c := channelsOf(t, path); c["di37"] != 4 {
		t.Fatalf("precondition: plain migration should reproduce the broken order, got %v", c["di37"])
	}
	var out bytes.Buffer
	if err := run([]string{"adopt-channels", path, ref}, nil, &out); err != nil {
		t.Fatal(err)
	}
	c := channelsOf(t, path)
	if c["ir0"] != 4 || c["hr"] != 8 || c["di37"] != 37 || c["coil38"] != 38 || c["n42"] != 42 {
		t.Fatalf("after adopt %v", c)
	}
	if !strings.Contains(out.String(), `"register":"di37","name":"di37","from":4,"to":37`) {
		t.Fatalf("change report %s", out.String())
	}
	// written for good: a later plain write keeps the repaired numbers
	if err := run([]string{"write", path}, strings.NewReader(deviceJSON(regs42...)), &out); err != nil {
		t.Fatal(err)
	}
	if c := channelsOf(t, path); c["ir0"] != 4 || c["di37"] != 37 {
		t.Fatalf("after write %v", c)
	}
}
