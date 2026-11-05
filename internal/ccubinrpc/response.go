// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/response.go
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
)

const TypeDouble uint32 = 4

// DecodeResponse reads one bounded BinRPC response frame containing a scalar.
// Nested values are deliberately unsupported at this stage.
func DecodeResponse(r io.Reader) (Parameter, error) {
	frame, err := ReadFrame(r)
	if err != nil {
		return Parameter{}, err
	}
	if frame.Kind != 1 {
		return Parameter{}, fmt.Errorf("expected BinRPC response, got type 0x%02x", frame.Kind)
	}
	data := bytes.NewReader(frame.Payload)
	p, err := readParam(data, 0)
	if err != nil {
		return Parameter{}, err
	}
	if data.Len() != 0 {
		return Parameter{}, errors.New("trailing BinRPC response data")
	}
	return p, nil
}

func readParam(data *bytes.Reader, depth int) (Parameter, error) {
	if depth > 32 {
		return Parameter{}, errors.New("BinRPC nesting too deep")
	}
	var typ uint32
	if err := binary.Read(data, binary.BigEndian, &typ); err != nil {
		return Parameter{}, err
	}
	p := Parameter{Type: typ}
	switch typ {
	case TypeInteger:
		if err := binary.Read(data, binary.BigEndian, &p.Number); err != nil {
			return Parameter{}, err
		}
	case TypeBoolean:
		var v uint32
		if err := binary.Read(data, binary.BigEndian, &v); err != nil {
			return Parameter{}, err
		}
		if v > 1 {
			return Parameter{}, errors.New("invalid binary boolean")
		}
		p.Flag = v == 1
	case TypeString:
		str, err := readLatin1(data)
		if err != nil {
			return Parameter{}, err
		}
		p.Text = str
	case TypeDouble:
		var m, e int32
		if err := binary.Read(data, binary.BigEndian, &m); err != nil {
			return Parameter{}, err
		}
		if err := binary.Read(data, binary.BigEndian, &e); err != nil {
			return Parameter{}, err
		}
		p.Float = math.Ldexp(float64(m)/float64(uint64(1)<<30), int(e))
		if math.IsInf(p.Float, 0) || math.IsNaN(p.Float) {
			return Parameter{}, errors.New("nonfinite double")
		}
	case TypeArray, TypeStruct:
		var count uint32
		if err := binary.Read(data, binary.BigEndian, &count); err != nil {
			return Parameter{}, err
		}
		if count > 1024 {
			return Parameter{}, errors.New("too many BinRPC members")
		}
		if typ == TypeArray {
			p.Items = make([]Parameter, 0, count)
			for i := uint32(0); i < count; i++ {
				v, err := readParam(data, depth+1)
				if err != nil {
					return Parameter{}, err
				}
				p.Items = append(p.Items, v)
			}
		} else {
			p.Fields = make(map[string]Parameter, count)
			for i := uint32(0); i < count; i++ {
				key, err := readLatin1(data)
				if err != nil {
					return Parameter{}, err
				}
				if _, ok := p.Fields[key]; ok {
					return Parameter{}, errors.New("duplicate BinRPC field")
				}
				v, err := readParam(data, depth+1)
				if err != nil {
					return Parameter{}, err
				}
				p.Fields[key] = v
			}
		}
	default:
		return Parameter{}, fmt.Errorf("unsupported BinRPC response type %d", typ)
	}
	return p, nil
}
func readLatin1(data *bytes.Reader) (string, error) {
	var n uint32
	if err := binary.Read(data, binary.BigEndian, &n); err != nil {
		return "", err
	}
	if n > uint32(data.Len()) {
		return "", io.ErrUnexpectedEOF
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(data, b); err != nil {
		return "", err
	}
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r), nil
}
