// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/instance_lock_test.go
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

package ccurpc

import (
	"path/filepath"
	"testing"
)

func TestExclusiveClientInstanceLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "modbusclient.lock")
	first, err := lockClientInstance(path)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := lockClientInstance(path); err == nil {
		unlockClientInstance(second)
		t.Fatal("second client instance accepted")
	}
	unlockClientInstance(first)
	third, err := lockClientInstance(path)
	if err != nil {
		t.Fatalf("cannot restart after release: %v", err)
	}
	unlockClientInstance(third)
}
