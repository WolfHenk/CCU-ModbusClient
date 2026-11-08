// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/xmlrpc_write.go
// Ersteller:    Wolfram Henkel
// Datum:        06. Nov. 2026
// Version:      0.3.1 (XML-RPC write correction)
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
	"math"
	"strconv"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
)

// RegisterXMLRPCWrites exposes setValue on the XML-RPC endpoint registered with
// OpenCCU. It shares BIN-RPC's device/channel mapping and write permissions.
// A successful response means queued, not that the Modbus write completed.
func RegisterXMLRPCWrites(handler *Handler, cfg *config.Config, queue WriteQueue) error {
	if handler == nil || cfg == nil || queue == nil {
		return errors.New("missing write endpoint prerequisites")
	}
	routes, err := buildWriteRoutes(cfg)
	if err != nil {
		return err
	}
	catalog, err := CatalogFromClientConfig(cfg)
	if err != nil {
		return err
	}
	return handler.Register("setValue", func(args []Value) (Value, error) {
		if len(args) != 3 || args[0].Kind != "string" || args[1].Kind != "string" {
			return Value{}, rpcInputError("setValue requires channel, datapoint and value")
		}
		route, found := routes[args[0].Text+"."+args[1].Text]
		if !found {
			if meta, err := catalog.ParamsetDescription(args[0].Text, "VALUES"); err == nil {
				if _, exists := meta.Members[args[1].Text]; exists {
					return Value{}, rpcInputError("datapoint is read-only")
				}
			}
			return Value{}, rpcInputError("unknown datapoint")
		}
		var accepted bool
		switch route.kind {
		case "bool":
			if args[2].Kind != "boolean" {
				return Value{}, rpcInputError("STATE requires boolean")
			}
			var state bool
			switch args[2].Text {
			case "1", "true":
				state = true
			case "0", "false":
				state = false
			default:
				return Value{}, rpcInputError("invalid boolean value")
			}
			accepted = queue.EnqueueCoil(route.device, route.register, state)
		case "numeric":
			if args[2].Kind != "int" && args[2].Kind != "double" && args[2].Kind != "string" {
				return Value{}, rpcInputError("VALUE requires numeric input")
			}
			value, parseErr := strconv.ParseFloat(args[2].Text, 64)
			if parseErr != nil {
				return Value{}, rpcInputError("invalid numeric VALUE")
			}
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return Value{}, rpcInputError("numeric VALUE must be finite")
			}
			accepted = queue.EnqueueNumeric(route.device, route.register, value)
		default:
			return Value{}, rpcInputError("unsupported writable datapoint type")
		}
		if !accepted {
			return Value{}, rpcInputError("write not accepted by Modbus engine")
		}
		return String(""), nil
	})
}
