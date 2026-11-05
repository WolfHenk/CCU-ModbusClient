// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_identity.go
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

import "errors"

const (
	LegacyInterfaceID = "CCU-Modbus"
	ClientInterfaceID = "CCU-ModbusClient"
)

// NewClientRegistration is only for the new independently named interface.
// It never registers using the legacy identity.
func NewClientRegistration(client Client) Registration {
	return Registration{Client: client, InterfaceID: ClientInterfaceID}
}

// ValidateClientIdentity prevents accidental takeover of the legacy registration.
func (r Registration) ValidateClientIdentity() error {
	if r.InterfaceID != ClientInterfaceID {
		return errors.New("new RPC runtime requires CCU-ModbusClient interface identity")
	}
	return nil
}
