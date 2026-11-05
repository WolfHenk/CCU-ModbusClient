// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/binrpc_ping.go
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
	"errors"
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccubinrpc"
)

// RegisterBinRPCPing implements the legacy ping(callerID) handshake.
// Acknowledgement means its CENTRAL.PONG event was accepted by the outbound
// queue, never that delivery to the CCU has already succeeded.
func RegisterBinRPCPing(server *ccubinrpc.TCPServer, queue *EventQueue) error {
	if server == nil || queue == nil {
		return errors.New("missing ping server or outbound event queue")
	}
	return server.Register("ping", func(args []ccubinrpc.Parameter) (ccubinrpc.Parameter, error) {
		if len(args) != 1 || args[0].Type != ccubinrpc.TypeString {
			return ccubinrpc.Parameter{}, errors.New("ping expects caller ID string")
		}
		if err := queue.Enqueue(PointEvent{Address: "CENTRAL", Key: "PONG", Value: String(args[0].Text)}); err != nil {
			return ccubinrpc.Parameter{}, err
		}
		return ccubinrpc.Parameter{Type: ccubinrpc.TypeBoolean, Flag: true}, nil
	})
}
