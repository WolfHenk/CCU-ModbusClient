// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/interface_registration_test.go
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

func TestInterfaceRegistrationInitAndDeinit(t *testing.T) {
	reg := NewInterfaceRegistration()
	h := NewHandler()
	if err := reg.Register(h); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	client := Client{URL: srv.URL}
	ctx := context.Background()
	if _, err := client.Call(ctx, "init", String("xmlrpc_bin://127.0.0.1:31999"), String("1040")); err != nil {
		t.Fatal(err)
	}
	if id, ok := reg.Receiver("xmlrpc_bin://127.0.0.1:31999"); !ok || id != "1040" {
		t.Fatalf("not registered: %q %v", id, ok)
	}
	for _, args := range [][]Value{{}, {Integer(1)}, {String("")}, {String("abc"), String("")}} {
		if _, err := client.Call(ctx, "init", args...); err == nil {
			t.Fatalf("accepted invalid args: %+v", args)
		}
	}
	if _, err := client.Call(ctx, "init", String("xmlrpc_bin://127.0.0.1:31999")); err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.Receiver("xmlrpc_bin://127.0.0.1:31999"); ok {
		t.Fatal("deinit retained receiver")
	}
}
