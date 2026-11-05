// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/registration.go
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

// Registration manages the CCU XML-RPC init/deinit handshake independently
// of the existing go-hmccu runtime. It does not change device descriptions.
type Registration struct {
	Client                   Client
	CallbackURL, InterfaceID string
	Retry                    time.Duration
}

func (r Registration) Init(ctx context.Context) error {
	if r.CallbackURL == "" || r.InterfaceID == "" {
		return errors.New("incomplete CCU registration")
	}
	_, err := r.Client.Call(ctx, "init", String(r.CallbackURL), String(r.InterfaceID))
	return err
}
func (r Registration) Deinit(ctx context.Context) error {
	if r.CallbackURL == "" {
		return errors.New("missing callback address")
	}
	_, err := r.Client.Call(ctx, "init", String(r.CallbackURL))
	return err
}

// RetryInit retries after failures until successful registration or cancellation.
// Caller controls whether and when to re-register after a later disconnect.
func (r Registration) RetryInit(ctx context.Context) error {
	if ctx == nil {
		return errors.New("nil context")
	}
	pause := r.Retry
	if pause <= 0 {
		pause = 2 * time.Second
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := r.Init(ctx); err == nil {
			return nil
		}
		timer := time.NewTimer(pause)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
