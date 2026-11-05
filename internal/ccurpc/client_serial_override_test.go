// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_serial_override_test.go
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

func TestStoredCollisionPrefixPreservesDistinctIdentities(t *testing.T) {
	base := "MBC03CF01F601"
	cfg := &config.Config{Devices: []config.Device{
		{ID: "first", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1, CCUAddress: base},
		{ID: "second", Enabled: true, Host: "10.0.3.207", Port: 502, UnitID: 1, CCUAddress: "MBCa03CF01F601"},
	}}
	for n := 0; n < 2; n++ {
		cat, err := CatalogFromClientConfig(cfg)
		if err != nil {
			t.Fatal(err)
		}
		for _, serial := range []string{base, "MBCa03CF01F601"} {
			if _, err := cat.DeviceDescription(serial); err != nil {
				t.Fatal(err)
			}
		}
		mapped, err := ClientSnapshotConfig(cfg)
		if err != nil {
			t.Fatal(err)
		}
		byID := map[string]string{}
		for _, d := range mapped.Devices {
			byID[d.ID] = d.CCUAddress
		}
		if byID["first"] != base || byID["second"] != "MBCa03CF01F601" {
			t.Fatalf("identity changed: %+v", mapped.Devices)
		}
		cfg.Devices[0], cfg.Devices[1] = cfg.Devices[1], cfg.Devices[0]
	}
}

func TestCollisionPrefixValidation(t *testing.T) {
	base := "MBC03CF01F601"
	for _, valid := range []string{base, "MBCa03CF01F601", "MBCX03CF01F601", "MBC903CF01F601"} {
		if !IsClientSerialForEndpoint(valid, base) {
			t.Errorf("rejected %s", valid)
		}
	}
	for _, bad := range []string{"aMBC03CF01F601", "AMBC03CF01F601", "MBC!03CF01F601", "AMBC03CF01F602", "AMBCa03CF01F601", "MB003207502001"} {
		if IsClientSerialForEndpoint(bad, base) {
			t.Errorf("accepted %s", bad)
		}
	}
	cfg := &config.Config{Devices: []config.Device{
		{ID: "first", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1, CCUAddress: "MBCa03CF01F601"},
		{ID: "second", Enabled: true, Host: "10.0.3.207", Port: 502, UnitID: 1, CCUAddress: "MBCa03CF01F601"},
	}}
	if _, err := CatalogFromClientConfig(cfg); err == nil {
		t.Fatal("accepted duplicate prefixed serial")
	}
}
