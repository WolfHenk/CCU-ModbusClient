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

func TestEnqueueNumericClampAndReject(t *testing.T) {
	for _, tc := range []struct {
		typ         string
		input, want float64
	}{
		{"int16", -40000, -32768}, {"int16", 40000, 32767},
		{"uint16", -1, 0}, {"uint16", 70000, 65535},
		{"int32", -3000000000, -2147483648}, {"int32", 3000000000, 2147483647},
		{"uint32", -1, 0}, {"uint32", 5000000000, 4294967295},
	} {
		reg := config.Register{ID: "out", Enabled: true, Type: "holding", NumericDirection: "output", DataType: tc.typ, Factor: 1}
		ch := make(chan writeRequest, 1)
		e := New()
		e.writers["test"] = ch
		e.active["test"] = config.Device{ID: "test", Registers: []config.Register{reg}}
		if !e.EnqueueNumeric("test", "out", tc.input) {
			t.Errorf("%s failed enqueue %g", tc.typ, tc.input)
			continue
		}
		req := <-ch
		if req.numeric == nil || *req.numeric != tc.want {
			t.Errorf("%s sent %v want %g", tc.typ, req.numeric, tc.want)
		}
		if e.EnqueueNumeric("test", "out", 1.5) {
			t.Errorf("%s accepted fractional raw", tc.typ)
		}
		if len(ch) != 0 {
			t.Errorf("%s queued invalid write", tc.typ)
		}
	}
}
