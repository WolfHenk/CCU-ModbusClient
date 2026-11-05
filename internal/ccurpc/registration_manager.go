// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/registration_manager.go
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

// RegistrationManager tracks registration separately from Modbus polling.
// Probe must check an actual remote condition; a successful event alone is
// not proof the CCU still knows the interface.
type RegistrationManager struct {
	Registration Registration
	Probe        func(context.Context) error
	Interval     time.Duration
	mu           sync.RWMutex
	connected    bool
}

func (m *RegistrationManager) Connected() bool {
	if m == nil {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}
func (m *RegistrationManager) setConnected(v bool) { m.mu.Lock(); m.connected = v; m.mu.Unlock() }

// Run registers initially and re-registers after a probe failure.
// It never launches network calls on the Modbus polling goroutine.
func (m *RegistrationManager) Run(ctx context.Context) error {
	if m == nil || ctx == nil || m.Probe == nil {
		return errors.New("invalid registration monitor")
	}
	interval := m.Interval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	m.setConnected(false)
	defer m.setConnected(false)
	for {
		if err := m.Registration.RetryInit(ctx); err != nil {
			return err
		}
		m.setConnected(true)
		ticker := time.NewTicker(interval)
		reconnect := false
		for !reconnect {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return nil
			case <-ticker.C:
				if err := m.Probe(ctx); err != nil {
					m.setConnected(false)
					reconnect = true
				}
			}
		}
		ticker.Stop()
	}
}
