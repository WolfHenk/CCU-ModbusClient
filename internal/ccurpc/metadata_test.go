// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/metadata_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"testing"
)

func TestServiceFlagsAndLimits(t *testing.T) {
	c, err := CatalogFromConfig(&config.Config{Devices: []config.Device{{ID: "service", Enabled: true, CCUAddress: "MB001002502001"}}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.ParamsetDescription("MB001002502001:0", "VALUES")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"MODBUS_ERROR", "MODBUS_EXCEPTION"} {
		meta := p.Members[key].Members
		if meta["FLAGS"].Text != "9" || meta["OPERATIONS"].Text != "5" {
			t.Fatalf("%s metadata: %+v", key, meta)
		}
	}
	meta := p.Members["MODBUS_ERROR"].Members
	if meta["MIN"].Text != "0" || meta["MAX"].Text != "65291" {
		t.Fatalf("limits: %+v", meta)
	}
}
