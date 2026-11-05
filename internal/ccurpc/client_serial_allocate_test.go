// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_serial_allocate_test.go
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

func TestAllocateClientSerialsCollisionAndOrder(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{
		{ID: "two", Enabled: true, Host: "10.0.3.207", Port: 502, UnitID: 1},
		{ID: "one", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1},
	}}
	first, changed, err := AllocateClientSerials(cfg)
	if err != nil || !changed {
		t.Fatalf("allocate: %v %v", changed, err)
	}
	if cfg.Devices[0].CCUAddress != "" {
		t.Fatal("modified original configuration")
	}
	ids := map[string]string{}
	for _, d := range first.Devices {
		ids[d.ID] = d.CCUAddress
	}
	if ids["one"] != "MBC03CF01F601" || ids["two"] != "MBCa03CF01F601" {
		t.Fatalf("allocation: %v", ids)
	}
	again, changed, err := AllocateClientSerials(first)
	if err != nil || changed {
		t.Fatalf("unstable reallocation changed=%v err=%v", changed, err)
	}
	again.Devices[0], again.Devices[1] = again.Devices[1], again.Devices[0]
	next, changed, err := AllocateClientSerials(again)
	if err != nil || changed {
		t.Fatalf("order-dependent allocation changed=%v err=%v", changed, err)
	}
	for _, d := range next.Devices {
		if ids[d.ID] != d.CCUAddress {
			t.Fatalf("device %s changed identity", d.ID)
		}
	}
}

func TestAllocateClientSerialsRespectsPreexistingIdentities(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{
		{ID: "existing", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1, CCUAddress: "MBCa03CF01F601"},
		{ID: "new", Enabled: true, Host: "10.0.3.207", Port: 502, UnitID: 1},
	}}
	result, changed, err := AllocateClientSerials(cfg)
	if err != nil || !changed {
		t.Fatalf("allocation %v %v", changed, err)
	}
	if result.Devices[0].CCUAddress != "MBCa03CF01F601" || result.Devices[1].CCUAddress != "MBC03CF01F601" {
		t.Fatalf("changed reserved identities %+v", result.Devices)
	}
}

func TestNewProductRejectsOldAddonSerial(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "legacy", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1, CCUAddress: "MB003207502001"}}}
	if _, _, err := AllocateClientSerials(cfg); err == nil {
		t.Fatal("legacy serial silently adopted")
	}
	if cfg.Devices[0].CCUAddress != "MB003207502001" {
		t.Fatal("legacy configuration modified")
	}
}
