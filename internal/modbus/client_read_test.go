// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/modbus/client_read_test.go
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
	"strings"
	"testing"
	"time"
)

// These read tests use only a loopback Modbus TCP server.
// They never connect to real field devices.
func TestReadQuantityLimitsBeforeDial(t *testing.T) {
	tests := []struct {
		function byte
		quantity uint16
	}{
		{1, 0}, {1, 2001}, {2, 2001},
		{3, 0}, {3, 126}, {4, 126},
	}
	for _, tc := range tests {
		c := NewClient("127.0.0.1:1", 1, 100*time.Millisecond)
		_, err := c.Read(tc.function, 0, tc.quantity)
		if err == nil || !strings.Contains(err.Error(), "quantity") {
			t.Errorf("FC%02d quantity %d must fail before dial, got %v", tc.function, tc.quantity, err)
		}
		if c.conn != nil {
			t.Fatalf("FC%02d quantity %d dialed despite local validation", tc.function, tc.quantity)
		}
	}
}

func TestReadAcceptsMaximumNormQuantities(t *testing.T) {
	cases := []struct {
		function      byte
		quantity      uint16
		expectedBytes int
	}{
		{1, 2000, 250}, {2, 2000, 250},
		{3, 125, 250}, {4, 125, 250},
		{1, 9, 2}, {2, 9, 2},
		{3, 1, 2}, {4, 1, 2},
	}
	for _, tc := range cases {
		t.Run(string(rune('0'+tc.function))+"-"+string(rune('0'+tc.quantity%10)), func(t *testing.T) {
			addr, seen := fc05TestServer(t, false, func(req []byte) []byte {
				data := make([]byte, tc.expectedBytes)
				data[len(data)-1] = 0xa5
				return fc05Response(req, tc.function, append([]byte{byte(len(data))}, data...))
			})
			client := NewClient(addr, 1, 700*time.Millisecond)
			defer client.Close()
			payload, err := client.Read(tc.function, 0, tc.quantity)
			if err != nil || len(payload) != tc.expectedBytes || payload[len(payload)-1] != 0xa5 {
				t.Fatalf("FC%02d quantity %d: %x %v", tc.function, tc.quantity, payload, err)
			}
			request := <-seen
			if request[7] != tc.function || binary.BigEndian.Uint16(request[10:12]) != tc.quantity {
				t.Fatalf("FC%02d wrong request %x", tc.function, request)
			}
		})
	}
}

func TestReadRejectsMismatchToRequest(t *testing.T) {
	cases := []struct {
		name     string
		function byte
		quantity uint16
		answer   []byte
	}{
		{"FC03-extra-register", 3, 1, []byte{3, 4, 0, 1, 0, 2}},
		{"FC03-missing-register", 3, 2, []byte{3, 2, 0, 1}},
		{"FC04-extra-register", 4, 1, []byte{4, 4, 0, 1, 0, 2}},
		{"FC01-extra-byte", 1, 1, []byte{1, 2, 0, 0}},
		{"FC02-missing-byte", 2, 9, []byte{2, 1, 0}},
		{"FC01-wrong-bytecount", 1, 9, []byte{1, 1, 0, 0}},
		{"FC03-wrong-bytecount", 3, 1, []byte{3, 4, 0, 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			addr, _ := fc05TestServer(t, false, func(req []byte) []byte {
				return fc05Response(req, tc.answer[0], tc.answer[1:])
			})
			client := NewClient(addr, 1, 500*time.Millisecond)
			defer client.Close()
			if data, err := client.Read(tc.function, 0, tc.quantity); err == nil {
				t.Fatalf("invalid FC%02d reply accepted: %x", tc.function, data)
			}
			if client.conn != nil {
				t.Fatal("invalid response did not discard unsynchronized TCP stream")
			}
		})
	}
}

func TestReadValidExceptionRetainsConnectionAndInvalidDoesNot(t *testing.T) {
	addr, _ := fc05TestServer(t, false,
		func(req []byte) []byte { return fc05Response(req, 0x83, []byte{0x02}) },
		func(req []byte) []byte { return fc05Response(req, 3, []byte{2, 0x12, 0x34}) },
	)
	client := NewClient(addr, 1, 500*time.Millisecond)
	defer client.Close()
	_, err := client.Read(3, 0, 1)
	var ex *ExceptionError
	if !errors.As(err, &ex) || ex.Function != 3 || ex.Code != 2 {
		t.Fatalf("exception not recognized: %v", err)
	}
	data, err := client.Read(3, 0, 1)
	if err != nil || len(data) != 2 || data[0] != 0x12 {
		t.Fatalf("exception invalidated TCP socket: data=%x error=%v", data, err)
	}

	addr2, _ := fc05TestServer(t, false, func(req []byte) []byte {
		return fc05Response(req, 0x83, []byte{0x02, 0xff})
	})
	c2 := NewClient(addr2, 1, 500*time.Millisecond)
	defer c2.Close()
	_, err = c2.Read(3, 0, 1)
	if err == nil || c2.conn != nil {
		t.Fatalf("invalid exception ADU must be rejected and socket dropped: %v", err)
	}
}

func TestReadFragmentedValidReplies(t *testing.T) {
	for _, function := range []byte{1, 2, 3, 4} {
		t.Run(string(rune('0'+function)), func(t *testing.T) {
			answer := []byte{1, 0x80}
			if function >= 3 {
				answer = []byte{2, 0x12, 0x34}
			}
			addr, _ := fc05TestServer(t, true, func(req []byte) []byte {
				return fc05Response(req, function, answer)
			})
			c := NewClient(addr, 1, time.Second)
			defer c.Close()
			if _, err := c.Read(function, 3, 1); err != nil {
				t.Fatal(err)
			}
		})
	}
}
