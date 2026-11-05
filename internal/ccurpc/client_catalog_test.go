// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_catalog_test.go
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
	"strings"
	"testing"
)

func TestClientCatalogIdentity(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "new", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1}}}
	cat, err := CatalogFromClientConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Devices[0].CCUAddress != "" {
		t.Fatal("mutated persistent configuration")
	}
	if _, err := cat.DeviceDescription("MBC03CF01F601"); err != nil {
		t.Fatal(err)
	}
	cfg.Devices[0].CCUAddress = "MB003207502001"
	if _, err := CatalogFromClientConfig(cfg); err == nil || !strings.Contains(err.Error(), "foreign") {
		t.Fatal("legacy identity was silently migrated", err)
	}
	cfg.Devices[0].CCUAddress = "MBC03CF01F601"
	if _, err := CatalogFromClientConfig(cfg); err != nil {
		t.Fatal(err)
	}
}
func TestClientCatalogRejectsCollisions(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "a", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1}, {ID: "b", Enabled: true, Host: "10.0.3.207", Port: 502, UnitID: 1}}}
	if _, err := CatalogFromClientConfig(cfg); err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatal("expected truncated IPv4 collision", err)
	}
}
