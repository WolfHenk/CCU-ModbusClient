// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/numeric_clamp_queue_test.go
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

package engine

import (
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"testing"
)

// Out-of-range writes are rejected instead of being clamped to the limit:
// a limit value the user did not request must never reach an actuator.
func TestEnqueueNumericRejectOutOfRange(t *testing.T) {
	for _, tc := range []struct {
		typ        string
		bad, limit float64
	}{
		{"int16", -40000, -32768}, {"int16", 40000, 32767},
		{"uint16", -1, 0}, {"uint16", 70000, 65535},
		{"int32", -3000000000, -2147483648}, {"int32", 3000000000, 2147483647},
		{"uint32", -1, 0}, {"uint32", 5000000000, 4294967295},
		{"uint8", 300, 255}, {"int8", -129, -128},
	} {
		reg := config.Register{ID: "out", Enabled: true, Type: "holding", NumericDirection: "output", DataType: tc.typ, Factor: 1}
		ch := make(chan writeRequest, 1)
		e := New()
		e.writers["test"] = ch
		e.active["test"] = config.Device{ID: "test", Registers: []config.Register{reg}}
		if e.EnqueueNumeric("test", "out", tc.bad) {
			t.Errorf("%s accepted out-of-range %g", tc.typ, tc.bad)
		}
		if len(ch) != 0 {
			t.Errorf("%s queued out-of-range write", tc.typ)
			<-ch
		}
		if !e.EnqueueNumeric("test", "out", tc.limit) {
			t.Errorf("%s rejected limit %g", tc.typ, tc.limit)
			continue
		}
		if req := <-ch; req.numeric == nil || *req.numeric != tc.limit {
			t.Errorf("%s sent %v want %g", tc.typ, req.numeric, tc.limit)
		}
		if e.EnqueueNumeric("test", "out", 1.5) {
			t.Errorf("%s accepted fractional raw", tc.typ)
		}
		if len(ch) != 0 {
			t.Errorf("%s queued invalid write", tc.typ)
		}
	}
}

// Decimal factors are inexact in binary floating point (0.3/0.1 = 2.9999999999999996).
// Such values must still be accepted and written as the exact integer raw value.
func TestEncodeNumericWriteDecimalFactor(t *testing.T) {
	for _, tc := range []struct {
		desired float64
		want    uint16
	}{{0.3, 3}, {0.7, 7}, {47.5, 475}, {-4.5, 0xffd3}, {3276.7, 32767}} {
		reg := config.Register{Type: "holding", NumericDirection: "output", DataType: "int16", Factor: 0.1}
		got, err := encodeNumericWrite(reg, tc.desired)
		if err != nil || len(got) != 1 || got[0] != tc.want {
			t.Errorf("%g: got %v %v want %#04x", tc.desired, got, err, tc.want)
		}
	}
	reg := config.Register{Type: "holding", NumericDirection: "output", DataType: "int16", Factor: 0.1}
	for _, bad := range []float64{45.25, 3276.8, -3276.9} {
		if _, err := encodeNumericWrite(reg, bad); err == nil {
			t.Errorf("%g accepted", bad)
		}
	}
}
