// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/binrpc_adapter_test.go
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

func TestClientCatalogOverBinRPC(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "unit1", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1}}}
	catalog, e := CatalogFromClientConfig(cfg)
	if e != nil {
		t.Fatal(e)
	}
	server := ccubinrpc.NewTCPServer()
	if e := RegisterBinRPCReadMethods(server, catalog); e != nil {
		t.Fatal(e)
	}
	runtime, e := ReserveCallback("127.0.0.1", 32200, 32209, server, NewClientRegistration(Client{}))
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, runtime.Listener) }()
	client := ccubinrpc.TCPClient{Address: runtime.Listener.Addr().String(), Timeout: time.Second}
	result, e := client.Call(context.Background(), "getDeviceDescription", ccubinrpc.Parameter{Type: ccubinrpc.TypeString, Text: "MBC03CF01F601"})
	if e != nil {
		t.Fatal(e)
	}
	if result.Type != ccubinrpc.TypeStruct || result.Fields["ADDRESS"].Text != "MBC03CF01F601" {
		t.Fatalf("wrong description: %+v", result)
	}
	list, e := client.Call(context.Background(), "listDevices")
	if e != nil || list.Type != ccubinrpc.TypeArray || len(list.Items) != 2 {
		t.Fatalf("list=%+v err=%v", list, e)
	}
	cancel()
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown stalled")
	}
}
