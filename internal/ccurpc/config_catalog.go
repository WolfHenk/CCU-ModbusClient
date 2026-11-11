// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/config_catalog.go
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
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
)

// CatalogFromConfig builds a disconnected model from the persistent config.
// An explicit stable CCU address is required: this prevents accidental
// generation of a new address and loss of CCU device identity.
func CatalogFromConfig(cfg *config.Config) (*MemoryCatalog, error) {
	if cfg == nil {
		return nil, fmt.Errorf("missing configuration")
	}
	result := NewMemoryCatalog()
	for _, d := range cfg.Devices {
		if !d.Enabled {
			continue
		}
		address := strings.TrimSpace(d.CCUAddress)
		if address == "" {
			return nil, fmt.Errorf("device %s has no persistent CCU address", d.ID)
		}
		channels := []Channel{{Address: address + ":0", Type: "MAINTENANCE", Version: 3, Points: []Point{
			{ID: "UNREACH", Type: "BOOL", Value: Boolean(false), Read: true, Event: true, Flags: 9, Default: boolValue(false), Minimum: boolValue(false), Maximum: boolValue(true)},
			{ID: "MODBUS_ERROR", Type: "INTEGER", Value: Integer(0), Read: true, Event: true, Flags: 9, Default: intValue(0), Minimum: intValue(0), Maximum: intValue(0xFF0B)},
			{ID: "MODBUS_EXCEPTION", Type: "BOOL", Value: Boolean(false), Read: true, Event: true, Flags: 9, Default: boolValue(false), Minimum: boolValue(false), Maximum: boolValue(true)},
			InstallTestPoint(), {ID: "STICKY_UNREACH", Type: "BOOL", Value: Boolean(false), Read: true, Write: true, Event: true, Flags: 25, Default: boolValue(false), Minimum: boolValue(false), Maximum: boolValue(true)}}}}
		// stable channel numbers: see channelPlan and config/channels.go
		for _, pc := range channelPlan(d) {
			r := pc.Reg
			if pc.Binary {
				var typ string
				var writable bool
				if r.Type == "coil" && r.CoilDirection != "input" {
					typ = "SWITCH"
					writable = true
				} else {
					typ = "DIGITAL_INPUT"
				}
				control := "SWITCH_TRANSMITTER.STATE"
				if writable {
					control = "SWITCH.STATE"
				}
				channels = append(channels, Channel{Address: fmt.Sprintf("%s:%d", address, pc.Index), Type: typ, Points: []Point{{ID: "STATE", Type: "BOOL", Value: Boolean(false), Read: true, Write: writable, Event: true, Flags: 1, Control: control, Default: boolValue(false), Minimum: boolValue(false), Maximum: boolValue(true)}, InstallTestPoint()}})
				continue
			}
			nc := numericChannelFor(r)
			point := Point{ID: nc.Key, Type: nc.DataType, Unit: r.Unit, Value: String(""), Read: true, Write: nc.Writable, Event: true, Flags: 1}
			switch nc.DataType {
			case "STRING":
				point.Default = stringValue("")
				point.Minimum = stringValue("")
				point.Maximum = stringValue("")
			case "INTEGER":
				point.Value = Integer(0)
				point.Control = "NONE"
				point.Default = intValue(0)
				point.Minimum = intValue(0)
				point.Maximum = intValue(100)
			case "FLOAT":
				point.Value = Value{Kind: "double", Text: "0"}
				point.Control = "NONE"
				point.Default = doubleValue(0)
				point.Minimum = doubleValue(-1000000000)
				point.Maximum = doubleValue(1000000000)
				if nc.Writable {
					if low, high, valid := outputDisplayRange(r); valid {
						point.Minimum = doubleValue(low)
						point.Maximum = doubleValue(high)
					}
				}
			}
			ch := Channel{Address: fmt.Sprintf("%s:%d", address, pc.Index), Type: nc.Type, Points: []Point{point}}
			channels = append(channels, ch)
		}
		if err := result.AddDevice(Device{Address: address, Type: "Modbus", Channels: channels}); err != nil {
			return nil, fmt.Errorf("device %s: %w", d.ID, err)
		}
	}
	return result, nil
}

// numericChannel describes how a numeric register appears in the CCU.
type numericChannel struct {
	Type, Key, DataType string
	Writable            bool
}

// numericChannelFor maps a numeric register to an existing CCU channel type
// whose data point already carries a WebUI label, so no WebUI file has to be
// overlaid. Readable values ("Ist") follow the configured measurement; every
// writable value ("Soll") uses ANALOG_OUTPUT_TRANSCEIVER.LEVEL ("Ausgangspegel")
// with the configured unit. BIN-RPC/XML-RPC writes, events and the catalog all
// use this single mapping.
func numericChannelFor(r config.Register) numericChannel {
	switch strings.ToLower(r.DataType) {
	case "uint64", "int64":
		return numericChannel{"DIGITAL_ANALOG_INPUT", "VALUE", "STRING", false}
	}
	if r.Type == "holding" && r.NumericDirection == "output" {
		return numericChannel{"ANALOG_OUTPUT_TRANSCEIVER", "LEVEL", "FLOAT", true}
	}
	switch r.Measurement {
	case "temperature":
		return numericChannel{"WEATHER_TRANSMIT", "TEMPERATURE", "FLOAT", false}
	case "humidity":
		// The WebUI renders HUMIDITY only correctly as INTEGER.
		return numericChannel{"WEATHER_TRANSMIT", "HUMIDITY", "INTEGER", false}
	case "power":
		return numericChannel{"POWERMETER", "POWER", "FLOAT", false}
	case "energy":
		return numericChannel{"POWERMETER", "ENERGY_COUNTER", "FLOAT", false}
	case "current":
		return numericChannel{"POWERMETER", "CURRENT", "FLOAT", false}
	case "voltage":
		return numericChannel{"POWERMETER", "VOLTAGE", "FLOAT", false}
	case "frequency":
		return numericChannel{"POWERMETER", "FREQUENCY", "FLOAT", false}
	}
	return numericChannel{"DIGITAL_ANALOG_INPUT", "VALUE", "FLOAT", false}
}

func floatPtr(v float64) *float64 { return &v }

func boolValue(b bool) *Value { v := Boolean(b); return &v }
func intValue(n int) *Value   { v := Integer(n); return &v }

func doubleValue(n float64) *Value {
	v := Value{Kind: "double", Text: strconv.FormatFloat(n, 'f', -1, 64)}
	return &v
}

func stringValue(v string) *Value { x := String(v); return &x }

// outputDisplayRange derives the CCU engineering-unit limits independently
// from the integer representation and scaling of the configured register.
func outputDisplayRange(r config.Register) (float64, float64, bool) {
	var lo, hi float64
	switch strings.ToLower(r.DataType) {
	case "uint8":
		lo, hi = 0, 255
	case "int8":
		lo, hi = -128, 127
	case "uint16":
		lo, hi = 0, 65535
	case "int16":
		lo, hi = -32768, 32767
	case "uint32":
		lo, hi = 0, 4294967295
	case "int32":
		lo, hi = -2147483648, 2147483647
	default:
		return 0, 0, false
	}
	lo, hi = lo*r.Factor+r.Offset, hi*r.Factor+r.Offset
	if math.IsNaN(lo) || math.IsNaN(hi) || math.IsInf(lo, 0) || math.IsInf(hi, 0) {
		return 0, 0, false
	}
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi, true
}
