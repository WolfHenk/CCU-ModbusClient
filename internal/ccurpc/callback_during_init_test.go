// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/callback_during_init_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccubinrpc"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCallbackIsAvailableDuringRegistration(t *testing.T) {
	rpc := NewHandler()
	callbackOK := make(chan bool, 1)
	if err := rpc.Register("init", func(args []Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return Value{}, context.Canceled
		}
		if len(args) == 1 {
			return String(""), nil
		}
		address := strings.TrimPrefix(args[0].Text, "xmlrpc_bin://")
		c := ccubinrpc.TCPClient{Address: address, Timeout: time.Second}
		answer, err := c.Call(context.Background(), "listDevices")
		callbackOK <- err == nil && answer.Type == ccubinrpc.TypeArray
		if err != nil {
			return Value{}, err
		}
		return String(""), nil
	}); err != nil {
		t.Fatal(err)
	}
	remote := httptest.NewServer(rpc)
	defer remote.Close()
	server := ccubinrpc.NewTCPServer()
	if err := server.Register("listDevices", func([]ccubinrpc.Parameter) (ccubinrpc.Parameter, error) {
		return ccubinrpc.Parameter{Type: ccubinrpc.TypeArray, Items: []ccubinrpc.Parameter{}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	runtime, err := ReserveCallback("127.0.0.1", 32500, 32509, server, NewClientRegistration(Client{URL: remote.URL}))
	if err != nil {
		t.Fatal(err)
	}
	runtime.LockPath = filepath.Join(t.TempDir(), "instance.lock")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runtime.ServeRegistered(ctx) }()
	select {
	case <-runtime.Registered:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("registration timed out")
	}
	select {
	case ok := <-callbackOK:
		if !ok {
			t.Fatal("callback unavailable during init")
		}
	case <-time.After(time.Second):
		t.Fatal("callback not attempted")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown timeout")
	}
}
