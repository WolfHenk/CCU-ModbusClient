// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/rega_ui_compat_test.go
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

func TestReGaChannelZeroUIRequests(t *testing.T) {
	cat := NewMemoryCatalog()
	if err := cat.AddDevice(Device{Address: "MBC000105DE01", Type: "Modbus", Channels: []Channel{{Address: "MBC000105DE01:0", Type: "MAINTENANCE", Points: []Point{{ID: "UNREACH", Type: "BOOL", Read: true, Event: true, Value: Boolean(false)}}}}}); err != nil {
		t.Fatal(err)
	}
	h := NewHandler()
	if err := RegisterCCUReadMethods(h, cat); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	rpc := Client{URL: srv.URL}
	ctx := context.Background()
	for _, address := range []string{"MBC000105DE01", "MBC000105DE01:0"} {
		v, err := rpc.Call(ctx, "getDeviceDescription", String(address))
		if err != nil || v.Members["ADDRESS"].Text != address {
			t.Fatalf("description %s: %v %+v", address, err, v)
		}
		v, err = rpc.Call(ctx, "getParamsetDescription", String(address), String("MASTER"))
		if err != nil || v.Kind != "struct" || len(v.Members) != 0 {
			t.Fatalf("master description %s: %v %+v", address, err, v)
		}
	}
	v, err := rpc.Call(ctx, "getParamsetDescription", String("MBC000105DE01:0"), String("VALUES"))
	if err != nil || v.Members["UNREACH"].Kind != "struct" {
		t.Fatalf("values: %v %+v", err, v)
	}
	v, err = rpc.Call(ctx, "getLinks", String("MBC000105DE01:0"), Integer(0))
	if err != nil || v.Kind != "array" || len(v.Array) != 0 {
		t.Fatalf("getLinks: %v %+v", err, v)
	}
}
