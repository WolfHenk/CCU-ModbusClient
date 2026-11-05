// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/registration_test.go
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
	"sync/atomic"
	"testing"
	"time"
)

func TestRegistrationRetryAndDeinit(t *testing.T) {
	h := NewHandler()
	var calls atomic.Int32
	received := make(chan []Value, 4)
	if err := h.Register("init", func(args []Value) (Value, error) {
		received <- args
		if len(args) == 2 && calls.Add(1) == 1 {
			return Value{}, errors.New("temporary error")
		}
		return String(""), nil
	}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	r := Registration{Client: Client{URL: srv.URL}, CallbackURL: "xmlrpc_bin://127.0.0.1:31999", InterfaceID: "CCU-Modbus", Retry: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := r.RetryInit(ctx); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected retry, got %d attempts", calls.Load())
	}
	if err := r.Deinit(ctx); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		select {
		case args := <-received:
			if len(args) != 2-(i/2) || args[0].Text != r.CallbackURL {
				t.Fatalf("bad registration %+v", args)
			}
			if i == 2 && len(args) != 1 {
				t.Fatalf("deinit ID not empty: %+v", args)
			}
		case <-time.After(time.Second):
			t.Fatal("missing init call")
		}
	}
}
func TestRegistrationCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := (Registration{}).RetryInit(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
