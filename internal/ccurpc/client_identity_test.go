// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_identity_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccubinrpc"
	"testing"
)

func TestNewClientIdentityRejectsLegacyAndKeepsDistinctCallbacks(t *testing.T) {
	server := ccubinrpc.NewTCPServer()
	legacy := NewClientRegistration(Client{})
	legacy.InterfaceID = LegacyInterfaceID
	if _, err := ReserveCallback("127.0.0.1", 31999, 32008, server, legacy); err == nil {
		t.Fatal("legacy identity accepted")
	}
	first, err := ReserveCallback("127.0.0.1", 31999, 32008, server, NewClientRegistration(Client{}))
	if err != nil {
		t.Fatal(err)
	}
	defer first.Listener.Close()
	second, err := ReserveCallback("127.0.0.1", 31999, 32008, server, NewClientRegistration(Client{}))
	if err != nil {
		t.Fatal(err)
	}
	defer second.Listener.Close()
	if first.Registration.CallbackURL == second.Registration.CallbackURL {
		t.Fatal("callback collision")
	}
	if first.Registration.InterfaceID != ClientInterfaceID {
		t.Fatal("wrong interface ID")
	}
}
