// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/event_queue_test.go
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
	"sync/atomic"
	"testing"
	"time"
)

func TestEventQueueBoundedFIFOAndRetry(t *testing.T) {
	q, err := NewEventQueue(2)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"A", "B"} {
		if err := q.Enqueue(PointEvent{Key: name}); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.Enqueue(PointEvent{Key: "C"}); err == nil {
		t.Fatal("queue overflow accepted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan struct{})
	var attempts atomic.Int32
	go func() {
		defer close(done)
		_ = q.Run(ctx, func(_ context.Context, e PointEvent) error {
			if e.Key == "A" && attempts.Add(1) == 1 {
				return errors.New("temporary failure")
			}
			if e.Key == "B" {
				cancel()
			}
			return nil
		}, time.Millisecond)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not exit")
	}
	if attempts.Load() != 2 {
		t.Fatalf("first event not retried: %d", attempts.Load())
	}
	if q.Len() > 1 {
		t.Fatalf("unbounded pending queue: %d", q.Len())
	}
}
func TestEventQueueInvalidCapacity(t *testing.T) {
	if _, err := NewEventQueue(0); err == nil {
		t.Fatal("zero capacity accepted")
	}
}
