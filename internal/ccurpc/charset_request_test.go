// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/charset_request_test.go
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

func TestParseCallReGaLatin1(t *testing.T) {
	wire := "<?xml version=\"1.0\" encoding=\"iso-8859-1\"?><methodCall><methodName>getDeviceDescription</methodName><params><param><value><string>" + string([]byte{0xfc}) + "</string></value></param></params></methodCall>"
	method, args, err := ParseCall(strings.NewReader(wire))
	if err != nil {
		t.Fatal(err)
	}
	if method != "getDeviceDescription" || len(args) != 1 || args[0].Text != "ü" {
		t.Fatalf("decoded %q %+v", method, args)
	}
}
