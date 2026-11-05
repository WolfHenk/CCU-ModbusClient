// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/instance_runtime_test.go
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
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSecondRuntimeCannotRegister(t *testing.T) {
	var active atomic.Int32
	h := NewHandler()
	if err := h.Register("init", func(args []Value) (Value, error) {
		if len(args) == 2 {
			active.Add(1)
		}
		return String(""), nil
	}); err != nil {
		t.Fatal(err)
	}
	remote := httptest.NewServer(h)
	defer remote.Close()
	shared := filepath.Join(t.TempDir(), "instance.lock")
	makeRuntime := func() *CallbackRuntime {
		r, err := ReserveCallback("127.0.0.1", 32140, 32149, ccubinrpc.NewTCPServer(), NewClientRegistration(Client{URL: remote.URL}))
		if err != nil {
			t.Fatal(err)
		}
		r.LockPath = shared
		return r
	}
	first := makeRuntime()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- first.ServeRegistered(ctx) }()
	select {
	case <-first.Registered:
	case <-time.After(2 * time.Second):
		t.Fatal("first did not register")
	}
	second := makeRuntime()
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if err := second.ServeRegistered(ctx2); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("second instance not denied: %v", err)
	}
	if active.Load() != 1 {
		t.Fatalf("unexpected registrations: %d", active.Load())
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown hung")
	}
	if active.Load() != 1 {
		t.Fatal("unexpected registration after shutdown")
	}
}
