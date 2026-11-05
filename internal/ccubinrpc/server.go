// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/server.go
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
	"sync"
	"time"
)

// Procedure is an independently registered BinRPC method.
type Procedure func([]Parameter) (Parameter, error)

// TCPServer handles a single bounded request per accepted TCP connection.
// The caller owns the listener and its shutdown lifecycle.
type TCPServer struct {
	Timeout time.Duration
	mu      sync.RWMutex
	methods map[string]Procedure
}

func NewTCPServer() *TCPServer { return &TCPServer{methods: make(map[string]Procedure)} }
func (s *TCPServer) Register(name string, fn Procedure) error {
	if s == nil || fn == nil || name == "" {
		return errors.New("invalid BinRPC procedure")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.methods == nil {
		s.methods = make(map[string]Procedure)
	}
	if _, exists := s.methods[name]; exists {
		return errors.New("duplicate BinRPC procedure")
	}
	s.methods[name] = fn
	return nil
}
func (s *TCPServer) ServeConn(conn net.Conn) {
	defer conn.Close()
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	name, args, err := DecodeRequest(conn)
	if err != nil {
		_ = EncodeFault(conn, -32600, "invalid BinRPC request")
		return
	}
	s.mu.RLock()
	fn := s.methods[name]
	s.mu.RUnlock()
	if fn == nil {
		_ = EncodeFault(conn, -32601, "unknown BinRPC method")
		return
	}
	value, err := fn(args)
	if err != nil {
		_ = EncodeFault(conn, -32500, "BinRPC method failed")
		return
	}
	_ = EncodeResponse(conn, value)
}

// Serve accepts connections until the context is cancelled or the listener closes.
// The listener is closed on cancellation, and connection workers have deadlines.
func (s *TCPServer) Serve(ctx context.Context, listener net.Listener) error {
	if s == nil || ctx == nil || listener == nil {
		return errors.New("nil BinRPC server input")
	}
	finished := make(chan struct{})
	defer close(finished)
	go func() {
		select {
		case <-ctx.Done():
			_ = listener.Close()
		case <-finished:
		}
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go s.ServeConn(conn)
	}
}
