// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/rega_announce.go
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
	"errors"
	"fmt"
	"strings"
)

// AnnounceDevices informs the ReGa logic layer of the complete device/channel
// descriptions of this interface. It never removes existing devices.
// The interface identifier must already exist in ReGa.
func AnnounceDevices(ctx context.Context, rega Client, interfaceID string, catalog DeviceCatalog) error {
	if ctx == nil || catalog == nil || strings.TrimSpace(interfaceID) == "" {
		return errors.New("missing ReGa announcement prerequisites")
	}
	entries, err := catalog.ListDevices()
	if err != nil {
		return fmt.Errorf("list devices: %w", err)
	}
	if len(entries) == 0 {
		return nil // Empty configuration is a valid initial state.
	}
	return AnnounceDescriptions(ctx, rega, interfaceID, entries)
}

// AnnounceDescriptions preserves the supplied descriptions and their order.
// A migration may announce exactly one existing channel, not a whole device.
func AnnounceDescriptions(ctx context.Context, rega Client, interfaceID string, entries []Value) error {
	if ctx == nil || strings.TrimSpace(interfaceID) == "" || len(entries) == 0 {
		return errors.New("invalid ReGa announcement")
	}
	for i, v := range entries {
		if v.Kind != "struct" || v.Members["ADDRESS"].Kind != "string" {
			return fmt.Errorf("invalid device description at index %d", i)
		}
	}
	reply, err := rega.Call(ctx, "newDevices", String(interfaceID), Array(entries...))
	if err != nil {
		return fmt.Errorf("ReGa newDevices: %w", err)
	}
	if reply.Kind != "string" || reply.Text != "" {
		return fmt.Errorf("unexpected ReGa newDevices response: %+v", reply)
	}
	return nil
}
