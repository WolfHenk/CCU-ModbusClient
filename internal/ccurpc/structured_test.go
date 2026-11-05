// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/structured_test.go
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
	"strings"
	"testing"
)

func TestNestedRoundTripShape(t *testing.T) {
	v := Struct(map[string]Value{"address": String("MB01:0"), "channels": Array(Integer(1), Boolean(true))})
	b, e := MarshalCall("newDevices", v)
	if e != nil {
		t.Fatal(e)
	}
	for _, frag := range []string{"<struct>", "<array><data>", "<name>channels</name>", "<boolean>1</boolean>"} {
		if !strings.Contains(string(b), frag) {
			t.Fatalf("missing %s in %s", frag, b)
		}
	}
	res := `<methodResponse><params><param><value><struct><member><name>channels</name><value><array><data><value><int>1</int></value><value><boolean>1</boolean></value></data></array></value></member></struct></value></param></params></methodResponse>`
	out, e := ParseResponse(strings.NewReader(res))
	if e != nil {
		t.Fatal(e)
	}
	if out.Kind != "struct" || len(out.Members["channels"].Array) != 2 || out.Members["channels"].Array[1].Text != "1" {
		t.Fatalf("wrong structure %+v", out)
	}
}
func TestRejectDuplicateStructMember(t *testing.T) {
	res := `<methodResponse><params><param><value><struct><member><name>a</name><value><int>1</int></value></member><member><name>a</name><value><int>2</int></value></member></struct></value></param></params></methodResponse>`
	if _, e := ParseResponse(strings.NewReader(res)); e == nil {
		t.Fatal("accepted duplicate member")
	}
}
