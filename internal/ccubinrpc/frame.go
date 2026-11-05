// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/frame.go
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

// Package ccubinrpc provides an independent, bounded frame transport for
// CCU binary RPC. Typed argument encoding is not implemented yet.
package ccubinrpc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const MaxPayload = 1 << 20

var signature = [3]byte{'B', 'i', 'n'}

// Frame contains a binary RPC message kind and its raw payload.
type Frame struct {
	Kind    byte
	Payload []byte
}

func WriteFrame(w io.Writer, kind byte, payload []byte) error {
	if kind != 0 && kind != 1 && kind != 0xff {
		return fmt.Errorf("invalid BinRPC message kind %d", kind)
	}
	if len(payload) > MaxPayload {
		return errors.New("BinRPC payload too large")
	}
	var header [8]byte
	copy(header[:3], signature[:])
	header[3] = kind
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	if err := writeFull(w, header[:]); err != nil {
		return err
	}
	return writeFull(w, payload)
}
func writeFull(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if n < 0 || n > len(b) {
			return io.ErrShortWrite
		}
		b = b[n:]
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func ReadFrame(r io.Reader) (Frame, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Frame{}, err
	}
	if header[0] != signature[0] || header[1] != signature[1] || header[2] != signature[2] {
		return Frame{}, errors.New("invalid BinRPC frame signature")
	}
	if header[3] != 0 && header[3] != 1 && header[3] != 0xff {
		return Frame{}, errors.New("invalid BinRPC message kind")
	}
	size := binary.BigEndian.Uint32(header[4:])
	if size > MaxPayload {
		return Frame{}, errors.New("BinRPC payload too large")
	}
	payload := make([]byte, int(size))
	if _, err := io.ReadFull(r, payload); err != nil {
		return Frame{}, err
	}
	return Frame{Kind: header[3], Payload: payload}, nil
}
