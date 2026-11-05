// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_type_serial_contract_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"testing"
)

func TestClientTypeAndSerialContract(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "client", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1}}}
	cat, err := CatalogFromClientConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	serial := "MBC03CF01F601"
	dev, err := cat.DeviceDescription(serial)
	if err != nil {
		t.Fatal(err)
	}
	if got := dev.Members["TYPE"].Text; got != "Modbus" {
		t.Fatalf("new client device type %q, want Modbus", got)
	}
	ch, err := cat.DeviceDescription(serial + ":0")
	if err != nil {
		t.Fatal(err)
	}
	if got := ch.Members["PARENT_TYPE"].Text; got != "Modbus" {
		t.Fatalf("channel parent type %q", got)
	}
	if cfg.Devices[0].CCUAddress != "" {
		t.Fatal("serial generation mutated original config")
	}
}

func TestClientSerialFirstOctetCollisionIsRejected(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{
		{ID: "one", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1},
		{ID: "two", Enabled: true, Host: "10.0.3.207", Port: 502, UnitID: 1},
	}}
	if _, err := CatalogFromClientConfig(cfg); err == nil {
		t.Fatal("truncated IP collision not rejected")
	}
}
