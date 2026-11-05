// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/maintenance_refresh_test.go
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
	"errors"
	"net/http/httptest"
	"testing"
)

func TestUpdateMaintenanceDescriptionRPCContract(t *testing.T) {
	calls := 0
	fail := false
	h := NewHandler()
	if err := h.Register("updateDevice", func(args []Value) (Value, error) {
		calls++
		if len(args) != 3 || args[0].Text != "1035" || args[1].Text != "MB007089504001:0" || args[2].Kind != "int" || args[2].Text != "0" {
			t.Fatalf("wrong RPC arguments: %+v", args)
		}
		if fail {
			return Value{}, errors.New("ReGa error")
		}
		return String(""), nil
	}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h)
	defer server.Close()
	client := Client{URL: server.URL}
	if err := UpdateMaintenanceDescription(context.Background(), client, "1035", "MB007089504001:0"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	for _, addr := range []string{"", "MB007089504001", "MB007089504001:1"} {
		if err := UpdateMaintenanceDescription(context.Background(), client, "1035", addr); err == nil {
			t.Fatalf("accepted %q", addr)
		}
	}
	if calls != 1 {
		t.Fatal("invalid calls reached ReGa")
	}
	fail = true
	if err := UpdateMaintenanceDescription(context.Background(), client, "1035", "MB007089504001:0"); err == nil {
		t.Fatal("ignored RPC error")
	}
}
