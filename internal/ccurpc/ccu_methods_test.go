// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/ccu_methods_test.go
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
	"context"
	"net/http/httptest"
	"testing"
)

type fakeCatalog struct{}

func (fakeCatalog) ListDevices() ([]Value, error) {
	return []Value{Struct(map[string]Value{"ADDRESS": String("MB001:0"), "TYPE": String("MAINTENANCE")})}, nil
}
func (fakeCatalog) DeviceDescription(address string) (Value, error) {
	return Struct(map[string]Value{"ADDRESS": String(address)}), nil
}
func (fakeCatalog) ParamsetDescription(address, set string) (Value, error) {
	return Struct(map[string]Value{"VALUE": Struct(map[string]Value{"TYPE": String("FLOAT")})}), nil
}
func (fakeCatalog) Paramset(address, set string) (Value, error) {
	return Struct(map[string]Value{"VALUE": String("42")}), nil
}
func TestCCUReadMethodDispatch(t *testing.T) {
	h := NewHandler()
	if err := RegisterCCUReadMethods(h, fakeCatalog{}); err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(h)
	defer s.Close()
	c := Client{URL: s.URL}
	devices, err := c.Call(context.Background(), "listDevices")
	if err != nil || len(devices.Array) != 1 || devices.Array[0].Members["ADDRESS"].Text != "MB001:0" {
		t.Fatalf("devices %+v %v", devices, err)
	}
	desc, err := c.Call(context.Background(), "getDeviceDescription", String("MB001:0"))
	if err != nil || desc.Members["ADDRESS"].Text != "MB001:0" {
		t.Fatalf("description %+v %v", desc, err)
	}
	params, err := c.Call(context.Background(), "getParamset", String("MB001:0"), String("VALUES"))
	if err != nil || params.Members["VALUE"].Text != "42" {
		t.Fatalf("params %+v %v", params, err)
	}
	for _, method := range []string{"getParamsetDescription", "setReadyConfig"} {
		var args []Value
		if method == "getParamsetDescription" {
			args = []Value{String("MB001:0"), String("VALUES")}
		}
		if _, err := c.Call(context.Background(), method, args...); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}
	if _, err := c.Call(context.Background(), "getParamset", Integer(3)); err == nil {
		t.Fatal("accepted wrong arguments")
	}
}
