// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_serial_test.go
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

import "testing"

func TestClientSerial(t *testing.T) {
	for _, tc := range []struct {
		ip         string
		port, unit int
		want       string
	}{
		{"192.168.3.207", 502, 1, "MBC03CF01F601"},
		{"192.0.2.200", 502, 255, "MBC02C801F6FF"},
		{"10.0.0.0", 1, 0, "MBC0000000100"},
		{"10.0.255.255", 65535, 255, "MBCFFFFFFFFFF"},
	} {
		got, err := ClientSerial(tc.ip, tc.port, tc.unit)
		if err != nil || got != tc.want || len(got) != 13 {
			t.Errorf("%s:%d/%d: got %q err %v want %q", tc.ip, tc.port, tc.unit, got, err, tc.want)
		}
	}
}
func TestClientSerialRejectsInvalid(t *testing.T) {
	for _, tc := range []struct {
		ip         string
		port, unit int
	}{{"hostname", 502, 1}, {"::1", 502, 1}, {"192.168.1.2", 0, 1}, {"192.168.1.2", 65536, 1}, {"192.168.1.2", 502, -1}, {"192.168.1.2", 502, 256}} {
		if got, err := ClientSerial(tc.ip, tc.port, tc.unit); err == nil {
			t.Errorf("accepted %q: %s", tc.ip, got)
		}
	}
}
