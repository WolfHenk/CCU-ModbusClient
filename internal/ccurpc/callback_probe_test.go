// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/callback_probe_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccubinrpc"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestRuntimeReRegistersAfterVerifiedRegistrationLoss(t *testing.T) {
	var registered atomic.Bool
	var attempts atomic.Int32
	h := NewHandler()
	if err := h.Register("init", func(args []Value) (Value, error) {
		if len(args) == 2 {
			attempts.Add(1)
			registered.Store(true)
		} else {
			registered.Store(false)
		}
		return String(""), nil
	}); err != nil {
		t.Fatal(err)
	}
	receiver := httptest.NewServer(h)
	defer receiver.Close()
	runtime, err := ReserveCallback("127.0.0.1", 32450, 32459, ccubinrpc.NewTCPServer(), NewClientRegistration(Client{URL: receiver.URL}))
	if err != nil {
		t.Fatal(err)
	}
	runtime.LockPath = filepath.Join(t.TempDir(), "lock")
	runtime.ProbeInterval = 10 * time.Millisecond
	runtime.RegistrationProbe = func(context.Context) error {
		if !registered.Load() {
			return errors.New("lost registration")
		}
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.ServeRegistered(ctx) }()
	select {
	case <-runtime.Registered:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("initial registration timed out")
	}
	registered.Store(false) // Simulate CCU restart erasing registered interface.
	deadline := time.Now().Add(time.Second)
	for attempts.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if attempts.Load() < 2 || !registered.Load() {
		cancel()
		t.Fatalf("re-registration missing, attempts=%d", attempts.Load())
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("monitor/server shutdown timeout")
	}
}
