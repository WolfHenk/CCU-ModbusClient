// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/nested_test.go
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
	"testing"
)

func TestNestedBinaryValues(t *testing.T) {
	nested := Parameter{Type: TypeStruct, Fields: map[string]Parameter{"ADDRESS": {Type: TypeString, Text: "MÜ:1"}, "VALUES": {Type: TypeArray, Items: []Parameter{{Type: TypeInteger, Number: -42}, {Type: TypeBoolean, Flag: true}}}}}
	var wire bytes.Buffer
	if err := writeParam(&wire, nested, 0); err != nil {
		t.Fatal(err)
	}
	var msg bytes.Buffer
	if err := WriteFrame(&msg, 1, wire.Bytes()); err != nil {
		t.Fatal(err)
	}
	got, err := DecodeResponse(&msg)
	if err != nil {
		t.Fatal(err)
	}
	if got.Fields["ADDRESS"].Text != "MÜ:1" || got.Fields["VALUES"].Items[0].Number != -42 || !got.Fields["VALUES"].Items[1].Flag {
		t.Fatalf("incorrect nested values: %+v", got)
	}
}
func TestNestedBinaryRequest(t *testing.T) {
	var wire bytes.Buffer
	if err := EncodeRequest(&wire, "newDevices", Parameter{Type: TypeArray, Items: []Parameter{{Type: TypeStruct, Fields: map[string]Parameter{"INDEX": {Type: TypeInteger, Number: 0}}}}}); err != nil {
		t.Fatal(err)
	}
	f, err := ReadFrame(&wire)
	if err != nil || f.Kind != 0 || len(f.Payload) < 12 {
		t.Fatalf("invalid request: %+v %v", f, err)
	}
}
