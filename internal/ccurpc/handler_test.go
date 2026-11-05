// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/handler_test.go
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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerDispatch(t *testing.T) {
	h := NewHandler()
	if err := h.Register("setReadyConfig", func(v []Value) (Value, error) {
		if len(v) != 1 || v[0].Text != "MB01" {
			t.Errorf("wrong params %+v", v)
		}
		return String(""), nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := h.Register("setReadyConfig", func([]Value) (Value, error) { return String(""), nil }); err == nil {
		t.Fatal("accepted duplicate")
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	got, err := (Client{URL: srv.URL}).Call(context.Background(), "setReadyConfig", String("MB01"))
	if err != nil || got.Kind != "string" {
		t.Fatalf("bad RPC response: %+v %v", got, err)
	}
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("bad status %d", res.StatusCode)
	}
}

func TestHandlerFaults(t *testing.T) {
	h := NewHandler()
	_ = h.Register("fail", func([]Value) (Value, error) { return Value{}, fmt.Errorf("sensitive internal detail") })
	s := httptest.NewServer(h)
	defer s.Close()
	for _, method := range []string{"missing", "fail"} {
		_, err := (Client{URL: s.URL}).Call(context.Background(), method)
		if err == nil || !strings.Contains(err.Error(), "XML-RPC fault:") {
			t.Fatalf("%s missing fault: %v", method, err)
		}
		if strings.Contains(err.Error(), "sensitive internal detail") {
			t.Fatal("internal detail leaked")
		}
	}
}
