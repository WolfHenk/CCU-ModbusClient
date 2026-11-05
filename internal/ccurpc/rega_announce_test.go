// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/rega_announce_test.go
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

func TestAnnounceDevicesMatchesReGaContract(t *testing.T) {
	catalog := NewMemoryCatalog()
	address := "MBC010105DE01"
	if err := catalog.AddDevice(Device{Address: address, Type: "Modbus", Channels: []Channel{{Address: address + ":0", Type: "MAINTENANCE"}}}); err != nil {
		t.Fatal(err)
	}
	h := NewHandler()
	if err := h.Register("newDevices", func(args []Value) (Value, error) {
		if len(args) != 2 || args[0].Kind != "string" || args[0].Text != "1040" {
			t.Fatalf("bad interface: %+v", args)
		}
		if args[1].Kind != "array" || len(args[1].Array) != 2 {
			t.Fatalf("bad descriptions: %+v", args[1])
		}
		if args[1].Array[0].Members["ADDRESS"].Text != address || args[1].Array[1].Members["ADDRESS"].Text != address+":0" {
			t.Fatalf("device descriptions wrong: %+v", args[1])
		}
		return String(""), nil
	}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	if err := AnnounceDevices(context.Background(), Client{URL: srv.URL}, "1040", catalog); err != nil {
		t.Fatal(err)
	}
}

func TestAnnounceEmptyDeviceCatalogWithoutReGaCall(t *testing.T) {
	if err := AnnounceDevices(context.Background(), Client{URL: "http://127.0.0.1:1/"}, "1040", NewMemoryCatalog()); err != nil {
		t.Fatal(err)
	}
}
