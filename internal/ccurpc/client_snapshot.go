// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client_snapshot.go
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
	"context"
	"fmt"
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
	"time"
)

// ClientSnapshotConfig creates an ephemeral MBC-addressed configuration used
// only for RPC snapshots. Persistent legacy addresses are never rewritten.
func ClientSnapshotConfig(cfg *config.Config) (*config.Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("missing configuration")
	}
	cp := *cfg
	cp.Devices = append([]config.Device(nil), cfg.Devices...)
	for i := range cp.Devices {
		d := &cp.Devices[i]
		if !d.Enabled {
			continue
		}
		identity, err := ClientSerial(d.Host, d.Port, d.UnitID)
		if err != nil {
			return nil, err
		}
		if d.CCUAddress != "" && !IsClientSerialForEndpoint(d.CCUAddress, identity) {
			return nil, fmt.Errorf("device %s requires explicit identity migration", d.ID)
		}
		d.CCUAddress = normalizedClientSerial(d.CCUAddress, identity)
	}
	return &cp, nil
}

// SnapshotSource is implemented by the existing Modbus engine.
type SnapshotSource interface{ Snapshot() []model.DeviceState }

// PollClientSnapshots updates cached RPC values using the existing engine.
// No Modbus write, RPC registration or CCU system modification occurs here.
func PollClientSnapshots(ctx context.Context, cat *MemoryCatalog, cfg *config.Config, src SnapshotSource, interval time.Duration) error {
	if ctx == nil || cat == nil || src == nil || interval <= 0 {
		return fmt.Errorf("invalid snapshot poller")
	}
	mapped, err := ClientSnapshotConfig(cfg)
	if err != nil {
		return err
	}
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := cat.ApplySnapshot(mapped, src.Snapshot()); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
	}
}
