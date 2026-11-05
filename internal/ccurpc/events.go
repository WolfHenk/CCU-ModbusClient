// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/events.go
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

// PointEvent holds a local, unsent data point change. Network I/O is handled
// separately, never by ApplySnapshot or while holding the catalog lock.
type PointEvent struct {
	Address, Key string
	Value        Value
}

// DrainEvents moves the queued events to the caller for later dispatch.
// The returned events have NOT been delivered to ReGa or OpenCCU.
func (c *MemoryCatalog) DrainEvents() []PointEvent {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	result := append([]PointEvent(nil), c.pending...)
	c.pending = nil
	return result
}
