// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/request_test.go
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
	"encoding/hex"
	"testing"
)

func TestBinRPCInitReferenceFrame(t *testing.T) {
	var b bytes.Buffer
	err := EncodeRequest(&b, "init", Parameter{Type: TypeString, Text: "xmlrpc_bin://172.16.23.180:2004"}, Parameter{Type: TypeString, Text: "test"})
	if err != nil {
		t.Fatal(err)
	}
	expected := "42696e000000003f00000004696e697400000002000000030000001f786d6c7270635f62696e3a2f2f3137322e31362e32332e3138303a32303034000000030000000474657374"
	if hex.EncodeToString(b.Bytes()) != expected {
		t.Fatalf("unexpected wire frame: %x", b.Bytes())
	}
}
func TestBinRPCFaultFrameAndEncodingErrors(t *testing.T) {
	var b bytes.Buffer
	if err := WriteFrame(&b, 0xff, nil); err != nil {
		t.Fatal(err)
	}
	frame, err := ReadFrame(&b)
	if err != nil || frame.Kind != 0xff {
		t.Fatalf("fault frame %v %v", frame, err)
	}
	if err := EncodeRequest(&b, "name", Parameter{Type: TypeString, Text: "€"}); err == nil {
		t.Fatal("accepted non-Latin1 string")
	}
	if err := EncodeRequest(&b, "name", Parameter{Type: 900}); err == nil {
		t.Fatal("accepted unknown type")
	}
}
