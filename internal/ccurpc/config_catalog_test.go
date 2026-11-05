// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/config_catalog_test.go
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

func TestCatalogFromConfigPreservesChannelOrder(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "sample", Enabled: true, CCUAddress: "MB007089505003", Registers: []config.Register{
		{ID: "numfirst", Enabled: true, Type: "holding", DataType: "int16", Unit: "V", NumericDirection: "output"},
		{ID: "disabled", Enabled: false, Type: "coil", DataType: "bool"},
		{ID: "digital", Enabled: true, Type: "discrete", DataType: "bool"},
		{ID: "legacy", Enabled: true, Type: "input", DataType: "uint16"},
	}}}}
	c, e := CatalogFromConfig(cfg)
	if e != nil {
		t.Fatal(e)
	}
	devices, e := c.ListDevices()
	if e != nil || len(devices) != 5 {
		t.Fatalf("device count=%d error=%v", len(devices), e)
	}
	checks := []struct{ addr, typ, key string }{
		{"MB007089505003:0", "MAINTENANCE", "UNREACH"},
		{"MB007089505003:1", "DIGITAL_INPUT", "STATE"},
		{"MB007089505003:2", "MODBUS_NUMERIC_OUTPUT", "VALUE"},
		{"MB007089505003:3", "ANALOG_INPUT_TRANSMITTER", "VOLTAGE"},
	}
	for _, want := range checks {
		d, e := c.DeviceDescription(want.addr)
		if e != nil || d.Members["TYPE"].Text != want.typ {
			t.Fatalf("%s type: %+v %v", want.addr, d, e)
		}
		p, e := c.ParamsetDescription(want.addr, "VALUES")
		if e != nil || p.Members[want.key].Kind != "struct" {
			t.Fatalf("%s datapoint: %+v %v", want.addr, p, e)
		}
	}
	p, _ := c.ParamsetDescription("MB007089505003:2", "VALUES")
	if p.Members["VALUE"].Members["OPERATIONS"].Text != "7" {
		t.Fatal("numeric output not writable")
	}
}
func TestCatalogFromConfigRefusesImplicitAddress(t *testing.T) {
	_, e := CatalogFromConfig(&config.Config{Devices: []config.Device{{ID: "unresolved", Enabled: true}}})
	if e == nil {
		t.Fatal("accepted implicit identity")
	}
}
