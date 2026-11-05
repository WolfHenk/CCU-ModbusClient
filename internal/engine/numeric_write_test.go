// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/numeric_write_test.go
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
	"math"
	"testing"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
)

func TestEncodeNumericWriteRoundTrip(t *testing.T) {
	cases := []struct {
		typ            string
		value          float64
		factor, offset float64
		swap, byteSwap bool
	}{
		{"int16", -123, 1, 0, false, false},
		{"uint16", 42, 1, 0, false, false},
		{"uint8", 201, 1, 0, false, false},
		{"int8", -22, 1, 0, false, false},
		{"uint32", 1234567, 1, 0, true, true},
		{"int32", -1234567, 1, 0, true, true},
		{"float32", 12.5, 1, 0, true, false},
		{"int16", 42, 0.5, 10, false, true},
	}
	for _, tc := range cases {
		r := config.Register{ID: "r", Name: "Test", Enabled: true, Type: "holding", NumericDirection: "output", DataType: tc.typ, Factor: tc.factor, Offset: tc.offset, WordSwap: tc.swap, ByteSwap: tc.byteSwap, PollSeconds: 10}
		words, err := encodeNumericWrite(r, tc.value)
		if err != nil {
			t.Errorf("%s encode: %v", tc.typ, err)
			continue
		}
		b := make([]byte, 0, len(words)*2)
		for _, w := range words {
			b = append(b, byte(w>>8), byte(w))
		}
		_, decoded, err := decode(r, b)
		if err != nil {
			t.Errorf("%s decode: %v", tc.typ, err)
			continue
		}
		f, ok := decoded.(float64)
		if !ok || math.Abs(f-tc.value) > 0.00001 {
			t.Errorf("%s round trip %v != %v", tc.typ, decoded, tc.value)
		}
	}
}
func TestEncodeNumericWriteRejectsUnsafeValues(t *testing.T) {
	r := config.Register{Type: "holding", NumericDirection: "output", DataType: "uint16", Factor: 1}
	for _, v := range []float64{-1, 65536, 1.5, math.NaN(), math.Inf(1)} {
		if _, err := encodeNumericWrite(r, v); err == nil {
			t.Errorf("accepted invalid %v", v)
		}
	}
	r.Factor = 0
	if _, err := encodeNumericWrite(r, 10); err == nil {
		t.Fatal("zero factor accepted")
	}
}
