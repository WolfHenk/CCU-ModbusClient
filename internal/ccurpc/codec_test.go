// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/codec_test.go
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
	"bytes"
	"strings"
	"testing"
)

func TestMarshalCallAndParseMethod(t *testing.T) {
	wire, err := MarshalCall("updateDevice", String("CCU-Modbus"), String("MB001:0"), Integer(0))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(wire, []byte("<int>0</int>")) {
		t.Fatalf("missing integer: %s", wire)
	}
	method, err := ParseMethodName(bytes.NewReader(wire))
	if err != nil || method != "updateDevice" {
		t.Fatalf("method=%q error=%v", method, err)
	}
}
func TestEscapingAndRejection(t *testing.T) {
	wire, err := MarshalCall("event", String("<&\"'"), Boolean(true))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(wire, []byte("&lt;&amp;")) {
		t.Fatalf("XML escaping missing: %s", wire)
	}
	for _, bad := range []string{"", "bad name", "bad\nname"} {
		if _, err := MarshalCall(bad); err == nil {
			t.Fatalf("accepted method %q", bad)
		}
	}
	if _, err := MarshalCall("event", Value{Kind: "object"}); err == nil {
		t.Fatal("accepted unsupported value")
	}
	if _, err := ParseMethodName(strings.NewReader("<wrong><methodName>x</methodName></wrong>")); err == nil {
		t.Fatal("accepted wrong root")
	}
}
