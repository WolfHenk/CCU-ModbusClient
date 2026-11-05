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

// The unit must arrive at ReGa as one degree sign, never mojibake "Â°C".
func TestCCUUnitDegreeUsesEncodingIndependentXMLReference(t *testing.T) {
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
	if bytes.Contains(wire, []byte("Â°C")) || bytes.Contains(wire, []byte("°C")) {
		t.Fatalf("wire must not carry raw UTF-8/Latin1 degree bytes: %x", wire)
	}
	if !bytes.Contains(wire, []byte("&#xB0;C")) {
		t.Fatalf("degree unit is not safely XML-escaped: %s", wire)
	}
	got, err := ParseResponse(bytes.NewReader(wire))
	if err != nil {
		t.Fatal(err)
	}
	if unit := got.Members["VOLTAGE"].Members["UNIT"].Text; unit != "°C" {
		t.Fatalf("decoded unit = %q, expected °C", unit)
	}
}

func TestUnicodeUnitInCCUAnnouncementAndXMLMarkup(t *testing.T) {
	for _, s := range []string{"°C", "µg/m³", "ä & < >"} {
		wire, err := MarshalCall("newDevices", String(s))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(wire, []byte(s)) {
			t.Fatalf("wire contains raw UTF-8: %q", s)
		}
		if !bytes.Contains(wire, []byte("&#x")) {
			t.Fatalf("no numeric XML entities: %q", s)
		}
		if strings.Contains(s, "&") && !bytes.Contains(wire, []byte("&amp;")) {
			t.Fatal("XML escaping broken")
		}
	}
}
