// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/channel_types_test.go
// Ersteller:    Wolfram Henkel
// Datum:        08. Nov. 2026
// Version:      0.3.2
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
	"testing"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
)

// Every numeric register uses an existing CCU channel type with a WebUI label.
func TestNumericChannelMapping(t *testing.T) {
	in := func(m string) config.Register {
		return config.Register{Type: "input", DataType: "int16", Measurement: m}
	}
	for _, c := range []struct {
		r               config.Register
		typ, key, dtype string
		writable        bool
	}{
		{in(""), "DIGITAL_ANALOG_INPUT", "VALUE", "FLOAT", false},
		{in("generic"), "DIGITAL_ANALOG_INPUT", "VALUE", "FLOAT", false},
		{in("number"), "DIGITAL_ANALOG_INPUT", "VALUE", "FLOAT", false},
		{in("temperature"), "WEATHER_TRANSMIT", "TEMPERATURE", "FLOAT", false},
		{in("humidity"), "WEATHER_TRANSMIT", "HUMIDITY", "INTEGER", false},
		{in("power"), "POWERMETER", "POWER", "FLOAT", false},
		{in("energy"), "POWERMETER", "ENERGY_COUNTER", "FLOAT", false},
		{in("current"), "POWERMETER", "CURRENT", "FLOAT", false},
		{in("voltage"), "POWERMETER", "VOLTAGE", "FLOAT", false},
		{in("frequency"), "POWERMETER", "FREQUENCY", "FLOAT", false},
		{config.Register{Type: "holding", DataType: "int16"}, "DIGITAL_ANALOG_INPUT", "VALUE", "FLOAT", false},
		{config.Register{Type: "holding", DataType: "int16", NumericDirection: "output"}, "ANALOG_OUTPUT_TRANSCEIVER", "LEVEL", "FLOAT", true},
		{config.Register{Type: "holding", DataType: "float32", NumericDirection: "output", Measurement: "temperature"}, "ANALOG_OUTPUT_TRANSCEIVER", "LEVEL", "FLOAT", true},
		{config.Register{Type: "input", DataType: "uint64"}, "DIGITAL_ANALOG_INPUT", "VALUE", "STRING", false},
	} {
		got := numericChannelFor(c.r)
		if got.Type != c.typ || got.Key != c.key || got.DataType != c.dtype || got.Writable != c.writable {
			t.Errorf("%+v: got %+v", c.r, got)
		}
	}
}

type rejectQueue struct{ reason error }

func (rejectQueue) EnqueueCoil(string, string, bool) bool        { return false }
func (rejectQueue) EnqueueNumeric(string, string, float64) bool  { return false }
func (q rejectQueue) CheckNumeric(string, string, float64) error { return q.reason }

func TestNumericRejectReasonNamesCause(t *testing.T) {
	got := numericRejectReason(rejectQueue{reason: errors.New("Sollwert 300 ausserhalb des uint8-Bereichs")}, "d", "r", 300)
	if got != "write rejected: Sollwert 300 ausserhalb des uint8-Bereichs" {
		t.Fatalf("reason = %q", got)
	}
	if got := numericRejectReason(rejectQueue{}, "d", "r", 1); got != "write not accepted by Modbus engine" {
		t.Fatalf("fallback = %q", got)
	}
}
