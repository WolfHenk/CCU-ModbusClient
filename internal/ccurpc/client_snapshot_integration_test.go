// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_snapshot_integration_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
	"testing"
	"time"
)

type fakeSnapshotSource struct{ states []model.DeviceState }

func (f fakeSnapshotSource) Snapshot() []model.DeviceState { return f.states }
func TestClientSnapshotAppearsOverBinRPC(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "test", Enabled: true, Host: "192.168.3.207", Port: 502, UnitID: 1, Registers: []config.Register{{ID: "door", Enabled: true, Type: "discrete", DataType: "bool"}}}}}
	catalog, err := CatalogFromClientConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	source := fakeSnapshotSource{states: []model.DeviceState{{ID: "test", State: "ONLINE", Registers: []model.RegisterState{{ID: "door", Value: model.Value{Quality: model.QualityGood, Value: true}}}}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pollDone := make(chan error, 1)
	go func() { pollDone <- PollClientSnapshots(ctx, catalog, cfg, source, 10*time.Millisecond) }()
	server := ccubinrpc.NewTCPServer()
	if err := RegisterBinRPCReadMethods(server, catalog); err != nil {
		t.Fatal(err)
	}
	runtime, err := ReserveCallback("127.0.0.1", 32250, 32259, server, NewClientRegistration(Client{}))
	if err != nil {
		t.Fatal(err)
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.Serve(ctx, runtime.Listener) }()
	client := ccubinrpc.TCPClient{Address: runtime.Listener.Addr().String(), Timeout: time.Second}
	var got ccubinrpc.Parameter
	matched := false
	for i := 0; i < 50; i++ {
		got, err = client.Call(context.Background(), "getParamset", ccubinrpc.Parameter{Type: ccubinrpc.TypeString, Text: "MBC03CF01F601:1"}, ccubinrpc.Parameter{Type: ccubinrpc.TypeString, Text: "VALUES"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Fields["STATE"].Flag {
			matched = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !matched {
		t.Fatalf("changed Modbus state not visible: %+v", got)
	}
	if cfg.Devices[0].CCUAddress != "" {
		t.Fatal("snapshot polling changed stored address")
	}
	cancel()
	select {
	case e := <-serverDone:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("server stop timeout")
	}
	select {
	case e := <-pollDone:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("poller stop timeout")
	}
}
