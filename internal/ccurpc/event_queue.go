// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/event_queue.go
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
	"sync"
	"time"
)

// EventSender performs one network delivery. It must honor the context.
type EventSender func(context.Context, PointEvent) error

// EventQueue bounds pending work. A full queue rejects new items explicitly;
// it never silently discards an older event or blocks a Modbus polling loop.
type EventQueue struct {
	mu       sync.Mutex
	items    []PointEvent
	capacity int
	wake     chan struct{}
}

func NewEventQueue(capacity int) (*EventQueue, error) {
	if capacity < 1 {
		return nil, errors.New("event queue capacity must be positive")
	}
	return &EventQueue{capacity: capacity, wake: make(chan struct{}, 1)}, nil
}
func (q *EventQueue) Enqueue(event PointEvent) error {
	if q == nil {
		return errors.New("nil event queue")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) >= q.capacity {
		return errors.New("event queue full")
	}
	q.items = append(q.items, event)
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return nil
}
func (q *EventQueue) Len() int { q.mu.Lock(); defer q.mu.Unlock(); return len(q.items) }

// Run sends in FIFO order, removes only acknowledged events and retries the
// failed head after delay. Delivery is at-least-once, not exactly-once.
func (q *EventQueue) Run(ctx context.Context, sender EventSender, retry time.Duration) error {
	if q == nil || ctx == nil || sender == nil {
		return errors.New("invalid event worker")
	}
	if retry <= 0 {
		retry = time.Second
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		q.mu.Lock()
		hasWork := len(q.items) > 0
		var event PointEvent
		if hasWork {
			event = q.items[0]
		}
		q.mu.Unlock()
		if !hasWork {
			select {
			case <-ctx.Done():
				return nil
			case <-q.wake:
			}
			continue
		}
		if err := sender(ctx, event); err != nil {
			timer := time.NewTimer(retry)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
			continue
		}
		q.mu.Lock()
		if len(q.items) > 0 {
			q.items = q.items[1:]
		}
		q.mu.Unlock()
	}
}
