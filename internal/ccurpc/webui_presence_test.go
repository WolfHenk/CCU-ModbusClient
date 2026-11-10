// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/webui_presence_test.go
// Ersteller:    Wolfram Henkel
// Datum:        10. Nov. 2026
// Version:      0.3.3
//
// Copyright (c) 2026 Wolfram Henkel.
//
// Lizenz:       CCU-ModbusClient-Lizenz für kostenfreie Nutzung und
// Weitergabe, Version 1.1.
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
	"reflect"
	"sort"
	"testing"
)

// OpenCCU's Interface.isPresent issues system.listMethods against /RPC3.
// The result must reflect later method registrations, including init/setValue.
func TestWebUIPresenceAndServiceMessages(t *testing.T) {
	handler := NewHandler()
	if err := RegisterCCUReadMethods(handler, fakeCatalog{}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/RPC3", handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := Client{URL: srv.URL + "/RPC3"}
	ctx := context.Background()

	check := func() []string {
		t.Helper()
		got, err := client.Call(ctx, "system.listMethods")
		if err != nil || got.Kind != "array" {
			t.Fatalf("system.listMethods: %+v, %v", got, err)
		}
		names := make([]string, 0, len(got.Array))
		for _, v := range got.Array {
			if v.Kind != "string" {
				t.Fatalf("list item is not a string: %+v", v)
			}
			names = append(names, v.Text)
		}
		if !sort.StringsAreSorted(names) {
			t.Fatalf("unstable method order: %v", names)
		}
		return names
	}
	before := check()
	if !containsMethod(before, "system.listMethods") ||
		!containsMethod(before, "getServiceMessages") ||
		!containsMethod(before, "listDevices") {
		t.Fatalf("incomplete method listing: %v", before)
	}
	if err := handler.Register("init", func([]Value) (Value, error) { return String(""), nil }); err != nil {
		t.Fatal(err)
	}
	if err := handler.Register("setValue", func([]Value) (Value, error) { return String(""), nil }); err != nil {
		t.Fatal(err)
	}
	after := check()
	expected := append(append([]string{}, before...), "init", "setValue")
	sort.Strings(expected)
	if !reflect.DeepEqual(after, expected) {
		t.Fatalf("method listing not dynamic: got=%v want=%v", after, expected)
	}

	service, err := client.Call(ctx, "getServiceMessages")
	if err != nil || service.Kind != "array" || len(service.Array) != 0 {
		t.Fatalf("getServiceMessages: %+v, %v", service, err)
	}
	for _, method := range []string{"system.listMethods", "getServiceMessages"} {
		if _, err := client.Call(ctx, method, String("extra")); err == nil {
			t.Fatalf("%s accepted unexpected parameter", method)
		}
	}
}
func containsMethod(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
