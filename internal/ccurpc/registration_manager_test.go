// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/registration_manager_test.go
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

func TestRegistrationManagerReconnectsAfterProbeFailure(t *testing.T) {
	h := NewHandler()
	var registrations atomic.Int32
	if err := h.Register("init", func(args []Value) (Value, error) { registrations.Add(1); return String(""), nil }); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var probes atomic.Int32
	mgr := &RegistrationManager{Registration: Registration{Client: Client{URL: srv.URL}, CallbackURL: "xmlrpc_bin://127.0.0.1:31999", InterfaceID: "CCU-Modbus", Retry: time.Millisecond}, Interval: time.Millisecond, Probe: func(context.Context) error {
		if probes.Add(1) == 1 {
			return errors.New("simulated disconnect")
		}
		return nil
	}}
	done := make(chan error, 1)
	go func() { done <- mgr.Run(ctx) }()
	deadline := time.Now().Add(600 * time.Millisecond)
	for (registrations.Load() < 2 || !mgr.Connected()) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if registrations.Load() < 2 || !mgr.Connected() {
		t.Fatalf("failed to reconnect, registrations=%d connected=%v", registrations.Load(), mgr.Connected())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("manager did not stop")
	}
	if mgr.Connected() {
		t.Fatal("connected state not reset")
	}
}
func TestRegistrationManagerRejectsMissingProbe(t *testing.T) {
	err := (&RegistrationManager{}).Run(context.Background())
	if err == nil {
		t.Fatal("missing probe accepted")
	}
}
