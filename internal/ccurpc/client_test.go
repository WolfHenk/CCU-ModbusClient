// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_test.go
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

func TestClientLocalOnly(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || !strings.HasPrefix(r.Header.Get("Content-Type"), "text/xml") {
			t.Errorf("bad request method/headers")
		}
		method, e := ParseMethodName(r.Body)
		if e != nil || method != "updateDevice" {
			t.Errorf("bad request %q %v", method, e)
		}
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?><methodResponse><params><param><value><int>7</int></value></param></params></methodResponse>`))
	}))
	defer s.Close()
	got, err := (Client{URL: s.URL}).Call(context.Background(), "updateDevice", String("Modbus"))
	if err != nil || got.Kind != "int" || got.Text != "7" {
		t.Fatalf("response: %+v %v", got, err)
	}
}
func TestFaultAndMalformedResponse(t *testing.T) {
	fault := `<methodResponse><fault><value><struct><member><name>faultCode</name><value><int>4</int></value></member><member><name>faultString</name><value><string>Denied</string></value></member></struct></value></fault></methodResponse>`
	if _, e := ParseResponse(strings.NewReader(fault)); e == nil || !strings.Contains(e.Error(), "Denied") {
		t.Fatalf("fault not reported: %v", e)
	}
	for _, s := range []string{`<methodResponse><params></params></methodResponse>`, `<bad></bad>`, `<methodResponse><params><param><value><boolean>garbage</boolean></value></param></params></methodResponse>`} {
		if _, e := ParseResponse(strings.NewReader(s)); e == nil {
			t.Fatalf("accepted invalid response %s", s)
		}
	}
}

func TestSimpleFaultDoesNotPanic(t *testing.T) {
	_, err := ParseResponse(strings.NewReader(`<methodResponse><fault><value><string>bad</string></value></fault></methodResponse>`))
	if err == nil {
		t.Fatal("fault silently accepted")
	}
}
