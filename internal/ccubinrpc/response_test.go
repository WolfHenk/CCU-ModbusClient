// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/response_test.go
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

package ccubinrpc

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestDoubleRoundTripResponse(t *testing.T) {
	for _, num := range []float64{0, 1, -1, 125.125, -9999.015625, 0.0000152587890625} {
		var request bytes.Buffer
		if err := EncodeRequest(&request, "setValue", Parameter{Type: TypeDouble, Float: num}); err != nil {
			t.Fatal(err)
		}
		frame, err := ReadFrame(&request)
		if err != nil {
			t.Fatal(err)
		}
		// The encoded request ends in the scalar double type, mantissa and exponent.
		suffix := frame.Payload[len(frame.Payload)-12:]
		var res bytes.Buffer
		if err := WriteFrame(&res, 1, suffix); err != nil {
			t.Fatal(err)
		}
		got, err := DecodeResponse(&res)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got.Float-num) > math.Max(1e-8, math.Abs(num)*1e-8) {
			t.Fatalf("%v -> %v", num, got.Float)
		}
	}
}
func TestResponseRejectsInvalidFrames(t *testing.T) {
	var b bytes.Buffer
	bad := make([]byte, 8)
	binary.BigEndian.PutUint32(bad[:4], TypeBoolean)
	binary.BigEndian.PutUint32(bad[4:], 2)
	_ = WriteFrame(&b, 1, bad)
	if _, err := DecodeResponse(&b); err == nil {
		t.Fatal("accepted invalid boolean")
	}
	for _, v := range []float64{math.NaN(), math.Inf(1)} {
		if err := EncodeRequest(&b, "event", Parameter{Type: TypeDouble, Float: v}); err == nil {
			t.Fatal("accepted nonfinite double")
		}
	}
}
