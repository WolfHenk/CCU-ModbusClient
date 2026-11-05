// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        cmd/ccu-modbusclientd/config_path_test.go
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
	"os"
	"path/filepath"
	"testing"
)

func TestClientConfigPathIsolation(t *testing.T) {
	for _, bad := range []string{"/usr/local/etc/config/addons/ccu-modbus/config.json", "/usr/local/etc/config/addons/ccu-modbus/../ccu-modbus/config.json", "/usr/local/etc/config/addons/other/config.json", "relative.json"} {
		if err := validateClientConfigPath(bad); err == nil {
			t.Fatalf("accepted foreign config %q", bad)
		}
	}
	for _, valid := range []string{"/usr/local/etc/config/addons/ccu-modbusclient/config.json", filepath.Join(t.TempDir(), "client.json")} {
		if err := validateClientConfigPath(valid); err != nil {
			t.Fatal(err)
		}
	}
}

func TestClientConfigPathRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	foreign := filepath.Join(root, "foreign")
	if err := os.Mkdir(foreign, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "redirect")
	if err := os.Symlink(foreign, link); err != nil {
		t.Fatal(err)
	}
	if err := validateClientConfigPath(filepath.Join(link, "client.json")); err == nil {
		t.Fatal("symlink directory accepted")
	}
	target := filepath.Join(root, "target.json")
	if err := os.WriteFile(target, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	fileLink := filepath.Join(root, "client.json")
	if err := os.Symlink(target, fileLink); err != nil {
		t.Fatal(err)
	}
	if err := validateClientConfigPath(fileLink); err == nil {
		t.Fatal("symlink file accepted")
	}
}
