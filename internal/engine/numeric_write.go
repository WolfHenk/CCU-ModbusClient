// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/numeric_write.go
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
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
)

// encodeNumericWrite reverses the register's engineering conversion.
// Reject values which cannot be represented instead of silently truncating.
func encodeNumericWrite(r config.Register, desired float64) ([]uint16, error) {
	if math.IsNaN(desired) || math.IsInf(desired, 0) || r.Factor == 0 {
		return nil, fmt.Errorf("ungueltiger numerischer Sollwert oder Faktor")
	}
	raw := (desired - r.Offset) / r.Factor
	if math.IsNaN(raw) || math.IsInf(raw, 0) {
		return nil, fmt.Errorf("ungueltiger Rohwert")
	}
	typ := strings.ToLower(r.DataType)
	size := 2
	if typ == "uint32" || typ == "int32" || typ == "float32" {
		size = 4
	}
	buf := make([]byte, size)
	integer := func(min, max float64) (float64, error) {
		// Factors such as 0.1 are not exact in binary floating point
		// (0.3/0.1 = 2.9999999999999996). Accept a result that is an integer
		// within a relative tolerance, then range-check the rounded value.
		rounded := math.Round(raw)
		if math.Abs(raw-rounded) > 1e-9*math.Max(1, math.Abs(raw)) {
			return 0, fmt.Errorf("Sollwert %v ist mit Faktor %v nicht als ganzzahliger %s-Rohwert darstellbar", desired, r.Factor, typ)
		}
		if rounded < min || rounded > max {
			return 0, fmt.Errorf("Sollwert %v ausserhalb des %s-Bereichs", desired, typ)
		}
		return rounded, nil
	}
	switch typ {
	case "uint8":
		x, e := integer(0, 255)
		if e != nil {
			return nil, e
		}
		buf[1] = byte(x)
	case "int8":
		x, e := integer(-128, 127)
		if e != nil {
			return nil, e
		}
		buf[1] = byte(int8(x))
	case "uint16":
		x, e := integer(0, 65535)
		if e != nil {
			return nil, e
		}
		binary.BigEndian.PutUint16(buf, uint16(x))
	case "int16":
		x, e := integer(-32768, 32767)
		if e != nil {
			return nil, e
		}
		binary.BigEndian.PutUint16(buf, uint16(int16(x)))
	case "uint32":
		x, e := integer(0, 4294967295)
		if e != nil {
			return nil, e
		}
		binary.BigEndian.PutUint32(buf, uint32(x))
	case "int32":
		x, e := integer(-2147483648, 2147483647)
		if e != nil {
			return nil, e
		}
		binary.BigEndian.PutUint32(buf, uint32(int32(x)))
	case "float32":
		if math.Abs(raw) > math.MaxFloat32 {
			return nil, fmt.Errorf("float32 ausserhalb Wertebereich")
		}
		binary.BigEndian.PutUint32(buf, math.Float32bits(float32(raw)))
	default:
		return nil, fmt.Errorf("numerischer Schreibtyp %s nicht unterstuetzt", typ)
	}
	// Swap operations are self-inverse; encode applies them in reverse decoding order.
	// For 2/4-byte payloads DWordSwap is a no-op.
	if r.WordSwap {
		for i := 0; i+3 < len(buf); i += 4 {
			buf[i], buf[i+2] = buf[i+2], buf[i]
			buf[i+1], buf[i+3] = buf[i+3], buf[i+1]
		}
	}
	if r.ByteSwap {
		for i := 0; i+1 < len(buf); i += 2 {
			buf[i], buf[i+1] = buf[i+1], buf[i]
		}
	}
	values := make([]uint16, len(buf)/2)
	for i := range values {
		values[i] = binary.BigEndian.Uint16(buf[i*2:])
	}
	return values, nil
}
