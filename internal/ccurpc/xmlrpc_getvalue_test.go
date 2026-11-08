// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/xmlrpc_getvalue_test.go
// Ersteller:    Wolfram Henkel
// Datum:        06. Nov. 2026
// Version:      0.3.1 (XML-RPC read correction)
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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
)

func TestXMLRPCGetValueUsesCurrentModbusCache(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		ID: "test", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1,
		Registers: []config.Register{
			{ID: "coil", Enabled: true, Type: "coil", DataType: "bool"},
			{ID: "discrete", Enabled: true, Type: "discrete", DataType: "bool"},
			{ID: "setpoint", Enabled: true, Type: "holding", DataType: "float32", NumericDirection: "output"},
			{ID: "measurement", Enabled: true, Type: "input", DataType: "uint16"},
			{ID: "counter", Enabled: true, Type: "input", DataType: "uint64"},
		},
	}}}
	cat, err := CatalogFromClientConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	mapped, err := ClientSnapshotConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	serial := mapped.Devices[0].CCUAddress
	state := model.DeviceState{ID: "test", State: "ONLINE", Registers: []model.RegisterState{
		{ID: "coil", Value: model.Value{Quality: model.QualityGood, Value: true}},
		{ID: "discrete", Value: model.Value{Quality: model.QualityGood, Value: false}},
		{ID: "setpoint", Value: model.Value{Quality: model.QualityGood, Value: 47.5}},
		{ID: "measurement", Value: model.Value{Quality: model.QualityGood, Value: 15.0}},
		{ID: "counter", Value: model.Value{Quality: model.QualityGood, Value: "18446744073709551615"}},
	}}
	if err := cat.ApplySnapshot(mapped, []model.DeviceState{state}); err != nil {
		t.Fatal(err)
	}
	h := NewHandler()
	if err := RegisterCCUReadMethods(h, cat); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/RPC3", h)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := Client{URL: server.URL + "/RPC3"}
	for _, c := range []struct{ addr, key, kind, text string }{
		{serial + ":1", "STATE", "boolean", "1"},
		{serial + ":2", "STATE", "boolean", "0"},
		{serial + ":3", "VALUE", "double", "47.5"},
		{serial + ":4", "VOLTAGE", "double", "15"},
		{serial + ":5", "VALUE", "string", "18446744073709551615"},
		{serial + ":0", "UNREACH", "boolean", "0"},
	} {
		got, err := client.Call(context.Background(), "getValue", String(c.addr), String(c.key))
		if err != nil || got.Kind != c.kind || got.Text != c.text {
			t.Errorf("getValue(%s,%s) = %+v, %v", c.addr, c.key, got, err)
		}
	}
	// Subsequent snapshots must be observed, not the initial cached value.
	state.Registers[2].Value.Value = 45.0
	if err := cat.ApplySnapshot(mapped, []model.DeviceState{state}); err != nil {
		t.Fatal(err)
	}
	got, err := client.Call(context.Background(), "getValue", String(serial+":3"), String("VALUE"))
	if err != nil || got.Kind != "double" || got.Text != "45" {
		t.Fatalf("updated value %+v %v", got, err)
	}
	for _, call := range []struct {
		args    []Value
		message string
	}{
		{[]Value{String(serial + ":0"), String("INSTALL_TEST")}, "not readable"},
		{[]Value{String(serial + ":99"), String("STATE")}, "unknown channel"},
		{[]Value{String(serial + ":3"), String("NOT_REAL")}, "unknown datapoint"},
		{[]Value{String(serial + ":3")}, "requires channel address"},
		{[]Value{Integer(3), String("VALUE")}, "requires channel address"},
	} {
		_, err := client.Call(context.Background(), "getValue", call.args...)
		if err == nil || !strings.Contains(err.Error(), call.message) {
			t.Errorf("getValue invalid: %v", err)
		}
	}
	if cfg.Devices[0].CCUAddress != "" {
		t.Fatal("read modified persistent identity")
	}
}

func TestXMLRPCFaultOnlyExposesKnownValidationMessages(t *testing.T) {
	h := NewHandler()
	if err := h.Register("known", func([]Value) (Value, error) { return Value{}, rpcInputError("datapoint is read-only") }); err != nil {
		t.Fatal(err)
	}
	if err := h.Register("internal", func([]Value) (Value, error) { return Value{}, errors.New("private password or internal address") }); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h)
	defer server.Close()
	client := Client{URL: server.URL}
	_, err := client.Call(context.Background(), "known")
	if err == nil || !strings.Contains(err.Error(), "datapoint is read-only") {
		t.Fatalf("public reason omitted: %v", err)
	}
	_, err = client.Call(context.Background(), "internal")
	if err == nil || !strings.Contains(err.Error(), "RPC method failed") || strings.Contains(err.Error(), "private password") {
		t.Fatalf("private fault disclosed: %v", err)
	}
}
