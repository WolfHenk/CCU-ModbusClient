// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/stable_channels_test.go
// Ersteller:    Wolfram Henkel
// Datum:        11. Nov. 2026
// Version:      0.3.7
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
	"testing"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
)

// Catalog, value snapshot and write routes use the stored channel numbers,
// including gaps (disabled ch 2, deleted ch 3) and a binary channel after a
// numeric one (regression test CCU 2026-11-11 18:19).
func TestStableChannelsEndToEnd(t *testing.T) {
	serial, err := ClientSerial("192.168.3.200", 502, 1)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Devices: []config.Device{{ID: "h", Enabled: true, Host: "192.168.3.200", Port: 502, UnitID: 1, CCUAddress: serial, NextChannel: 7, Registers: []config.Register{
		{ID: "coilA", Enabled: true, Type: "coil", DataType: "bool", Channel: 1},
		{ID: "temp", Enabled: true, Type: "input", DataType: "float32", Unit: "°C", Channel: 4},
		{ID: "coilB", Enabled: false, Type: "coil", DataType: "bool", Channel: 2},
		{ID: "diNew", Enabled: true, Type: "discrete", DataType: "bool", Channel: 5},
		{ID: "setp", Enabled: true, Type: "holding", DataType: "float32", NumericDirection: "output", Channel: 6},
	}}}}
	c, err := CatalogFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	devices, _ := c.ListDevices()
	if len(devices) != 6 { // device + :0 + 4 channels
		t.Fatalf("%d descriptions", len(devices))
	}
	for _, want := range []struct {
		n   int
		typ string
	}{{1, "SWITCH"}, {5, "DIGITAL_INPUT"}} {
		addr := serial + ":" + string(rune('0'+want.n))
		d, err := c.DeviceDescription(addr)
		if err != nil || d.Members["TYPE"].Text != want.typ || d.Members["INDEX"].Text != string(rune('0'+want.n)) {
			t.Fatalf("%s: %+v %v", addr, d.Members, err)
		}
	}
	for _, gone := range []string{":2", ":3"} {
		if _, err := c.DeviceDescription(serial + gone); err == nil {
			t.Fatalf("%s exists", gone)
		}
	}
	key4 := numericChannelFor(cfg.Devices[0].Registers[1]).Key
	snap := model.DeviceState{ID: "h", State: "ONLINE", Registers: []model.RegisterState{
		{ID: "diNew", Value: model.Value{Quality: model.QualityGood, Value: true}},
		{ID: "temp", Value: model.Value{Quality: model.QualityGood, Value: 21.5}},
	}}
	if err := c.ApplySnapshot(cfg, []model.DeviceState{snap}); err != nil {
		t.Fatal(err)
	}
	v5, _ := c.Paramset(serial+":5", "VALUES")
	v4, _ := c.Paramset(serial+":4", "VALUES")
	if v5.Members["STATE"].Text != "1" && v5.Members["STATE"].Text != "true" {
		t.Fatalf(":5 STATE %+v", v5.Members["STATE"])
	}
	if v4.Members[key4].Text != "21.5" {
		t.Fatalf(":4 %s %+v", key4, v4.Members[key4])
	}
	routes, err := buildWriteRoutes(cfg)
	if err != nil {
		t.Fatal(err)
	}
	key6 := numericChannelFor(cfg.Devices[0].Registers[4]).Key
	if r, ok := routes[serial+":1.STATE"]; !ok || r.register != "coilA" {
		t.Fatalf("coil route %v", routes)
	}
	if r, ok := routes[serial+":6."+key6]; !ok || r.register != "setp" {
		t.Fatalf("setpoint route %v", routes)
	}
	if len(routes) != 2 {
		t.Fatalf("routes %v", routes)
	}
}
