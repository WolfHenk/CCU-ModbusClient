// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/compatibility_test.go
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

// The existing ccuvirtual integration uses VALUE, not VOLTAGE, on 64-bit
// string channels; this test prevents identity-changing regressions.
func Test64BitPointIdentityMatchesExistingCCUVirtual(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "large", Enabled: true, CCUAddress: "MB003207502001", Registers: []config.Register{{ID: "max", Enabled: true, Type: "input", DataType: "uint64"}}}}}
	c, err := CatalogFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.ParamsetDescription("MB003207502001:1", "VALUES")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.Members["VALUE"]; !ok {
		t.Fatalf("uint64 missing VALUE: %+v", p)
	}
	if _, ok := p.Members["VOLTAGE"]; ok {
		t.Fatal("legacy VOLTAGE incorrectly announced for uint64")
	}
	state := model.DeviceState{ID: "large", Registers: []model.RegisterState{{ID: "max", Value: model.Value{Quality: model.QualityGood, Value: "18446744073709551615"}}}}
	if err := c.ApplySnapshot(cfg, []model.DeviceState{state}); err != nil {
		t.Fatal(err)
	}
	values, _ := c.Paramset("MB003207502001:1", "VALUES")
	if values.Members["VALUE"].Text != "18446744073709551615" {
		t.Fatalf("wrong uint64: %+v", values)
	}
}
