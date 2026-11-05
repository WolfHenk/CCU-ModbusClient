// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/event_delivery.go
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
)

// MoveEvents transfers as many pending events as fit into the bounded queue.
// The remainder stays in the catalog. It performs no network operations.
func (c *MemoryCatalog) MoveEvents(q *EventQueue) (int, error) {
	if c == nil || q == nil {
		return 0, errors.New("missing event catalog or queue")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	moved := 0
	for moved < len(c.pending) {
		if err := q.Enqueue(c.pending[moved]); err != nil {
			c.pending = append([]PointEvent(nil), c.pending[moved:]...)
			return moved, err
		}
		moved++
	}
	c.pending = nil
	return moved, nil
}

// XMLRPCEventSender prepares CCU event(interfaceID, channel, key, value).
// It can point to a local test receiver or, after validation, to ReGa.
type XMLRPCEventSender struct {
	Client      Client
	InterfaceID string
}

func (s XMLRPCEventSender) Send(ctx context.Context, event PointEvent) error {
	if s.InterfaceID == "" || event.Address == "" || event.Key == "" {
		return errors.New("incomplete CCU event")
	}
	_, err := s.Client.Call(ctx, "event", String(s.InterfaceID), String(event.Address), String(event.Key), event.Value)
	return err
}
