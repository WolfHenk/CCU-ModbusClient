// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_serial_allocate.go
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
	"sort"
	"strings"
)

const collisionAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// AllocateClientSerials returns a copy with deterministic, stable addresses.
// Existing valid identities are reserved before assigning any new identity.
// Caller must persist the result before using it to register CCU devices.
func AllocateClientSerials(cfg *config.Config) (*config.Config, bool, error) {
	if cfg == nil {
		return nil, false, fmt.Errorf("missing configuration")
	}
	cp := *cfg
	cp.Devices = append([]config.Device(nil), cfg.Devices...)
	used := map[string]string{}
	bases := make([]string, len(cp.Devices))
	pending := make([]int, 0)
	for i := range cp.Devices {
		d := &cp.Devices[i]
		if !d.Enabled {
			continue
		}
		base, err := ClientSerial(d.Host, d.Port, d.UnitID)
		if err != nil {
			return nil, false, fmt.Errorf("device %s: %w", d.ID, err)
		}
		bases[i] = base
		actual := strings.TrimSpace(d.CCUAddress)
		if actual == "" {
			pending = append(pending, i)
			continue
		}
		if !IsClientSerialForEndpoint(actual, base) {
			return nil, false, fmt.Errorf("device %s: foreign or incompatible CCU identity %q: CCU-ModbusClient never adopts old addon serials", d.ID, actual)
		}
		if prior, ok := used[actual]; ok {
			return nil, false, fmt.Errorf("serial %s used by %s and %s", actual, prior, d.ID)
		}
		used[actual] = d.ID
	}
	sort.Slice(pending, func(a, b int) bool {
		x, y := pending[a], pending[b]
		if bases[x] != bases[y] {
			return bases[x] < bases[y]
		}
		if cp.Devices[x].ID != cp.Devices[y].ID {
			return cp.Devices[x].ID < cp.Devices[y].ID
		}
		return x < y
	})
	changed := false
	for _, i := range pending {
		d := &cp.Devices[i]
		base := bases[i]
		serial := base
		if _, ok := used[serial]; ok {
			serial = ""
			for _, c := range collisionAlphabet {
				candidate := "MBC" + string(c) + base[3:]
				if _, exists := used[candidate]; !exists {
					serial = candidate
					break
				}
			}
			if serial == "" {
				return nil, false, fmt.Errorf("all 62 collision variants exhausted for %s", base)
			}
		}
		d.CCUAddress = serial
		used[serial] = d.ID
		changed = true
	}
	return &cp, changed, nil
}
