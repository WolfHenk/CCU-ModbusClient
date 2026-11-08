// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/unit_encoding_test.go
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

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
)

// The unit must arrive at ReGa as one ISO-8859-1 degree byte (0xB0).
// ReGa stores XML character references such as "&#xB0;" literally, and raw
// UTF-8 (0xC2 0xB0) is shown as mojibake "Â°C".
func TestCCUUnitDegreeIsLatin1Byte(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{ID: "d", Enabled: true, CCUAddress: "MBC020B01F601", Registers: []config.Register{{
		ID: "temperature", Name: "Temperatur", Enabled: true, Type: "input", DataType: "int16", Unit: "°C",
	}}}}}
	cat, err := CatalogFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	desc, err := cat.ParamsetDescription("MBC020B01F601:1", "VALUES")
	if err != nil {
		t.Fatal(err)
	}
	wire, err := MarshalResponse(desc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(wire, []byte(`<?xml version="1.0" encoding="ISO-8859-1"?>`)) {
		t.Fatalf("missing ISO-8859-1 declaration: %.60s", wire)
	}
	if bytes.Contains(wire, []byte("\xc2\xb0")) || bytes.Contains(wire, []byte("&#x")) {
		t.Fatalf("degree sign must be one Latin-1 byte: %q", wire)
	}
	if !bytes.Contains(wire, []byte("\xb0C")) {
		t.Fatalf("degree unit missing as 0xB0: %q", wire)
	}
	got, err := ParseResponse(bytes.NewReader(wire))
	if err != nil {
		t.Fatal(err)
	}
	if unit := got.Members["VALUE"].Members["UNIT"].Text; unit != "°C" {
		t.Fatalf("decoded unit = %q, expected °C", unit)
	}
}

func TestLatin1TextInCCUAnnouncementAndXMLMarkup(t *testing.T) {
	for _, c := range []struct {
		in   string
		wire []byte
	}{
		{"°C", []byte("\xb0C")},
		{"µg/m³", []byte("\xb5g/m\xb3")},
		{"ä & < >", []byte("\xe4 &amp; &lt; &gt;")},
		{"5 €", []byte("5 ?")}, // outside Latin-1
	} {
		wire, err := MarshalCall("newDevices", String(c.in))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(wire, c.wire) {
			t.Fatalf("%q: wire %q does not contain %q", c.in, wire, c.wire)
		}
		if strings.Contains(string(wire), "&#x") {
			t.Fatalf("%q: numeric character reference on wire", c.in)
		}
	}
}
