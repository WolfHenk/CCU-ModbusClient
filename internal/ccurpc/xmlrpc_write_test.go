// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/xmlrpc_write_test.go
// Ersteller:    Wolfram Henkel
// Datum:        06. Nov. 2026
// Version:      0.3.1 (XML-RPC write correction)
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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
)

type xmlWriteSpy struct {
	coils   []bool
	numbers []float64
	reject  bool
}

func (w *xmlWriteSpy) EnqueueCoil(device, register string, state bool) bool {
	if w.reject || device != "d" || register != "coil" {
		return false
	}
	w.coils = append(w.coils, state)
	return true
}
func (w *xmlWriteSpy) EnqueueNumeric(device, register string, value float64) bool {
	if w.reject || device != "d" || register != "out" {
		return false
	}
	w.numbers = append(w.numbers, value)
	return true
}

func TestXMLRPCSetValueCCUPathAndGuards(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		ID: "d", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1,
		Registers: []config.Register{
			{ID: "coil", Enabled: true, Type: "coil", DataType: "bool"},
			{ID: "discrete", Enabled: true, Type: "discrete", DataType: "bool"},
			{ID: "coilInput", Enabled: true, Type: "coil", DataType: "bool", CoilDirection: "input"},
			{ID: "out", Enabled: true, Type: "holding", DataType: "float32", NumericDirection: "output"},
			{ID: "holdInput", Enabled: true, Type: "holding", DataType: "float32", NumericDirection: "input"},
			{ID: "int64", Enabled: true, Type: "holding", DataType: "int64", NumericDirection: "output"},
			{ID: "input", Enabled: true, Type: "input", DataType: "float32"},
			{ID: "off", Enabled: false, Type: "coil", DataType: "bool"},
		},
	}}}
	spy := new(xmlWriteSpy)
	handler := NewHandler()
	if err := RegisterXMLRPCWrites(handler, cfg, spy); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/RPC3", handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := Client{URL: srv.URL + "/RPC3"}
	ctx := context.Background()
	serial, err := ClientSerial("192.168.3.207", 502, 1)
	if err != nil {
		t.Fatal(err)
	}
	coil := serial + ":1"
	numeric := serial + ":4"
	call := func(address, point string, value Value) (Value, error) {
		return client.Call(ctx, "setValue", String(address), String(point), value)
	}
	for _, state := range []bool{true, false} {
		resp, err := call(coil, "STATE", Boolean(state))
		if err != nil || resp.Kind != "string" || resp.Text != "" {
			t.Fatalf("coil state %v response %+v err=%v", state, resp, err)
		}
	}
	for _, v := range []Value{
		{Kind: "double", Text: "47.5"},
		Integer(45),
		String("47.5"),
	} {
		resp, err := call(numeric, "VALUE", v)
		if err != nil || resp.Kind != "string" || resp.Text != "" {
			t.Fatalf("numeric %+v response %+v err=%v", v, resp, err)
		}
	}
	if len(spy.coils) != 2 || !spy.coils[0] || spy.coils[1] {
		t.Fatalf("wrong coil enqueues: %v", spy.coils)
	}
	if len(spy.numbers) != 3 || spy.numbers[0] != 47.5 || spy.numbers[1] != 45 || spy.numbers[2] != 47.5 {
		t.Fatalf("wrong numeric enqueues: %v", spy.numbers)
	}
	denied := []struct {
		name, address, key string
		value              Value
	}{
		{"read-only discrete", serial + ":2", "STATE", Boolean(true)},
		{"read-only coil", serial + ":3", "STATE", Boolean(true)},
		{"read-only holding", serial + ":5", "VALUE", Integer(4)},
		{"unsupported 64-bit write", serial + ":6", "VALUE", Integer(4)},
		{"read-only input", serial + ":7", "VALUE", Integer(4)},
		{"no disabled channel", serial + ":8", "STATE", Boolean(true)},
		{"unknown address", serial + ":99", "VALUE", Integer(4)},
		{"wrong datapoint", numeric, "STATE", Boolean(true)},
		{"wrong coil type", coil, "STATE", String("true")},
		{"wrong numeric type", numeric, "VALUE", Boolean(true)},
		{"invalid numeric", numeric, "VALUE", String("47,5")},
		{"NaN", numeric, "VALUE", Value{Kind: "double", Text: "NaN"}},
		{"infinity", numeric, "VALUE", Value{Kind: "double", Text: "+Inf"}},
		{"invalid boolean", coil, "STATE", Value{Kind: "boolean", Text: "wrong"}},
	}
	for _, test := range denied {
		if _, err := call(test.address, test.key, test.value); err == nil {
			t.Errorf("%s unexpectedly accepted", test.name)
		}
	}
	for _, check := range []struct {
		address, key string
		value        Value
		message      string
	}{
		{serial + ":2", "STATE", Boolean(true), "datapoint is read-only"},
		{serial + ":99", "STATE", Boolean(true), "unknown datapoint"},
		{coil, "STATE", String("true"), "STATE requires boolean"},
		{numeric, "VALUE", Boolean(true), "VALUE requires numeric input"},
		{numeric, "VALUE", String("47,5"), "invalid numeric VALUE"},
		{numeric, "VALUE", Value{Kind: "double", Text: "NaN"}, "numeric VALUE must be finite"},
	} {
		_, err := call(check.address, check.key, check.value)
		if err == nil || !strings.Contains(err.Error(), check.message) {
			t.Errorf("wanted %q, got %v", check.message, err)
		}
	}
	if len(spy.coils) != 2 || len(spy.numbers) != 3 {
		t.Fatalf("invalid calls changed write queue: %+v", spy)
	}
	spy.reject = true
	if _, err := call(numeric, "VALUE", Value{Kind: "double", Text: "47.5"}); err == nil {
		t.Error("engine queue refusal was not returned as XML-RPC fault")
	}
	if _, err := client.Call(ctx, "setValue", String(coil)); err == nil {
		t.Error("missing arguments unexpectedly accepted")
	}
}
