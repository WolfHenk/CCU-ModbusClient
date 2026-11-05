// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/numeric_clamp.go
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

package engine

import (
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"math"
	"strings"
)

// ClampNumericOutput limits an integer write to its register representation.
// Invalid fractional or non-finite requests are left unchanged, so the strict
// encoder will reject them. Factor/offset are considered in engineering units.
func ClampNumericOutput(r config.Register, desired float64) (float64, bool) {
	if math.IsNaN(desired) || math.IsInf(desired, 0) || r.Factor == 0 {
		return desired, false
	}
	var lo, hi float64
	switch strings.ToLower(r.DataType) {
	case "uint8":
		lo, hi = 0, 255
	case "int8":
		lo, hi = -128, 127
	case "uint16":
		lo, hi = 0, 65535
	case "int16":
		lo, hi = -32768, 32767
	case "uint32":
		lo, hi = 0, 4294967295
	case "int32":
		lo, hi = -2147483648, 2147483647
	default:
		return desired, false
	}
	raw := (desired - r.Offset) / r.Factor
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		return desired, false
	}
	if raw < lo {
		return lo*r.Factor + r.Offset, true
	}
	if raw > hi {
		return hi*r.Factor + r.Offset, true
	}
	return desired, false
}
