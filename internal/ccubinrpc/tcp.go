// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/tcp.go
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

package ccubinrpc

import (
	"context"
	"errors"
	"net"
	"time"
)

// TCPClient performs one bounded BinRPC transaction per TCP connection.
// It deliberately does not modify or register any OpenCCU devices.
type TCPClient struct {
	Address string
	Timeout time.Duration
}

func (c TCPClient) Call(ctx context.Context, method string, args ...Parameter) (Parameter, error) {
	if ctx == nil {
		return Parameter{}, errors.New("nil context")
	}
	if c.Address == "" {
		return Parameter{}, errors.New("missing BinRPC address")
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", c.Address)
	if err != nil {
		return Parameter{}, err
	}
	defer conn.Close()
	deadline := time.Now().Add(timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return Parameter{}, err
	}
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-finished:
		}
	}()
	if err := EncodeRequest(conn, method, args...); err != nil {
		return Parameter{}, err
	}
	return DecodeResult(conn)
}
