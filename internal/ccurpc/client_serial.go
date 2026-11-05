// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_serial.go
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
	"fmt"
	"net"
)

// ClientSerial derives the new CCU-ModbusClient identity from the Modbus endpoint.
// MBC + penultimate IPv4 octet (2 hex) + final octet (2 hex)
// + TCP port (4 hex) + unit ID (2 hex), 13 ASCII characters in total.
func ClientSerial(host string, port, unit int) (string, error) {
	ip := net.ParseIP(host)
	if ip == nil || ip.To4() == nil {
		return "", fmt.Errorf("ModbusClient serial requires numeric IPv4 address: %q", host)
	}
	if port < 1 || port > 65535 || unit < 0 || unit > 255 {
		return "", fmt.Errorf("invalid Modbus port %d or unit %d", port, unit)
	}
	v4 := ip.To4()
	return fmt.Sprintf("MBC%02X%02X%04X%02X", v4[2], v4[3], port, unit), nil
}
