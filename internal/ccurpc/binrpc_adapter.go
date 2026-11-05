// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/binrpc_adapter.go
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
	"strconv"
)

// RegisterBinRPCReadMethods exposes the independent catalog on the callback
// listener. Device mutations are intentionally not supported here.
func RegisterBinRPCReadMethods(s *ccubinrpc.TCPServer, c DeviceCatalog) error {
	if s == nil || c == nil {
		return errors.New("missing BinRPC server or device catalog")
	}
	methods := []struct {
		name string
		call func([]ccubinrpc.Parameter) (Value, error)
	}{
		{"listDevices", func(args []ccubinrpc.Parameter) (Value, error) {
			if len(args) != 0 {
				return Value{}, errors.New("listDevices expects no args")
			}
			items, e := c.ListDevices()
			return Array(items...), e
		}},
		{"getDeviceDescription", func(args []ccubinrpc.Parameter) (Value, error) {
			if len(args) != 1 || args[0].Type != ccubinrpc.TypeString {
				return Value{}, errors.New("invalid address")
			}
			return c.DeviceDescription(args[0].Text)
		}},
		{"getParamsetDescription", func(args []ccubinrpc.Parameter) (Value, error) {
			if len(args) != 2 || args[0].Type != ccubinrpc.TypeString || args[1].Type != ccubinrpc.TypeString {
				return Value{}, errors.New("invalid paramset request")
			}
			return c.ParamsetDescription(args[0].Text, args[1].Text)
		}},
		{"getParamset", func(args []ccubinrpc.Parameter) (Value, error) {
			if len(args) != 2 || args[0].Type != ccubinrpc.TypeString || args[1].Type != ccubinrpc.TypeString {
				return Value{}, errors.New("invalid paramset request")
			}
			return c.Paramset(args[0].Text, args[1].Text)
		}},
	}
	for _, entry := range methods {
		fn := entry.call
		if err := s.Register(entry.name, func(args []ccubinrpc.Parameter) (ccubinrpc.Parameter, error) {
			v, e := fn(args)
			if e != nil {
				return ccubinrpc.Parameter{}, e
			}
			return toBinRPC(v, 0)
		}); err != nil {
			return err
		}
	}
	return nil
}
func toBinRPC(v Value, depth int) (ccubinrpc.Parameter, error) {
	if depth > 32 {
		return ccubinrpc.Parameter{}, errors.New("nested RPC value too deep")
	}
	switch v.Kind {
	case "string":
		return ccubinrpc.Parameter{Type: ccubinrpc.TypeString, Text: v.Text}, nil
	case "int":
		num, e := strconv.ParseInt(v.Text, 10, 32)
		if e != nil {
			return ccubinrpc.Parameter{}, e
		}
		return ccubinrpc.Parameter{Type: ccubinrpc.TypeInteger, Number: int32(num)}, nil
	case "boolean":
		return ccubinrpc.Parameter{Type: ccubinrpc.TypeBoolean, Flag: v.Text == "1"}, nil
	case "double":
		num, e := strconv.ParseFloat(v.Text, 64)
		if e != nil {
			return ccubinrpc.Parameter{}, e
		}
		return ccubinrpc.Parameter{Type: ccubinrpc.TypeDouble, Float: num}, nil
	case "array":
		a := ccubinrpc.Parameter{Type: ccubinrpc.TypeArray, Items: make([]ccubinrpc.Parameter, 0, len(v.Array))}
		for _, el := range v.Array {
			x, e := toBinRPC(el, depth+1)
			if e != nil {
				return ccubinrpc.Parameter{}, e
			}
			a.Items = append(a.Items, x)
		}
		return a, nil
	case "struct":
		a := ccubinrpc.Parameter{Type: ccubinrpc.TypeStruct, Fields: make(map[string]ccubinrpc.Parameter, len(v.Members))}
		for key, val := range v.Members {
			x, e := toBinRPC(val, depth+1)
			if e != nil {
				return ccubinrpc.Parameter{}, e
			}
			a.Fields[key] = x
		}
		return a, nil
	default:
		return ccubinrpc.Parameter{}, fmt.Errorf("unsupported RPC kind %q", v.Kind)
	}
}
