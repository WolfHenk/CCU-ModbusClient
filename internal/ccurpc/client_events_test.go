// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_events_test.go
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

func TestClientEventPumpRetryAndInterface(t *testing.T) {
	cat := NewMemoryCatalog()
	if err := cat.AddDevice(Device{Address: "MBC03CF01F601", Type: "Modbus", Channels: []Channel{{Address: "MBC03CF01F601:1", Type: "DIGITAL_INPUT", Points: []Point{{ID: "STATE", Type: "BOOL", Value: Boolean(false), Read: true, Event: true}}}}}); err != nil {
		t.Fatal(err)
	}
	cat.mu.Lock()
	cat.pending = append(cat.pending, PointEvent{Address: "MBC03CF01F601:1", Key: "STATE", Value: Boolean(true)})
	cat.mu.Unlock()
	queue, err := NewEventQueue(2)
	if err != nil {
		t.Fatal(err)
	}
	var attempts atomic.Int32
	received := make(chan []Value, 1)
	h := NewHandler()
	if err := h.Register("event", func(args []Value) (Value, error) {
		if attempts.Add(1) == 1 {
			return Value{}, errors.New("temporary refusal")
		}
		received <- args
		return String(""), nil
	}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	pumpDone := make(chan error, 1)
	sendDone := make(chan error, 1)
	go func() { pumpDone <- PumpClientEvents(ctx, cat, queue, 10*time.Millisecond) }()
	go func() { sendDone <- DeliverClientEvents(ctx, queue, Client{URL: srv.URL}) }()
	select {
	case args := <-received:
		if len(args) != 4 || args[0].Text != ClientInterfaceID || args[1].Text != "MBC03CF01F601:1" || args[2].Text != "STATE" || args[3].Text != "1" {
			t.Fatalf("wrong event: %+v", args)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("event retry not delivered")
	}
	if attempts.Load() < 2 {
		t.Fatal("retry was not exercised")
	}
	cancel()
	select {
	case err := <-pumpDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("pump did not stop")
	}
	select {
	case err := <-sendDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("sender did not stop")
	}
}
