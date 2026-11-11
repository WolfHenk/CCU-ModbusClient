// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/events_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
	"testing"
)

func TestSnapshotEventChangesOnly(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "x", Enabled: true, CCUAddress: "MB001002502001", Registers: []config.Register{{ID: "di", Enabled: true, Type: "discrete", DataType: "bool"}}}}}
	c, err := CatalogFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	snap := model.DeviceState{ID: "x", State: "ONLINE", Registers: []model.RegisterState{{ID: "di", Value: model.Value{Quality: model.QualityGood, Value: true}}}}
	if err := c.ApplySnapshot(cfg, []model.DeviceState{snap}); err != nil {
		t.Fatal(err)
	}
	// The first update also publishes the maintenance channel once (its
	// first values); the data channel must carry the new state.
	ev := c.DrainEvents()
	var data []PointEvent
	for _, e := range ev {
		if e.Address != "MB001002502001:0" {
			data = append(data, e)
		}
	}
	if len(data) != 1 || data[0].Address != "MB001002502001:1" || data[0].Key != "STATE" || data[0].Value.Text != "1" {
		t.Fatalf("first update: %+v", ev)
	}
	if err := c.ApplySnapshot(cfg, []model.DeviceState{snap}); err != nil {
		t.Fatal(err)
	}
	if ev := c.DrainEvents(); len(ev) != 0 {
		t.Fatalf("unchanged value created events: %+v", ev)
	}
	snap.Registers[0].Value.Quality = model.QualityReadError
	snap.Registers[0].Value.Value = false
	if err := c.ApplySnapshot(cfg, []model.DeviceState{snap}); err != nil {
		t.Fatal(err)
	}
	if ev := c.DrainEvents(); len(ev) != 0 {
		t.Fatalf("bad-quality value created events: %+v", ev)
	}
	snap.Registers[0].Value.Quality = model.QualityGood
	if err := c.ApplySnapshot(cfg, []model.DeviceState{snap}); err != nil {
		t.Fatal(err)
	}
	ev = c.DrainEvents()
	if len(ev) != 1 || ev[0].Value.Text != "0" {
		t.Fatalf("changed value missing: %+v", ev)
	}
}
