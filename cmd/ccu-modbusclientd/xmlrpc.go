// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        cmd/ccu-modbusclientd/xmlrpc.go
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

package main

import (
	"context"
	"fmt"
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccurpc"
	"github.com/WolfHenk/CCU-ModbusClient/internal/engine"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func serveXMLRPC(ctx context.Context, path, addr string) error {
	cfg, err := loadClientConfig(path, false)
	if err != nil {
		return err
	}
	catalog, err := ccurpc.CatalogFromClientConfig(cfg)
	if err != nil {
		return err
	}
	eng := engine.New()
	eng.Start(ctx, cfg)
	defer eng.Stop()
	go func() {
		if err := ccurpc.PollClientSnapshots(ctx, catalog, cfg, eng, 500*time.Millisecond); err != nil {
			log.Printf("client snapshot: %v", err)
		}
	}()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()
	handler := ccurpc.NewHandler()
	if err := ccurpc.RegisterCCUReadMethods(handler, catalog); err != nil {
		return err
	}
	if err := ccurpc.RegisterXMLRPCWrites(handler, cfg, eng); err != nil {
		return err
	}
	queue, err := ccurpc.NewEventQueue(1024)
	if err != nil {
		return err
	}
	go func() {
		if err := ccurpc.PumpClientEvents(ctx, catalog, queue, 200*time.Millisecond); err != nil {
			log.Printf("event pump: %v", err)
		}
	}()
	var deliveryCancel context.CancelFunc
	var deliveryMu sync.Mutex
	defer func() {
		deliveryMu.Lock()
		if deliveryCancel != nil {
			deliveryCancel()
		}
		deliveryMu.Unlock()
	}()
	registration := ccurpc.NewInterfaceRegistration()
	registration.OnInit = func(receiver, id string) error {
		if _, err := strconv.ParseUint(id, 10, 32); err != nil {
			return fmt.Errorf("non-numeric ReGa interface ID: %w", err)
		}
		if err := ccurpc.AnnounceDevices(ctx, ccurpc.Client{URL: "http://127.0.0.1:1999/"}, id, catalog); err != nil {
			return err
		}
		deliveryMu.Lock()
		if deliveryCancel != nil {
			deliveryCancel()
		}
		deliveryCtx, cancel := context.WithCancel(ctx)
		deliveryCancel = cancel
		deliveryMu.Unlock()
		go func() {
			if err := ccurpc.DeliverEventsForInterface(deliveryCtx, queue, ccurpc.Client{URL: "http://127.0.0.1:1999/"}, id); err != nil {
				log.Printf("event delivery: %v", err)
			}
		}()
		return nil
	}
	if err := registration.Register(handler); err != nil {
		return err
	}
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			if ctx.Err() != nil {
				return
			}
			const receiver = "xmlrpc_bin://127.0.0.1:31999"
			if _, ok := registration.Receiver(receiver); ok {
				return
			}
			id, err := ccurpc.ResolveClientInterfaceID(ctx, "http://127.0.0.1:8181/rega.exe")
			if err == nil {
				err = registration.Bootstrap(receiver, id)
				if err == nil {
					log.Printf("ReGa auto-registration successful: interface %s", id)
					return
				}
			}
			if err != nil {
				log.Printf("ReGa auto-registration retry: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/RPC3", handler)
	server := &http.Server{Handler: mux}
	go func() { <-ctx.Done(); _ = server.Close() }()
	err = server.Serve(listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
