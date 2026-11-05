// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/port_selection_test.go
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
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestBindAvailableSkipsOccupiedPort(t *testing.T) {
	occupied, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port
	if port >= 65535 {
		t.Skip("no next port")
	}
	listener, url, err := BindAvailable("127.0.0.1", port, port+1)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if listener.Addr().(*net.TCPAddr).Port != port+1 {
		t.Fatalf("did not skip occupied %d: %s", port, url)
	}
	if !strings.HasSuffix(url, ":"+strconv.Itoa(port+1)) {
		t.Fatalf("unexpected callback %q", url)
	}
	if _, err := net.Dial("tcp", occupied.Addr().String()); err != nil {
		t.Fatalf("existing server disturbed: %v", err)
	}
}
func TestBindAvailableRejectsInvalidRanges(t *testing.T) {
	if _, _, e := BindAvailable("0.0.0.0", 32000, 32002); e == nil {
		t.Fatal("non-loopback bound")
	}
	if _, _, e := BindAvailable("127.0.0.1", 32003, 32000); e == nil {
		t.Fatal("invalid range accepted")
	}
}
