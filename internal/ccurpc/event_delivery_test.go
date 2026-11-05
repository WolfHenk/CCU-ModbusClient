// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/event_delivery_test.go
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
	"time"
)

func TestEventDeliveryToLocalXMLRPCReceiver(t *testing.T) {
	h := NewHandler()
	received := make(chan []Value, 2)
	if e := h.Register("event", func(args []Value) (Value, error) { received <- args; return String(""), nil }); e != nil {
		t.Fatal(e)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	catalog := NewMemoryCatalog()
	catalog.pending = []PointEvent{{Address: "MB001:1", Key: "STATE", Value: Boolean(true)}, {Address: "MB001:2", Key: "VALUE", Value: Integer(12)}}
	queue, _ := NewEventQueue(1)
	moved, err := catalog.MoveEvents(queue)
	if moved != 1 || err == nil || len(catalog.pending) != 1 {
		t.Fatalf("partial transfer: %d %v, pending %d", moved, err, len(catalog.pending))
	}
	sender := XMLRPCEventSender{Client: Client{URL: srv.URL}, InterfaceID: "CCU-Modbus"}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- queue.Run(ctx, sender.Send, time.Millisecond) }()
	select {
	case args := <-received:
		if len(args) != 4 || args[0].Text != "CCU-Modbus" || args[1].Text != "MB001:1" || args[2].Text != "STATE" || args[3].Text != "1" {
			t.Fatalf("incorrect event args %+v", args)
		}
	case <-time.After(time.Second):
		t.Fatal("first event not received")
	}
	deadline := time.Now().Add(time.Second)
	for queue.Len() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	moved, err = catalog.MoveEvents(queue)
	if err != nil || moved != 1 {
		t.Fatalf("remaining transfer %d %v", moved, err)
	}
	select {
	case args := <-received:
		if args[1].Text != "MB001:2" || args[3].Text != "12" {
			t.Fatalf("second event %+v", args)
		}
	case <-time.After(time.Second):
		t.Fatal("second event not received")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestEventDeliveryRejectsMissingInterface(t *testing.T) {
	err := (XMLRPCEventSender{}).Send(context.Background(), PointEvent{Address: "MB:1", Key: "STATE"})
	if !errors.Is(err, errors.New("incomplete CCU event")) && err == nil {
		t.Fatal("missing interface accepted")
	}
}
