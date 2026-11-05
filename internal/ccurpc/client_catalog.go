// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_catalog.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"strings"
)

// CatalogFromClientConfig derives new MBC identities but never changes a
// persisted legacy identity. The supplied configuration remains unmodified.
func CatalogFromClientConfig(cfg *config.Config) (*MemoryCatalog, error) {
	if cfg == nil {
		return nil, fmt.Errorf("missing configuration")
	}
	copyConfig := *cfg
	copyConfig.Devices = append([]config.Device(nil), cfg.Devices...)
	seen := make(map[string]string)
	for i := range copyConfig.Devices {
		d := &copyConfig.Devices[i]
		if !d.Enabled {
			continue
		}
		expected, err := ClientSerial(d.Host, d.Port, d.UnitID)
		if err != nil {
			return nil, fmt.Errorf("device %s: %w", d.ID, err)
		}
		actual := strings.TrimSpace(d.CCUAddress)
		if actual != "" && !IsClientSerialForEndpoint(actual, expected) {
			return nil, fmt.Errorf("device %s: foreign CCU identity %q: CCU-ModbusClient uses only its own serial %q", d.ID, actual, expected)
		}
		serial := normalizedClientSerial(actual, expected)
		if prior, exists := seen[serial]; exists {
			return nil, fmt.Errorf("MBC identity collision %s between devices %s and %s", expected, prior, d.ID)
		}
		seen[serial] = d.ID
		d.CCUAddress = serial
	}
	return CatalogFromConfig(&copyConfig)
}
