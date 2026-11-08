// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/binrpc_write_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"testing"
	"time"
)

type writeSpy struct {
	coils int
	nums  int
	last  float64
}

func (w *writeSpy) EnqueueCoil(d, r string, b bool) bool {
	if d != "d" || r != "switch" || !b {
		return false
	}
	w.coils++
	return true
}
func (w *writeSpy) EnqueueNumeric(d, r string, n float64) bool {
	if d != "d" || r != "out" {
		return false
	}
	w.nums++
	w.last = n
	return true
}
func TestBinRPCSetValueOutputGuards(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "d", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1, Registers: []config.Register{{ID: "switch", Enabled: true, Type: "coil", DataType: "bool"}, {ID: "input", Enabled: true, Type: "discrete", DataType: "bool"}, {ID: "out", Enabled: true, Type: "holding", DataType: "float32", NumericDirection: "output"}}}}}
	spy := new(writeSpy)
	s := ccubinrpc.NewTCPServer()
	if e := RegisterBinRPCWrites(s, cfg, spy); e != nil {
		t.Fatal(e)
	}
	runtime, e := ReserveCallback("127.0.0.1", 32300, 32309, s, NewClientRegistration(Client{}))
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, runtime.Listener) }()
	client := ccubinrpc.TCPClient{Address: runtime.Listener.Addr().String(), Timeout: time.Second}
	str := func(v string) ccubinrpc.Parameter { return ccubinrpc.Parameter{Type: ccubinrpc.TypeString, Text: v} }
	if _, e := client.Call(ctx, "setValue", str("MBC03CF01F601:1"), str("STATE"), ccubinrpc.Parameter{Type: ccubinrpc.TypeBoolean, Flag: true}); e != nil {
		t.Fatal(e)
	}
	if _, e := client.Call(ctx, "setValue", str("MBC03CF01F601:3"), str("LEVEL"), ccubinrpc.Parameter{Type: ccubinrpc.TypeDouble, Float: 12.5}); e != nil {
		t.Fatal(e)
	}
	if _, e := client.Call(ctx, "setValue", str("MBC03CF01F601:2"), str("STATE"), ccubinrpc.Parameter{Type: ccubinrpc.TypeBoolean, Flag: true}); e == nil {
		t.Fatal("input write accepted")
	}
	if _, e := client.Call(ctx, "setValue", str("MBC03CF01F601:1"), str("STATE"), str("true")); e == nil {
		t.Fatal("wrong type accepted")
	}
	if spy.coils != 1 || spy.nums != 1 || spy.last != 12.5 {
		t.Fatalf("writes: %+v", spy)
	}
	cancel()
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown blocked")
	}
}
