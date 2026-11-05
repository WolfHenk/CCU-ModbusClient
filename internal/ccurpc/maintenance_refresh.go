// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/maintenance_refresh.go
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

// UpdateMaintenanceDescription uses the documented ReGa updateDevice call.
// It does not change, delete, or recreate an existing CCU object.
func UpdateMaintenanceDescription(ctx context.Context, rega Client, interfaceID, channelAddress string) error {
	if ctx == nil || strings.TrimSpace(interfaceID) == "" || strings.TrimSpace(channelAddress) == "" || !strings.HasSuffix(channelAddress, ":0") {
		return errors.New("invalid maintenance channel target")
	}
	result, err := rega.Call(ctx, "updateDevice", String(interfaceID), String(channelAddress), Integer(0))
	if err != nil {
		return fmt.Errorf("ReGa updateDevice(%s): %w", channelAddress, err)
	}
	if result.Kind != "string" || result.Text != "" {
		return fmt.Errorf("unexpected ReGa updateDevice response: %+v", result)
	}
	return nil
}
