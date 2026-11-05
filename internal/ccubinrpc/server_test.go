// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/server_test.go
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
	"strings"
	"testing"
	"time"
)

func TestBinRPCTCPServerIntegration(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := NewTCPServer()
	server.Timeout = time.Second
	if err := server.Register("event", func(p []Parameter) (Parameter, error) {
		if len(p) != 2 || p[0].Text != "CCU-Modbus" || p[1].Number != 42 {
			t.Errorf("wrong RPC arguments: %+v", p)
		}
		return Parameter{Type: TypeBoolean, Flag: true}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := server.Register("fail", func([]Parameter) (Parameter, error) { return Parameter{}, errors.New("secret failure") }); err != nil {
		t.Fatal(err)
	}
	if err := server.Register("event", func([]Parameter) (Parameter, error) { return Parameter{}, nil }); err == nil {
		t.Fatal("duplicate accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, ln) }()
	c := TCPClient{Address: ln.Addr().String(), Timeout: time.Second}
	got, err := c.Call(context.Background(), "event", Parameter{Type: TypeString, Text: "CCU-Modbus"}, Parameter{Type: TypeInteger, Number: 42})
	if err != nil || !got.Flag {
		t.Fatalf("result %+v err %v", got, err)
	}
	for _, name := range []string{"missing", "fail"} {
		_, err := c.Call(context.Background(), name)
		if err == nil || !strings.Contains(err.Error(), "BinRPC fault") {
			t.Fatalf("%s missing fault: %v", name, err)
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatal("internal error leaked")
		}
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("server failed to stop")
	}
}
