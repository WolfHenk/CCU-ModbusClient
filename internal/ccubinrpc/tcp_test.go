// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/tcp_test.go
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

func TestTCPClientLocal(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	serverDone := make(chan error, 1)
	go func() {
		c, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(time.Second))
		method, args, err := DecodeRequest(c)
		if err != nil {
			serverDone <- err
			return
		}
		if method != "init" || len(args) != 2 || args[1].Text != "test" {
			serverDone <- errors.New("unexpected request")
			return
		}
		serverDone <- EncodeResponse(c, Parameter{Type: TypeBoolean, Flag: true})
	}()
	got, err := (TCPClient{Address: listener.Addr().String(), Timeout: time.Second}).Call(context.Background(), "init", Parameter{Type: TypeString, Text: "receiver"}, Parameter{Type: TypeString, Text: "test"})
	if err != nil || got.Type != TypeBoolean || !got.Flag {
		t.Fatalf("result %+v err %v", got, err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}
func TestTCPClientRemoteTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{})
	go func() {
		c, err := listener.Accept()
		if err != nil {
			return
		}
		close(accepted)
		defer c.Close()
		_, _, _ = DecodeRequest(c)
		time.Sleep(250 * time.Millisecond)
	}()
	_, err = (TCPClient{Address: listener.Addr().String(), Timeout: 35 * time.Millisecond}).Call(context.Background(), "init")
	if err == nil {
		t.Fatal("timeout not enforced")
	}
	<-accepted
}
func TestTCPClientCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (TCPClient{Address: "127.0.0.1:1"}).Call(ctx, "init")
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("context cancellation not observed: %v", err)
	}
}
