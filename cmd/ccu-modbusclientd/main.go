// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        cmd/ccu-modbusclientd/main.go
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

// ccu-modbusclientd is the isolated independent RPC prototype.
// No CCU files, old daemon, or legacy virtual interface are modified.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/WolfHenk/CCU-ModbusClient/internal/ccubinrpc"
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccurpc"
	"github.com/WolfHenk/CCU-ModbusClient/internal/engine"
	"time"
)

func run(ctx context.Context, path, ccuURL string, check bool, first, last int) error {
	return runWithEvents(ctx, path, ccuURL, "", check, first, last)
}

func runWithEvents(ctx context.Context, path, ccuURL, eventURL string, check bool, first, last int) error {
	if !check && ccuURL == "" {
		return fmt.Errorf("CCU RPC endpoint required; refusing implicit remote registration")
	}
	cfg, err := loadClientConfig(path, check)
	if err != nil {
		return err
	}
	catalog, err := ccurpc.CatalogFromClientConfig(cfg)
	if err != nil {
		return err
	}
	devices, err := catalog.ListDevices()
	if err != nil {
		return err
	}
	if check {
		fmt.Printf("CCU-ModbusClient configuration valid; %d virtual entries\n", len(devices))
		return nil
	}
	if ccuURL == "" {
		return fmt.Errorf("CCU RPC endpoint required; refusing implicit remote registration")
	}
	server := ccubinrpc.NewTCPServer()
	if err := ccurpc.RegisterBinRPCReadMethods(server, catalog); err != nil {
		return err
	}
	runtime, err := ccurpc.ReserveCallback("127.0.0.1", first, last, server, ccurpc.NewClientRegistration(ccurpc.Client{URL: ccuURL}))
	if err != nil {
		return err
	}
	eng := engine.New()
	if err := ccurpc.RegisterBinRPCWrites(server, cfg, eng); err != nil {
		runtime.Listener.Close()
		return err
	}
	eng.Start(ctx, cfg)
	defer eng.Stop()
	pollingCtx, stopPolling := context.WithCancel(ctx)
	defer stopPolling()
	pollErrors := make(chan error, 1)
	go func() { pollErrors <- ccurpc.PollClientSnapshots(pollingCtx, catalog, cfg, eng, 500*time.Millisecond) }()
	if eventURL != "" {
		queue, err := ccurpc.NewEventQueue(1024)
		if err != nil {
			runtime.Listener.Close()
			return err
		}
		if err := ccurpc.RegisterBinRPCPing(server, queue); err != nil {
			runtime.Listener.Close()
			return err
		}
		go func() {
			if err := ccurpc.PumpClientEvents(pollingCtx, catalog, queue, 200*time.Millisecond); err != nil {
				log.Printf("event pump: %v", err)
			}
		}()
		go func() {
			if err := ccurpc.DeliverClientEvents(pollingCtx, queue, ccurpc.Client{URL: eventURL}); err != nil {
				log.Printf("event delivery: %v", err)
			}
		}()
	}
	log.Printf("CCU-ModbusClient RPC callback %s; interface %s", runtime.Registration.CallbackURL, runtime.Registration.InterfaceID)
	return runtime.ServeRegistered(ctx)
}

var version = "0.3.2"

func main() {
	showVersion := flag.Bool("version", false, "print program version and exit")
	configPath := flag.String("config", "", "configuration JSON file")
	ccuURL := flag.String("ccu-rpc", "", "explicit CCU XML-RPC URL; omit for check-only")
	eventURL := flag.String("ccu-events", "", "explicit CCU event receiver URL; empty disables events")
	check := flag.Bool("check", false, "validate MBC identities without network access")
	xmlListen := flag.String("xmlrpc-listen", "", "serve independent OpenCCU XML-RPC on this address")
	first := flag.Int("port-first", 31999, "first callback port")
	last := flag.Int("port-last", 32030, "last callback port")
	purge := flag.Bool("purge-ccu", false, "remove all CCU-ModbusClient devices and the interface object from ReGa and exit (used by uninstall)")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	if *purge {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		res, err := ccurpc.PurgeFromReGa(ctx, ccurpc.ReGaEndpoints{
			ScriptURL:   "http://127.0.0.1:8181/rega.exe",
			CallbackURL: "http://127.0.0.1:1999/",
		})
		if err != nil {
			log.Print(err)
			os.Exit(1)
		}
		fmt.Printf("ReGa interface %q: %d devices deleted, interface object deleted=%v\n", res.InterfaceID, len(res.DeletedDevices), res.InterfaceDeleted)
		return
	}
	if *configPath == "" {
		log.Print("-config is required")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if *xmlListen != "" && !*check {
		if err := serveXMLRPC(ctx, *configPath, *xmlListen); err != nil {
			log.Print(err)
			os.Exit(1)
		}
		return
	}
	if err := runWithEvents(ctx, *configPath, *ccuURL, *eventURL, *check, *first, *last); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
