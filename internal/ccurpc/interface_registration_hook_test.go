// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/interface_registration_hook_test.go
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

func TestInterfaceInitOnlyCommitsAfterAnnouncement(t *testing.T) {
	reg := NewInterfaceRegistration()
	fail := true
	calls := 0
	reg.OnInit = func(receiver, id string) error {
		calls++
		if receiver != "xmlrpc_bin://127.0.0.1:31998" || id != "1040" {
			t.Fatalf("wrong init: %q %q", receiver, id)
		}
		if fail {
			return errors.New("rega not ready")
		}
		return nil
	}
	handler := NewHandler()
	if err := reg.Register(handler); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	client := Client{URL: server.URL}
	address := String("xmlrpc_bin://127.0.0.1:31998")
	if _, err := client.Call(context.Background(), "init", address, String("1040")); err == nil {
		t.Fatal("failed announcement accepted")
	}
	if _, ok := reg.Receiver(address.Text); ok {
		t.Fatal("failed announcement stored registration")
	}
	fail = false
	if _, err := client.Call(context.Background(), "init", address, String("1040")); err != nil {
		t.Fatal(err)
	}
	if id, ok := reg.Receiver(address.Text); !ok || id != "1040" {
		t.Fatalf("registration missing %q %v", id, ok)
	}
	if calls != 2 {
		t.Fatalf("hook called %d times", calls)
	}
}
