// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/catalog_test.go
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

import "testing"

func TestMemoryCatalogStableIDsAndAccess(t *testing.T) {
	c := NewMemoryCatalog()
	d := Device{Address: "MB007089505003", Type: "Modbus", Channels: []Channel{{Address: "MB007089505003:0", Type: "MAINTENANCE", Points: []Point{{ID: "UNREACH", Type: "BOOL", Value: Boolean(false), Read: true, Event: true}}}, {Address: "MB007089505003:1", Type: "MODBUS_NUMERIC_OUTPUT", Points: []Point{{ID: "VALUE", Type: "FLOAT", Unit: "V", Value: Value{Kind: "double", Text: "42.0"}, Read: true, Write: true, Event: true}}}}}
	if err := c.AddDevice(d); err != nil {
		t.Fatal(err)
	}
	if err := c.AddDevice(d); err == nil {
		t.Fatal("duplicate accepted")
	}
	got, err := c.ListDevices()
	if err != nil || len(got) != 3 || got[2].Members["ADDRESS"].Text != "MB007089505003:1" {
		t.Fatalf("unexpected addresses %+v %v", got, err)
	}
	desc, err := c.ParamsetDescription("MB007089505003:1", "VALUES")
	if err != nil {
		t.Fatal(err)
	}
	if desc.Members["VALUE"].Members["OPERATIONS"].Text != "7" {
		t.Fatalf("incorrect access flags %+v", desc)
	}
	values, err := c.Paramset("MB007089505003:0", "VALUES")
	if err != nil || values.Members["UNREACH"].Text != "0" {
		t.Fatalf("unexpected service value %+v %v", values, err)
	}
	if master, err := c.Paramset("MB007089505003:1", "MASTER"); err != nil || len(master.Members) != 0 {
		t.Fatalf("empty master expected: %+v %v", master, err)
	}
	if _, err := c.Paramset("MB007089505003:1", "BAD"); err == nil {
		t.Fatal("invalid paramset accepted")
	}
}
func TestCatalogRejectsChannelIdentityCollision(t *testing.T) {
	c := NewMemoryCatalog()
	if e := c.AddDevice(Device{Address: "MB1", Type: "Modbus", Channels: []Channel{{Address: "MB2:1", Type: "DIGITAL_INPUT"}}}); e == nil {
		t.Fatal("foreign channel accepted")
	}
}
