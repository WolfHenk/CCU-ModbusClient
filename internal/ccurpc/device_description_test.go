// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/device_description_test.go
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

func TestDeviceChannelRelationships(t *testing.T) {
	c := NewMemoryCatalog()
	d := Device{Address: "MB003203504001", Type: "Modbus", Channels: []Channel{{Address: "MB003203504001:0", Type: "MAINTENANCE"}, {Address: "MB003203504001:1", Type: "DIGITAL_INPUT"}}}
	if e := c.AddDevice(d); e != nil {
		t.Fatal(e)
	}
	device, e := c.DeviceDescription(d.Address)
	if e != nil {
		t.Fatal(e)
	}
	children := device.Members["CHILDREN"]
	if len(children.Array) != 2 || children.Array[0].Text != "MB003203504001:0" || children.Array[1].Text != "MB003203504001:1" {
		t.Fatalf("children %+v", children)
	}
	channel, e := c.DeviceDescription(d.Channels[1].Address)
	if e != nil {
		t.Fatal(e)
	}
	if channel.Members["PARENT"].Text != d.Address || channel.Members["PARENT_TYPE"].Text != "Modbus" || channel.Members["INDEX"].Text != "1" {
		t.Fatalf("channel metadata %+v", channel)
	}
	ps := channel.Members["PARAMSETS"]
	if len(ps.Array) != 2 || ps.Array[0].Text != "MASTER" || ps.Array[1].Text != "VALUES" {
		t.Fatalf("paramsets %+v", ps)
	}
}

func TestMaintenanceVersionFromConfiguration(t *testing.T) {
	c, err := CatalogFromConfig(&config.Config{Devices: []config.Device{{ID: "d", Enabled: true, CCUAddress: "MB003203504001"}}})
	if err != nil {
		t.Fatal(err)
	}
	ch, err := c.DeviceDescription("MB003203504001:0")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Members["VERSION"].Text != "3" {
		t.Fatalf("maintenance VERSION: %+v", ch)
	}
}
