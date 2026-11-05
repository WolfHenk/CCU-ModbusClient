// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        cmd/ccu-modbusclientd/config_identity_test.go
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

func TestClientSerialAssignmentIsPersistedAndCheckIsReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new-client.json")
	initial := []byte(`{"schema":2,"devices":[{"id":"two","name":"Second","enabled":true,"host":"10.0.3.207","port":502,"unit_id":1,"registers":[]},{"id":"one","name":"First","enabled":true,"host":"192.168.3.207","port":502,"unit_id":1,"registers":[]}]}`)
	if err := os.WriteFile(path, initial, 0600); err != nil {
		t.Fatal(err)
	}
	checked, err := loadClientConfig(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if checked.Devices[0].CCUAddress != "MBCa03CF01F601" || checked.Devices[1].CCUAddress != "MBC03CF01F601" {
		t.Fatalf("wrong addresses: %+v", checked.Devices)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(initial) {
		t.Fatal("check wrote configuration")
	}
	saved, err := loadClientConfig(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Devices[0].CCUAddress != "MBCa03CF01F601" || saved.Devices[1].CCUAddress != "MBC03CF01F601" {
		t.Fatalf("wrong stored addresses: %+v", saved.Devices)
	}
	reloaded, err := loadClientConfig(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Devices[0].CCUAddress != saved.Devices[0].CCUAddress || reloaded.Devices[1].CCUAddress != saved.Devices[1].CCUAddress {
		t.Fatal("serial changed on restart")
	}
}
func TestClientSerialRejectsForeignIdentityWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	original := []byte(`{"schema":2,"devices":[{"id":"legacy","name":"Legacy","enabled":true,"host":"192.168.3.207","port":502,"unit_id":1,"ccu_address":"MB003207502001","registers":[]}]}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadClientConfig(path, false); err == nil {
		t.Fatal("accepted foreign serial")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != string(original) {
		t.Fatal("modified foreign config")
	}
}
