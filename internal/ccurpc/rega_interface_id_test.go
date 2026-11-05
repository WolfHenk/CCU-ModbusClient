// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/rega_interface_id_test.go
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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveClientInterfaceID(t *testing.T) {
	reply := "1049<xml><exec>/rega.exe</exec></xml>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Error("method")
		}
		w.Write([]byte(reply))
	}))
	defer srv.Close()
	id, err := ResolveClientInterfaceID(context.Background(), srv.URL)
	if err != nil || id != "1049" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	reply = "<xml><exec>/rega.exe</exec></xml>"
	if _, err := ResolveClientInterfaceID(context.Background(), srv.URL); err == nil {
		t.Fatal("accepted missing ID")
	}
	reply = strings.Repeat("x", 5000)
	if _, err := ResolveClientInterfaceID(context.Background(), srv.URL); err == nil {
		t.Fatal("accepted oversize")
	}
}
