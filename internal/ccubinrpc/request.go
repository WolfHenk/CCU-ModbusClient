// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/request.go
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

package ccubinrpc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
)

// Parameter represents a simple CCU BinRPC parameter. Complex parameters
// are deliberately unsupported until their independent tests are complete.
type Parameter struct {
	Type   uint32
	Text   string
	Number int32
	Flag   bool
	Float  float64
	Items  []Parameter
	Fields map[string]Parameter
}

const (
	TypeInteger uint32 = 1
	TypeBoolean uint32 = 2
	TypeString  uint32 = 3
	TypeArray   uint32 = 0x100
	TypeStruct  uint32 = 0x101
)

// EncodeRequest builds a BinRPC method request with strict size and
// ISO-8859-1 text validation. The caller supplies an external RPC method.
func EncodeRequest(w io.Writer, method string, args ...Parameter) error {
	var p bytes.Buffer
	if err := writeLatin1(&p, method); err != nil {
		return err
	}
	if len(args) > 1024 {
		return errors.New("too many RPC parameters")
	}
	_ = binary.Write(&p, binary.BigEndian, uint32(len(args)))
	for _, a := range args {
		if err := writeParam(&p, a, 0); err != nil {
			return err
		}
	}
	return WriteFrame(w, 0, p.Bytes())
}
func writeLatin1(w io.Writer, s string) error {
	if len(s) > MaxPayload {
		return errors.New("BinRPC string too long")
	}
	data := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 255 {
			return fmt.Errorf("character U+%04X cannot be represented as Latin-1", r)
		}
		data = append(data, byte(r))
	}
	if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	return writeFull(w, data)
}

func writeParam(p *bytes.Buffer, a Parameter, depth int) error {
	if depth > 32 {
		return errors.New("BinRPC nesting too deep")
	}
	if err := binary.Write(p, binary.BigEndian, a.Type); err != nil {
		return err
	}
	switch a.Type {
	case TypeInteger:
		return binary.Write(p, binary.BigEndian, a.Number)
	case TypeBoolean:
		if a.Flag {
			return binary.Write(p, binary.BigEndian, uint32(1))
		}
		return binary.Write(p, binary.BigEndian, uint32(0))
	case TypeString:
		return writeLatin1(p, a.Text)
	case TypeDouble:
		if math.IsNaN(a.Float) || math.IsInf(a.Float, 0) {
			return errors.New("nonfinite double")
		}
		m, e := math.Frexp(a.Float)
		v := math.Floor(m * float64(uint64(1)<<30))
		if v < math.MinInt32 || v > math.MaxInt32 {
			return errors.New("double mantissa overflow")
		}
		if err := binary.Write(p, binary.BigEndian, int32(v)); err != nil {
			return err
		}
		return binary.Write(p, binary.BigEndian, int32(e))
	case TypeArray:
		if len(a.Items) > 1024 {
			return errors.New("too many array elements")
		}
		_ = binary.Write(p, binary.BigEndian, uint32(len(a.Items)))
		for _, item := range a.Items {
			if err := writeParam(p, item, depth+1); err != nil {
				return err
			}
		}
	case TypeStruct:
		if len(a.Fields) > 1024 {
			return errors.New("too many struct members")
		}
		_ = binary.Write(p, binary.BigEndian, uint32(len(a.Fields)))
		keys := make([]string, 0, len(a.Fields))
		for key := range a.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if err := writeLatin1(p, key); err != nil {
				return err
			}
			if err := writeParam(p, a.Fields[key], depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported BinRPC parameter type %d", a.Type)
	}
	return nil
}
