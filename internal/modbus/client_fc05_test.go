// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/modbus/client_fc05_test.go
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
	"strings"
	"testing"
	"time"
)

// All FC05 protocol tests use loopback sockets. They never contact a Modbus device.
func fc05TestServer(t *testing.T, fragment bool, replies ...func([]byte) []byte) (string, <-chan []byte) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	requests := make(chan []byte, len(replies))
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		for _, reply := range replies {
			header := make([]byte, 7)
			if _, err := io.ReadFull(conn, header); err != nil {
				return
			}
			size := int(binary.BigEndian.Uint16(header[4:6])) - 1
			if size < 0 || size > 260 {
				return
			}
			data := make([]byte, size)
			if _, err := io.ReadFull(conn, data); err != nil {
				return
			}
			req := append(header, data...)
			requests <- req
			answer := reply(req)
			if fragment {
				for _, b := range answer {
					if _, err := conn.Write([]byte{b}); err != nil {
						return
					}
					time.Sleep(time.Millisecond)
				}
			} else {
				if _, err := conn.Write(answer); err != nil {
					return
				}
			}
		}
	}()
	return listener.Addr().String(), requests
}

func fc05Response(request []byte, function byte, payload []byte) []byte {
	header := append([]byte(nil), request[:7]...)
	binary.BigEndian.PutUint16(header[4:6], uint16(2+len(payload)))
	return append(header, append([]byte{function}, payload...)...)
}

func TestFC05SuccessEchoOnOff(t *testing.T) {
	for _, on := range []bool{false, true} {
		t.Run(map[bool]string{false: "off", true: "on"}[on], func(t *testing.T) {
			addr, requests := fc05TestServer(t, false, func(req []byte) []byte { return append([]byte(nil), req...) })
			client := NewClient(addr, 7, 500*time.Millisecond)
			defer client.Close()
			if err := client.WriteSingleCoil(0x1234, on); err != nil {
				t.Fatal(err)
			}
			req := <-requests
			if len(req) != 12 || req[7] != 5 || req[6] != 7 || binary.BigEndian.Uint16(req[4:6]) != 6 ||
				binary.BigEndian.Uint16(req[8:10]) != 0x1234 {
				t.Fatalf("incorrect FC05 request %x", req)
			}
			expected := uint16(0)
			if on {
				expected = 0xff00
			}
			if binary.BigEndian.Uint16(req[10:12]) != expected {
				t.Fatalf("incorrect FC05 coil value %x", req[10:12])
			}
		})
	}
}

func TestFC05ExceptionIsNotTimeoutAndConnectionRemainsUsable(t *testing.T) {
	addr, requests := fc05TestServer(t, false,
		func(req []byte) []byte { return fc05Response(req, 0x85, []byte{0x02}) },
		func(req []byte) []byte { return append([]byte(nil), req...) },
	)
	client := NewClient(addr, 1, 500*time.Millisecond)
	defer client.Close()
	err := client.WriteSingleCoil(9, true)
	var exception *ExceptionError
	if !errors.As(err, &exception) || exception.Function != 5 || exception.Code != 2 {
		t.Fatalf("expected exception FC05/02; got %T: %v", err, err)
	}
	if err := client.WriteSingleCoil(10, false); err != nil {
		t.Fatalf("valid exception wrongly invalidated TCP connection: %v", err)
	}
	first, second := <-requests, <-requests
	if binary.BigEndian.Uint16(first[0:2]) == binary.BigEndian.Uint16(second[0:2]) {
		t.Fatal("transaction ID was not incremented")
	}
}

func TestFC05FragmentedReply(t *testing.T) {
	for _, exception := range []bool{false, true} {
		name := "normal"
		if exception {
			name = "exception"
		}
		t.Run(name, func(t *testing.T) {
			addr, _ := fc05TestServer(t, true, func(req []byte) []byte {
				if exception {
					return fc05Response(req, 0x85, []byte{0x06})
				}
				return append([]byte(nil), req...)
			})
			client := NewClient(addr, 1, time.Second)
			defer client.Close()
			err := client.WriteSingleCoil(1, true)
			if !exception && err != nil {
				t.Fatal(err)
			}
			if exception {
				var ex *ExceptionError
				if !errors.As(err, &ex) || ex.Code != 6 {
					t.Fatalf("fragmented exception not parsed: %v", err)
				}
			}
		})
	}
}

func TestFC05RejectMalformedReplies(t *testing.T) {
	tests := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"wrong transaction", func(b []byte) []byte { b[1] ^= 1; return b }},
		{"wrong protocol", func(b []byte) []byte { b[3] = 1; return b }},
		{"wrong unit", func(b []byte) []byte { b[6] ^= 1; return b }},
		{"wrong echo address", func(b []byte) []byte { b[9] ^= 1; return b }},
		{"wrong echo value", func(b []byte) []byte { b[10] ^= 1; return b }},
		{"wrong function", func(b []byte) []byte { b[7] = 6; return b }},
		{"invalid mbap length", func(b []byte) []byte { b[5] = 3; return b }},
		{"exception with success length", func(b []byte) []byte { b[7] = 0x85; b[8] = 2; return b }},
		{"success with exception length", func(b []byte) []byte {
			return fc05Response(b, 5, []byte{0x02})
		}},
		{"truncated reply", func(b []byte) []byte { return b[:9] }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			addr, _ := fc05TestServer(t, false, func(req []byte) []byte {
				return tc.mutate(append([]byte(nil), req...))
			})
			client := NewClient(addr, 1, 120*time.Millisecond)
			defer client.Close()
			if err := client.WriteSingleCoil(3, true); err == nil {
				t.Fatal("malformed FC05 reply incorrectly accepted")
			}
		})
	}
}

func TestFC05RejectsInvalidMBAPWithoutWaitingForPayload(t *testing.T) {
	addr, _ := fc05TestServer(t, false, func(req []byte) []byte {
		b := append([]byte(nil), req[:7]...)
		binary.BigEndian.PutUint16(b[4:6], 400)
		return b
	})
	client := NewClient(addr, 1, 500*time.Millisecond)
	defer client.Close()
	err := client.WriteSingleCoil(3, true)
	if err == nil || !strings.Contains(err.Error(), "response length") {
		t.Fatalf("invalid oversized MBAP length not rejected: %v", err)
	}
}
