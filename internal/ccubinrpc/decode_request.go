// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/decode_request.go
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
)

// DecodeRequest decodes one bounded BinRPC request and validates consumption.
func DecodeRequest(r io.Reader) (string, []Parameter, error) {
	frame, err := ReadFrame(r)
	if err != nil {
		return "", nil, err
	}
	if frame.Kind != 0 {
		return "", nil, fmt.Errorf("expected request, got message type 0x%02x", frame.Kind)
	}
	data := bytes.NewReader(frame.Payload)
	method, err := readLatin1(data)
	if err != nil {
		return "", nil, err
	}
	if method == "" {
		return "", nil, errors.New("empty BinRPC method name")
	}
	var count uint32
	if err := binary.Read(data, binary.BigEndian, &count); err != nil {
		return "", nil, err
	}
	if count > 1024 {
		return "", nil, errors.New("too many BinRPC parameters")
	}
	args := make([]Parameter, 0, count)
	for i := uint32(0); i < count; i++ {
		v, err := readParam(data, 0)
		if err != nil {
			return "", nil, fmt.Errorf("BinRPC parameter %d: %w", i, err)
		}
		args = append(args, v)
	}
	if data.Len() != 0 {
		return "", nil, errors.New("trailing BinRPC request bytes")
	}
	return method, args, nil
}

// EncodeResponse serializes one typed BinRPC result.
func EncodeResponse(w io.Writer, p Parameter) error {
	var payload bytes.Buffer
	if err := writeParam(&payload, p, 0); err != nil {
		return err
	}
	return WriteFrame(w, 1, payload.Bytes())
}

// EncodeFault serializes a structured BinRPC fault with code and message.
func EncodeFault(w io.Writer, code int32, message string) error {
	var payload bytes.Buffer
	err := writeParam(&payload, Parameter{Type: TypeStruct, Fields: map[string]Parameter{
		"faultCode": {Type: TypeInteger, Number: code}, "faultString": {Type: TypeString, Text: message},
	}}, 0)
	if err != nil {
		return err
	}
	return WriteFrame(w, 0xff, payload.Bytes())
}

// DecodeResult accepts a result or fault frame. A fault is always an error.
func DecodeResult(r io.Reader) (Parameter, error) {
	frame, err := ReadFrame(r)
	if err != nil {
		return Parameter{}, err
	}
	if frame.Kind != 1 && frame.Kind != 0xff {
		return Parameter{}, fmt.Errorf("unexpected BinRPC message type 0x%02x", frame.Kind)
	}
	rd := bytes.NewReader(frame.Payload)
	p, err := readParam(rd, 0)
	if err != nil {
		return Parameter{}, err
	}
	if rd.Len() != 0 {
		return Parameter{}, errors.New("trailing BinRPC result bytes")
	}
	if frame.Kind == 0xff {
		code, ok := p.Fields["faultCode"]
		if p.Type != TypeStruct || !ok || code.Type != TypeInteger {
			return Parameter{}, errors.New("invalid BinRPC fault structure")
		}
		msg, ok := p.Fields["faultString"]
		if !ok || msg.Type != TypeString {
			return Parameter{}, errors.New("invalid BinRPC fault text")
		}
		return Parameter{}, fmt.Errorf("BinRPC fault %d: %s", code.Number, msg.Text)
	}
	return p, nil
}
