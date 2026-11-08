// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/report_value_usage_test.go
// Ersteller:    Wolfram Henkel
// Datum:        08. Nov. 2026
// Version:      0.3.2
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
	"context"
	"net/http/httptest"
	"testing"
)

// ReGa calls reportValueUsage when it removes datapoints, also for datapoints
// that the catalog no longer has (old VALUE after a change to LEVEL).
func TestReportValueUsageAcknowledged(t *testing.T) {
	h := NewHandler()
	if err := RegisterCCUReadMethods(h, fakeCatalog{}); err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(h)
	defer s.Close()
	c := Client{URL: s.URL}
	got, err := c.Call(context.Background(), "reportValueUsage", String("MBC075905DE02:24"), String("VALUE"), Integer(0))
	if err != nil {
		t.Fatalf("reportValueUsage: %v", err)
	}
	if got.Kind != Boolean(true).Kind || got.Text != Boolean(true).Text {
		t.Fatalf("reportValueUsage result %+v", got)
	}
	if _, err := c.Call(context.Background(), "reportValueUsage", String("MBC075905DE02:24")); err == nil {
		t.Fatal("accepted missing arguments")
	}
}
