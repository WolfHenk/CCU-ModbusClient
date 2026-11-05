// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/callback_retry_test.go
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
	"sync/atomic"
	"testing"
	"time"
)

func TestRuntimeRetriesUnavailableRegistration(t *testing.T) {
	var count atomic.Int32
	h := NewHandler()
	if err := h.Register("init", func(args []Value) (Value, error) {
		if len(args) == 2 && count.Add(1) < 3 {
			return Value{}, context.DeadlineExceeded
		}
		return String(""), nil
	}); err != nil {
		t.Fatal(err)
	}
	remote := httptest.NewServer(h)
	defer remote.Close()
	r, err := ReserveCallback("127.0.0.1", 32400, 32409, ccubinrpc.NewTCPServer(), NewClientRegistration(Client{URL: remote.URL}))
	if err != nil {
		t.Fatal(err)
	}
	r.LockPath = filepath.Join(t.TempDir(), "instance.lock")
	r.Registration.Retry = 15 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.ServeRegistered(ctx) }()
	select {
	case <-r.Registered:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("retry registration not completed")
	}
	if count.Load() != 3 {
		cancel()
		t.Fatalf("unexpected registration attempts: %d", count.Load())
	}
	cancel()
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown timeout")
	}
}
