// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/rega_announce_subset_test.go
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

func TestAnnounceOnlyMaintenanceChannel(t *testing.T) {
	h := NewHandler()
	calls := 0
	h.Register("newDevices", func(args []Value) (Value, error) {
		calls++
		if len(args) != 2 || args[0].Text != "1049" || args[1].Kind != "array" || len(args[1].Array) != 1 || args[1].Array[0].Members["ADDRESS"].Text != "MB003206502001:0" {
			t.Fatalf("incorrect maintenance-only announcement %+v", args)
		}
		return String(""), nil
	})
	srv := httptest.NewServer(h)
	defer srv.Close()
	desc := Struct(map[string]Value{"ADDRESS": String("MB003206502001:0"), "TYPE": String("MAINTENANCE"), "VERSION": Integer(3)})
	if err := AnnounceDescriptions(context.Background(), Client{URL: srv.URL}, "1049", []Value{desc}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	if err := AnnounceDescriptions(context.Background(), Client{URL: srv.URL}, "1049", nil); err == nil {
		t.Fatal("empty migration allowed")
	}
	if calls != 1 {
		t.Fatal("invalid migration reached ReGa")
	}
}
