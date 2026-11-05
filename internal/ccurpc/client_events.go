// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_events.go
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
	"time"
)

// PumpClientEvents moves change notifications into a bounded retry queue.
// Failures to enqueue are retained in the catalog for a later retry.
func PumpClientEvents(ctx context.Context, catalog *MemoryCatalog, queue *EventQueue, interval time.Duration) error {
	if ctx == nil || catalog == nil || queue == nil || interval <= 0 {
		return errors.New("invalid event pump")
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return nil
		}
		_, _ = catalog.MoveEvents(queue) // Queue-full is transient, remainder is retained.
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// DeliverClientEvents uses the independent XML-RPC event method and retries
// failures in order. The caller must use a verified CCU event endpoint.
// DeliverEventsForInterface uses the runtime ReGa interface ID, not the addon name.
func DeliverEventsForInterface(ctx context.Context, queue *EventQueue, receiver Client, interfaceID string) error {
	if interfaceID == "" {
		return errors.New("missing ReGa interface ID")
	}
	sender := XMLRPCEventSender{Client: receiver, InterfaceID: interfaceID}
	return queue.Run(ctx, sender.Send, 500*time.Millisecond)
}

func DeliverClientEvents(ctx context.Context, queue *EventQueue, receiver Client) error {
	sender := XMLRPCEventSender{Client: receiver, InterfaceID: ClientInterfaceID}
	return queue.Run(ctx, sender.Send, 500*time.Millisecond)
}
