// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/config/config_atomic_test.go
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

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicDoesNotFollowPredictableTempOrBackupSymlinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("DO NOT CHANGE"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{path + ".new", path + ".bak"} {
		if err := os.Symlink(victim, p); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &Config{Schema: 2, Devices: []Device{}}
	if err := WriteAtomic(path, cfg); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(path, cfg); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "DO NOT CHANGE" {
		t.Fatalf("symlink target was modified: %q", b)
	}
	if info, err := os.Lstat(path + ".bak"); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("backup not regular file: %v %v", info, err)
	}
}
