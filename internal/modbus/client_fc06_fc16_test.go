// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/modbus/client_fc06_fc16_test.go
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

package modbus

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestFC06AndFC16TCP(t *testing.T) {
	for _, tc := range []struct {
		fc        byte
		exception bool
	}{
		{6, false}, {16, false}, {6, true}, {16, true},
	} {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() {
			conn, err := l.Accept()
			if err != nil {
				done <- err
				return
			}
			defer conn.Close()
			hdr := make([]byte, 7)
			if _, err = io.ReadFull(conn, hdr); err != nil {
				done <- err
				return
			}
			length := int(binary.BigEndian.Uint16(hdr[4:]))
			body := make([]byte, length-1)
			if _, err = io.ReadFull(conn, body); err != nil {
				done <- err
				return
			}
			if body[0] != tc.fc || hdr[6] != 2 || binary.BigEndian.Uint16(body[1:3]) != 100 {
				done <- errors.New("incorrect FC/address/unit")
				return
			}
			if tc.fc == 6 && binary.BigEndian.Uint16(body[3:5]) != 0x1234 {
				done <- errors.New("FC06 bad value")
				return
			}
			if tc.fc == 16 && (binary.BigEndian.Uint16(body[3:5]) != 2 || body[5] != 4 || binary.BigEndian.Uint16(body[6:8]) != 0x1234 || binary.BigEndian.Uint16(body[8:10]) != 0x5678) {
				done <- errors.New("FC16 payload")
				return
			}
			resp := []byte{tc.fc, 0, 0, 0, 0}
			copy(resp[1:3], body[1:3])
			if tc.fc == 6 {
				copy(resp[3:5], body[3:5])
			} else {
				binary.BigEndian.PutUint16(resp[3:5], 2)
			}
			if tc.exception {
				resp = []byte{tc.fc | 0x80, 2}
			}
			out := make([]byte, 7+len(resp))
			copy(out[0:4], hdr[0:4])
			binary.BigEndian.PutUint16(out[4:6], uint16(1+len(resp)))
			out[6] = 2
			copy(out[7:], resp)
			_, err = conn.Write(out)
			done <- err
		}()
		client := NewClient(l.Addr().String(), 2, time.Second)
		if tc.fc == 6 {
			err = client.WriteSingleRegister(100, 0x1234)
		} else {
			err = client.WriteMultipleRegisters(100, []uint16{0x1234, 0x5678})
		}
		if tc.exception {
			var ex *ExceptionError
			if !errors.As(err, &ex) || ex.Function != tc.fc || ex.Code != 2 {
				t.Errorf("FC%d exception: %v", tc.fc, err)
			}
		} else if err != nil {
			t.Errorf("FC%d write: %v", tc.fc, err)
		}
		if serverErr := <-done; serverErr != nil {
			t.Fatal(serverErr)
		}
		client.Close()
		l.Close()
	}
}
func TestFC16RejectInvalidBeforeIO(t *testing.T) {
	c := NewClient("127.0.0.1:1", 1, time.Second)
	for _, vals := range [][]uint16{nil, make([]uint16, 124)} {
		if err := c.WriteMultipleRegisters(10, vals); err == nil {
			t.Fatal("invalid quantity accepted")
		}
	}
	if err := c.WriteMultipleRegisters(65535, []uint16{1, 2}); err == nil {
		t.Fatal("address overflow accepted")
	}
}
