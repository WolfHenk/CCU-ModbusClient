// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/callback_runtime.go
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
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/WolfHenk/CCU-ModbusClient/internal/ccubinrpc"
)

// CallbackRuntime owns a reserved BinRPC listener and its matching CCU registration.
// It does not select an interface identity: callers must supply an isolated identity.
type CallbackRuntime struct {
	Listener          net.Listener
	Registration      Registration
	Server            *ccubinrpc.TCPServer
	Registered        chan struct{}
	LockPath          string
	RegistrationProbe func(context.Context) error
	ProbeInterval     time.Duration
}

// ReserveCallback binds before registering, avoiding a check-then-bind race.
func ReserveCallback(host string, first, last int, server *ccubinrpc.TCPServer, reg Registration) (*CallbackRuntime, error) {
	if server == nil || reg.ValidateClientIdentity() != nil || reg.CallbackURL != "" {
		return nil, errors.New("invalid callback setup")
	}
	listener, address, err := ccubinrpc.BindAvailable(host, first, last)
	if err != nil {
		return nil, err
	}
	reg.CallbackURL = address
	return &CallbackRuntime{Listener: listener, Registration: reg, Server: server, Registered: make(chan struct{}), LockPath: filepath.Join(os.TempDir(), "ccu-modbusclient-rpc.lock")}, nil
}

// ServeRegistered registers only after successful bind, then serves until cancellation.
// Registration failure releases the reservation and does not start the server.
func (r *CallbackRuntime) ServeRegistered(ctx context.Context) error {
	if r == nil || r.Listener == nil || r.Server == nil || ctx == nil {
		return errors.New("invalid callback runtime")
	}
	defer r.Listener.Close()
	lock, err := lockClientInstance(r.LockPath)
	if err != nil {
		return err
	}
	defer unlockClientInstance(lock)
	// Serve callback requests before init: CCU can query during registration.
	serverCtx, stopServer := context.WithCancel(ctx)
	serverDone := make(chan struct{})
	var serveError error
	go func() {
		serveError = r.Server.Serve(serverCtx, r.Listener)
		close(serverDone)
	}()
	defer func() { stopServer(); _ = r.Listener.Close(); <-serverDone }()
	if r.RegistrationProbe == nil {
		if err := r.Registration.RetryInit(ctx); err != nil {
			return err
		}
	} else {
		// The registration monitor owns re-registration; it starts before serving.
		manager := &RegistrationManager{Registration: r.Registration, Probe: r.RegistrationProbe, Interval: r.ProbeInterval}
		monitorCtx, stopMonitor := context.WithCancel(ctx)
		monitorDone := make(chan error, 1)
		go func() { monitorDone <- manager.Run(monitorCtx) }()
		// Do not expose a successful registration until the monitor confirms init.
		for !manager.Connected() {
			select {
			case err := <-monitorDone:
				stopMonitor()
				return err
			case <-ctx.Done():
				stopMonitor()
				<-monitorDone
				return ctx.Err()
			case <-time.After(5 * time.Millisecond):
			}
		}
		defer func() { stopMonitor(); <-monitorDone }()
	}
	close(r.Registered)
	defer func() { // Best effort: use a fresh bounded context when caller cancelled.
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = r.Registration.Deinit(cleanup)
	}()
	select {
	case <-serverDone:
		return serveError
	case <-ctx.Done():
		return nil
	}
}
