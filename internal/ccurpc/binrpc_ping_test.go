// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/binrpc_ping_test.go
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
	"testing"
	"time"
)

func TestBinRPCPingQueuesPong(t *testing.T) {
	q, err := NewEventQueue(1)
	if err != nil {
		t.Fatal(err)
	}
	s := ccubinrpc.NewTCPServer()
	if err := RegisterBinRPCPing(s, q); err != nil {
		t.Fatal(err)
	}
	listener, _, err := ccubinrpc.BindAvailable("127.0.0.1", 32600, 32609)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, listener) }()
	defer func() {
		cancel()
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("ping server shutdown stalled")
		}
	}()
	c := ccubinrpc.TCPClient{Address: listener.Addr().String(), Timeout: time.Second}
	answer, err := c.Call(ctx, "ping", ccubinrpc.Parameter{Type: ccubinrpc.TypeString, Text: "probe-1"})
	if err != nil {
		t.Fatal(err)
	}
	if answer.Type != ccubinrpc.TypeBoolean || !answer.Flag {
		t.Fatalf("wrong ping response: %+v", answer)
	}
	if q.Len() != 1 {
		t.Fatalf("PONG event absent: %d", q.Len())
	}
	// Never acknowledge a PONG that could not be queued.
	if _, err := c.Call(ctx, "ping", ccubinrpc.Parameter{Type: ccubinrpc.TypeString, Text: "probe-2"}); err == nil {
		t.Fatal("queue overflow silently accepted")
	}
	events := make(chan PointEvent, 1)
	deliveryCtx, stop := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() {
		finished <- q.Run(deliveryCtx, func(_ context.Context, e PointEvent) error { events <- e; stop(); return nil }, time.Millisecond)
	}()
	select {
	case e := <-events:
		if e.Address != "CENTRAL" || e.Key != "PONG" || e.Value.Text != "probe-1" {
			t.Fatalf("wrong event: %+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("PONG not delivered")
	}
	<-finished
}
