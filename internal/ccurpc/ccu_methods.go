// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/ccu_methods.go
// Ersteller:    Wolfram Henkel
// Datum:        05. Nov. 2026
// Version:      0.3.3
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
	"fmt"
	"sort"
	"strconv"
)

// DeviceCatalog supplies immutable device descriptions to a future CCU
// interface adapter. It is intentionally independent of go-hmccu.
type DeviceCatalog interface {
	ListDevices() ([]Value, error)
	DeviceDescription(address string) (Value, error)
	ParamsetDescription(address, set string) (Value, error)
	Paramset(address, set string) (Value, error)
}

// RegisterCCUReadMethods wires the read-only CCU method subset. Device
// mutation and ReGa registration must be added only after independent tests.
func RegisterCCUReadMethods(h *Handler, catalog DeviceCatalog) error {
	if h == nil || catalog == nil {
		return fmt.Errorf("CCU RPC handler or catalog missing")
	}
	specs := []struct {
		name string
		fn   Method
	}{
		// OpenCCU's Interface.isPresent calls system.listMethods every 30 seconds.
		// Advertise the methods actually registered on this XML-RPC endpoint.
		{"system.listMethods", func(args []Value) (Value, error) {
			if len(args) != 0 {
				return Value{}, rpcInputError("system.listMethods expects no arguments")
			}
			h.mu.RLock()
			names := make([]string, 0, len(h.methods))
			for name := range h.methods {
				names = append(names, name)
			}
			h.mu.RUnlock()
			sort.Strings(names)
			methods := make([]Value, 0, len(names))
			for _, name := range names {
				methods = append(methods, String(name))
			}
			return Array(methods...), nil
		}},
		{"listDevices", func(args []Value) (Value, error) {
			if len(args) != 0 {
				return Value{}, fmt.Errorf("listDevices expects no arguments")
			}
			items, e := catalog.ListDevices()
			if e != nil {
				return Value{}, e
			}
			return Array(items...), nil
		}},
		{"getDeviceDescription", func(args []Value) (Value, error) {
			if len(args) != 1 || args[0].Kind != "string" {
				return Value{}, fmt.Errorf("getDeviceDescription expects address")
			}
			return catalog.DeviceDescription(args[0].Text)
		}},
		{"getParamsetDescription", func(args []Value) (Value, error) {
			address, set, e := twoStrings(args)
			if e != nil {
				return Value{}, e
			}
			return catalog.ParamsetDescription(address, set)
		}},
		{"getParamset", func(args []Value) (Value, error) {
			address, set, e := twoStrings(args)
			if e != nil {
				return Value{}, e
			}
			return catalog.Paramset(address, set)
		}},
		{"getValue", func(args []Value) (Value, error) {
			address, key, e := twoStrings(args)
			if e != nil {
				return Value{}, rpcInputError("getValue requires channel address and datapoint name")
			}
			meta, e := catalog.ParamsetDescription(address, "VALUES")
			if e != nil {
				return Value{}, rpcInputError("unknown channel")
			}
			entry, found := meta.Members[key]
			if !found {
				return Value{}, rpcInputError("unknown datapoint")
			}
			operations, e := strconv.Atoi(entry.Members["OPERATIONS"].Text)
			if entry.Kind != "struct" || e != nil || operations&1 == 0 {
				return Value{}, rpcInputError("datapoint not readable")
			}
			// Use the same freshly updated cache as getParamset/events. No
			// additional Modbus read or device state mutation is performed.
			values, e := catalog.Paramset(address, "VALUES")
			if e != nil {
				return Value{}, e
			}
			value, ok := values.Members[key]
			if !ok {
				return Value{}, rpcInputError("unknown datapoint")
			}
			return value, nil
		}},
		// CCU expects an array of interface service messages. The virtual
		// Modbus devices do not currently generate Homematic service messages.
		{"getServiceMessages", func(args []Value) (Value, error) {
			if len(args) != 0 {
				return Value{}, rpcInputError("getServiceMessages expects no arguments")
			}
			return Array(), nil
		}},
		{"getLinks", func(args []Value) (Value, error) {
			if len(args) != 2 || args[0].Kind != "string" || args[1].Kind != "int" {
				return Value{}, fmt.Errorf("getLinks expects address and integer flags")
			}
			if _, err := catalog.DeviceDescription(args[0].Text); err != nil {
				return Value{}, err
			}
			return Array(), nil
		}},
		{"setReadyConfig", func(args []Value) (Value, error) {
			if len(args) > 1 {
				return Value{}, fmt.Errorf("setReadyConfig expects zero or one argument")
			}
			return String(""), nil
		}},
		// ReGa reports how many of its objects use a datapoint, e.g. when it
		// removes datapoints after a channel type changed. Modbus has no
		// on-demand polling to adjust, so the call is only acknowledged. The
		// address is not checked against the catalog because ReGa also sends
		// it for datapoints that no longer exist.
		{"reportValueUsage", func(args []Value) (Value, error) {
			if len(args) != 3 || args[0].Kind != "string" || args[1].Kind != "string" || args[2].Kind != "int" {
				return Value{}, rpcInputError("reportValueUsage requires address, datapoint name and usage count")
			}
			return Boolean(true), nil
		}},
	}
	for _, spec := range specs {
		if err := h.Register(spec.name, spec.fn); err != nil {
			return err
		}
	}
	return nil
}
func twoStrings(args []Value) (string, string, error) {
	if len(args) != 2 || args[0].Kind != "string" || args[1].Kind != "string" {
		return "", "", fmt.Errorf("expected address and paramset name")
	}
	return args[0].Text, args[1].Text, nil
}
