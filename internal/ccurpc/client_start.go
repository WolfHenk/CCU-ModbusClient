// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_start.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccubinrpc"
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
)

// PrepareClient builds validated, isolated MBC identities before any socket is
// opened or registration attempted. The caller owns the resulting runtime.
func PrepareClient(cfg *config.Config, client Client, server *ccubinrpc.TCPServer, first, last int) (*CallbackRuntime, *MemoryCatalog, error) {
	catalog, err := CatalogFromClientConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("client identity validation: %w", err)
	}
	runtime, err := ReserveCallback("127.0.0.1", first, last, server, NewClientRegistration(client))
	if err != nil {
		return nil, nil, err
	}
	return runtime, catalog, nil
}
