// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/binrpc_write.go
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
	"fmt"
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccubinrpc"
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"math"
	"strconv"
	"strings"
)

// WriteQueue implements the engine's existing validated write enqueue API.
type WriteQueue interface {
	EnqueueCoil(deviceID, registerID string, value bool) bool
	EnqueueNumeric(deviceID, registerID string, value float64) bool
}

// writeRoute is the shared, read-only CCU datapoint-to-Modbus register map.
type writeRoute struct{ device, register, kind string }

// buildWriteRoutes uses identical write permissions and channel numbering for
// BIN-RPC and XML-RPC. Input and disabled datapoints are never writable.
func buildWriteRoutes(cfg *config.Config) (map[string]writeRoute, error) {
	if cfg == nil {
		return nil, errors.New("missing write configuration")
	}
	routes := make(map[string]writeRoute)
	for _, d := range cfg.Devices {
		if !d.Enabled {
			continue
		}
		serial, e := ClientSerial(d.Host, d.Port, d.UnitID)
		if e != nil {
			return nil, e
		}
		if d.CCUAddress != "" && !IsClientSerialForEndpoint(d.CCUAddress, serial) {
			return nil, fmt.Errorf("device %s requires identity migration", d.ID)
		}
		serial = normalizedClientSerial(d.CCUAddress, serial)
		idx := 1
		binary := make([]config.Register, 0)
		numeric := make([]config.Register, 0)
		for _, r := range d.Registers {
			if !r.Enabled {
				continue
			}
			if strings.EqualFold(r.DataType, "bool") && (r.Type == "coil" || r.Type == "discrete") {
				binary = append(binary, r)
			} else if r.Type == "holding" || r.Type == "input" {
				numeric = append(numeric, r)
			}
		}
		for _, r := range binary {
			if r.Type == "coil" && r.CoilDirection != "input" {
				routes[fmt.Sprintf("%s:%d.STATE", serial, idx)] = writeRoute{d.ID, r.ID, "bool"}
			}
			idx++
		}
		for _, r := range numeric {
			if r.Type == "holding" && r.NumericDirection == "output" && r.DataType != "uint64" && r.DataType != "int64" {
				routes[fmt.Sprintf("%s:%d.VALUE", serial, idx)] = writeRoute{d.ID, r.ID, "numeric"}
			}
			idx++
		}
	}
	return routes, nil
}

// RegisterBinRPCWrites adds a strictly validated setValue endpoint.
// Success confirms enqueueing only, not a completed Modbus transaction.
func RegisterBinRPCWrites(server *ccubinrpc.TCPServer, cfg *config.Config, queue WriteQueue) error {
	if server == nil || cfg == nil || queue == nil {
		return errors.New("missing write endpoint prerequisites")
	}
	routes, err := buildWriteRoutes(cfg)
	if err != nil {
		return err
	}
	return server.Register("setValue", func(args []ccubinrpc.Parameter) (ccubinrpc.Parameter, error) {
		if len(args) != 3 || args[0].Type != ccubinrpc.TypeString || args[1].Type != ccubinrpc.TypeString {
			return ccubinrpc.Parameter{}, errors.New("setValue expects channel, key, value")
		}
		route, ok := routes[args[0].Text+"."+args[1].Text]
		if !ok {
			return ccubinrpc.Parameter{}, errors.New("unknown or read-only datapoint")
		}
		accepted := false
		switch route.kind {
		case "bool":
			if args[2].Type != ccubinrpc.TypeBoolean {
				return ccubinrpc.Parameter{}, errors.New("STATE requires boolean")
			}
			accepted = queue.EnqueueCoil(route.device, route.register, args[2].Flag)
		case "numeric":
			var n float64
			switch args[2].Type {
			case ccubinrpc.TypeInteger:
				n = float64(args[2].Number)
			case ccubinrpc.TypeDouble:
				n = args[2].Float
			case ccubinrpc.TypeString:
				val, e := strconv.ParseFloat(args[2].Text, 64)
				if e != nil {
					return ccubinrpc.Parameter{}, e
				}
				n = val
			default:
				return ccubinrpc.Parameter{}, errors.New("VALUE requires numeric input")
			}
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return ccubinrpc.Parameter{}, errors.New("nonfinite output")
			}
			accepted = queue.EnqueueNumeric(route.device, route.register, n)
		}
		if !accepted {
			return ccubinrpc.Parameter{}, errors.New("write not accepted by Modbus engine")
		}
		return ccubinrpc.Parameter{Type: ccubinrpc.TypeString, Text: ""}, nil
	})
}
