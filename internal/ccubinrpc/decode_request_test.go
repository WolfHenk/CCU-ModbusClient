// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/decode_request_test.go
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
	"strings"
	"testing"
)

func TestRequestDecodeRoundTrip(t *testing.T) {
	var b bytes.Buffer
	err := EncodeRequest(&b, "init", Parameter{Type: TypeString, Text: "xmlrpc_bin://127.0.0.1:31999"}, Parameter{Type: TypeString, Text: "CCU-Modbus"})
	if err != nil {
		t.Fatal(err)
	}
	method, args, err := DecodeRequest(&b)
	if err != nil || method != "init" || len(args) != 2 || args[1].Text != "CCU-Modbus" {
		t.Fatalf("request %q %+v err=%v", method, args, err)
	}
}
func TestResultAndFault(t *testing.T) {
	var b bytes.Buffer
	if err := EncodeResponse(&b, Parameter{Type: TypeArray, Items: []Parameter{{Type: TypeInteger, Number: 42}}}); err != nil {
		t.Fatal(err)
	}
	got, err := DecodeResult(&b)
	if err != nil || got.Type != TypeArray || got.Items[0].Number != 42 {
		t.Fatalf("response %+v %v", got, err)
	}
	if err := EncodeFault(&b, 4, "not supported"); err != nil {
		t.Fatal(err)
	}
	_, err = DecodeResult(&b)
	if err == nil || !strings.Contains(err.Error(), "fault 4: not supported") {
		t.Fatalf("fault missing: %v", err)
	}
}
func TestRejectMalformedBinRPCRequest(t *testing.T) {
	var b bytes.Buffer
	_ = WriteFrame(&b, 0, []byte{0, 0, 0, 0})
	if _, _, err := DecodeRequest(&b); err == nil {
		t.Fatal("empty method accepted")
	}
	_ = EncodeResponse(&b, Parameter{Type: TypeString, Text: "x"})
	if _, _, err := DecodeRequest(&b); err == nil {
		t.Fatal("response accepted as request")
	}
}
