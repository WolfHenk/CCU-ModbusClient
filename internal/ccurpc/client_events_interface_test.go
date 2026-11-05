// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_events_interface_test.go
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
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeliverEventsUsesNumericReGaID(t *testing.T) {
	queue, _ := NewEventQueue(2)
	if err := queue.Enqueue(PointEvent{Address: "MBC000105DE01:1", Key: "VOLTAGE", Value: Integer(12)}); err != nil {
		t.Fatal(err)
	}
	received := make(chan []Value, 1)
	handler := NewHandler()
	if err := handler.Register("event", func(args []Value) (Value, error) { received <- args; return String(""), nil }); err != nil {
		t.Fatal(err)
	}
	receiver := httptest.NewServer(handler)
	defer receiver.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- DeliverEventsForInterface(ctx, queue, Client{URL: receiver.URL}, "1049") }()
	select {
	case args := <-received:
		if len(args) != 4 || args[0].Text != "1049" || args[1].Text != "MBC000105DE01:1" || args[2].Text != "VOLTAGE" {
			t.Fatalf("incorrect ReGa event: %+v", args)
		}
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
