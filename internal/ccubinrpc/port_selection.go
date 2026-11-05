// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/port_selection.go
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
	"errors"
	"fmt"
	"net"
	"strconv"
)

// BindAvailable reserves the first free local port from an explicit range.
// The returned listener must be held open until Serve starts, avoiding TOCTOU.
// Only loopback listeners are supported; no occupied port is disturbed.
func BindAvailable(host string, first, last int) (net.Listener, string, error) {
	if host != "127.0.0.1" && host != "::1" {
		return nil, "", errors.New("callback listener must use loopback")
	}
	if first < 1024 || last > 65535 || last < first {
		return nil, "", errors.New("invalid callback port range")
	}
	for port := first; port <= last; port++ {
		addr := net.JoinHostPort(host, strconv.Itoa(port))
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			return listener, "xmlrpc_bin://" + addr, nil
		}
	}
	return nil, "", fmt.Errorf("no free callback port in %d-%d", first, last)
}
