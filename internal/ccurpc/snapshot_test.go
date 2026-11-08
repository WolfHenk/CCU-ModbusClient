// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/snapshot_test.go
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

func TestSnapshotUpdatesAndRejectsBadQuality(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "dev", Enabled: true, CCUAddress: "MB001002502001", Registers: []config.Register{{ID: "input", Enabled: true, Type: "discrete", DataType: "bool", InvertInput: true}, {ID: "volts", Enabled: true, Type: "input", DataType: "float32"}}}}}
	c, e := CatalogFromConfig(cfg)
	if e != nil {
		t.Fatal(e)
	}
	st := model.DeviceState{ID: "dev", State: "OFFLINE", ModbusException: &model.ModbusException{Function: 3, Code: 2}, Registers: []model.RegisterState{{ID: "input", Value: model.Value{Quality: model.QualityGood, Value: true}}, {ID: "volts", Value: model.Value{Quality: model.QualityGood, Value: 12.5}}}}
	if e := c.ApplySnapshot(cfg, []model.DeviceState{st}); e != nil {
		t.Fatal(e)
	}
	for _, check := range []struct{ addr, key, want string }{{"MB001002502001:0", "UNREACH", "1"}, {"MB001002502001:0", "MODBUS_ERROR", "770"}, {"MB001002502001:1", "STATE", "0"}, {"MB001002502001:2", "VALUE", "12.5"}} {
		ps, e := c.Paramset(check.addr, "VALUES")
		if e != nil || ps.Members[check.key].Text != check.want {
			t.Fatalf("%s %s: %v %v", check.addr, check.key, ps, e)
		}
	}
	st.Registers[1].Value = model.Value{Quality: model.QualityReadError, Value: 99.0}
	if e := c.ApplySnapshot(cfg, []model.DeviceState{st}); e != nil {
		t.Fatal(e)
	}
	ps, _ := c.Paramset("MB001002502001:2", "VALUES")
	if ps.Members["VALUE"].Text != "12.5" {
		t.Fatal("bad quality overwrote last good value")
	}
}
