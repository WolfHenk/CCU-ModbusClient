// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_start_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccubinrpc"
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientStartRegistersMBCIdentity(t *testing.T) {
	observed := make(chan []Value, 2)
	h := NewHandler()
	if err := h.Register("init", func(args []Value) (Value, error) { observed <- args; return String(""), nil }); err != nil {
		t.Fatal(err)
	}
	remote := httptest.NewServer(h)
	defer remote.Close()
	cfg := &config.Config{Devices: []config.Device{{ID: "test", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1}}}
	runtime, catalog, err := PrepareClient(cfg, Client{URL: remote.URL}, ccubinrpc.NewTCPServer(), 32040, 32049)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.DeviceDescription("MBC03CF01F601"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.ServeRegistered(ctx) }()
	select {
	case args := <-observed:
		if len(args) != 2 || args[1].Text != ClientInterfaceID || args[0].Text != runtime.Registration.CallbackURL {
			t.Fatalf("wrong registration: %+v", args)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("registration not received")
	}
	select {
	case <-runtime.Registered:
	case <-time.After(2 * time.Second):
		t.Fatal("registration response not completed")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server failed to stop")
	}
	select {
	case args := <-observed:
		if len(args) == 2 {
			t.Fatalf("deinit failed: %+v", args)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("deinit missing")
	}
}
