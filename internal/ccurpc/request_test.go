// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/request_test.go
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

func TestParseCallWithNestedParams(t *testing.T) {
	source, e := MarshalCall("newDevices", String("CCU-Modbus"), Array(Struct(map[string]Value{"ADDRESS": String("MB01:0"), "INDEX": Integer(0)})))
	if e != nil {
		t.Fatal(e)
	}
	name, args, e := ParseCall(bytes.NewReader(source))
	if e != nil {
		t.Fatal(e)
	}
	if name != "newDevices" || len(args) != 2 || args[1].Kind != "array" || args[1].Array[0].Members["ADDRESS"].Text != "MB01:0" {
		t.Fatalf("bad request: %s %+v", name, args)
	}
}
func TestResponseRoundTrip(t *testing.T) {
	response, e := MarshalResponse(Array(Integer(42), Boolean(false)))
	if e != nil {
		t.Fatal(e)
	}
	got, e := ParseResponse(bytes.NewReader(response))
	if e != nil || len(got.Array) != 2 || got.Array[0].Text != "42" {
		t.Fatalf("got %+v error %v", got, e)
	}
}
func TestRequestValidation(t *testing.T) {
	cases := []string{`<wrong><methodName>init</methodName></wrong>`, `<methodCall><methodName></methodName></methodCall>`, `<methodCall><methodName>init</methodName><params><param><value><int>xx</int></value></param></params></methodCall>`}
	for _, input := range cases {
		if _, _, e := ParseCall(strings.NewReader(input)); e == nil {
			t.Fatalf("accepted invalid request %q", input)
		}
	}
	if _, _, e := ParseCall(strings.NewReader(strings.Repeat("X", (1<<20)+1))); e == nil {
		t.Fatal("accepted oversized request")
	}
}
