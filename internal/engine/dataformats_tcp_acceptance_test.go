// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/dataformats_tcp_acceptance_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	mb "github.com/WolfHenk/CCU-ModbusClient/internal/modbus"
	"io"
	"math"
	"net"
	"testing"
	"time"
)

// Actual Modbus/TCP client, TCP socket, MBAP/PDU and decoding (isolated loopback server).
func TestDataFormatsWireEndToEnd(t *testing.T) {
	cases := []struct {
		name, dt                      string
		value                         float64
		wire                          []byte
		write                         bool
		byteSwap, wordSwap, dwordSwap bool
	}{
		{"UINT8", "uint8", 255, []byte{0, 255}, true, false, false, false},
		{"INT8", "int8", -128, []byte{0, 128}, true, false, false, false},
		{"UINT16", "uint16", 65535, []byte{255, 255}, true, false, false, false},
		{"INT16", "int16", -32768, []byte{128, 0}, true, false, false, false},
		{"UINT32", "uint32", 4294967295, []byte{255, 255, 255, 255}, true, false, false, false},
		{"INT32", "int32", -2147483648, []byte{128, 0, 0, 0}, true, false, false, false},
		{"FLOAT32", "float32", 12.5, []byte{0x41, 0x48, 0, 0}, true, false, false, false},
		{"FLOAT32_WORD_SWAP", "float32", 12.5, []byte{0, 0, 0x41, 0x48}, true, false, true, false},
		{"INT32_BYTE_SWAP", "int32", -1234567, []byte{0xed, 0xff, 0x79, 0x29}, true, true, false, false},
		{"UINT64_EXACT", "uint64", 0, []byte{255, 255, 255, 255, 255, 255, 255, 255}, false, false, false, false},
		{"INT64_NEG", "int64", 0, []byte{255, 255, 255, 255, 255, 255, 255, 254}, false, false, false, false},
		{"UINT64_DWORD_SWAP", "uint64", 0, []byte{0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67}, false, false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := config.Register{ID: "r", Name: tc.name, Enabled: true, Type: "holding", NumericDirection: "input", DataType: tc.dt, Address: 100, Factor: 1, PollSeconds: 10, ByteSwap: tc.byteSwap, WordSwap: tc.wordSwap, DWordSwap: tc.dwordSwap}
			if tc.write {
				reg.NumericDirection = "output"
			}
			if err := config.ValidateRegister(reg); err != nil {
				t.Fatal(err)
			}
			var words []uint16
			if tc.write {
				var err error
				words, err = encodeNumericWrite(reg, tc.value)
				if err != nil {
					t.Fatal(err)
				}
				for j, w := range words {
					if w != binary.BigEndian.Uint16(tc.wire[2*j:]) {
						t.Fatalf("encoded word %d = %04x expected %04x", j, w, binary.BigEndian.Uint16(tc.wire[2*j:]))
					}
				}
			}
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()
			serverResult := make(chan error, 1)
			go func() {
				c, e := ln.Accept()
				if e != nil {
					serverResult <- e
					return
				}
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(2 * time.Second))
				for seq := 0; seq < 2; seq++ {
					if !tc.write && seq == 0 {
						seq = 1
					}
					h := make([]byte, 7)
					if _, e = io.ReadFull(c, h); e != nil {
						serverResult <- e
						return
					}
					n := int(binary.BigEndian.Uint16(h[4:]))
					b := make([]byte, n-1)
					if _, e = io.ReadFull(c, b); e != nil {
						serverResult <- e
						return
					}
					var reply []byte
					if seq == 0 {
						expectedFC := byte(6)
						if len(tc.wire) > 2 {
							expectedFC = 16
						}
						if b[0] != expectedFC {
							serverResult <- fmt.Errorf("FC=%d want=%d", b[0], expectedFC)
							return
						}
						if expectedFC == 6 {
							if len(b) != 5 || string(b[3:]) != string(tc.wire) {
								serverResult <- fmt.Errorf("FC06 payload=%x", b)
								return
							}
							reply = b
						} else {
							if int(binary.BigEndian.Uint16(b[3:5])) != len(tc.wire)/2 || int(b[5]) != len(tc.wire) || string(b[6:]) != string(tc.wire) {
								serverResult <- fmt.Errorf("FC16 payload=%x", b)
								return
							}
							reply = append([]byte{16}, b[1:5]...)
						}
					}
					if seq == 1 {
						if b[0] != 3 || int(binary.BigEndian.Uint16(b[3:5])) != len(tc.wire)/2 {
							serverResult <- fmt.Errorf("FC03 request=%x", b)
							return
						}
						reply = append([]byte{3, byte(len(tc.wire))}, tc.wire...)
					}
					out := make([]byte, 7+len(reply))
					copy(out, h)
					binary.BigEndian.PutUint16(out[4:], uint16(len(reply)+1))
					copy(out[7:], reply)
					if _, e = c.Write(out); e != nil {
						serverResult <- e
						return
					}
				}
				serverResult <- nil
			}()
			client := mb.NewClient(ln.Addr().String(), 4, time.Second)
			defer client.Close()
			if tc.write {
				if len(words) == 1 {
					err = client.WriteSingleRegister(100, words[0])
				} else {
					err = client.WriteMultipleRegisters(100, words)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			got, err := client.Read(3, 100, uint16(len(tc.wire)/2))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(tc.wire) {
				t.Fatalf("wire got %x expected %x", got, tc.wire)
			}
			raw, value, err := decode(reg, got)
			if err != nil {
				t.Fatal(err)
			}
			if tc.dt == "uint64" || tc.dt == "int64" {
				if _, ok := value.(string); !ok {
					t.Fatalf("64-bit not exact string: %T %v", value, value)
				}
			} else if v, ok := value.(float64); !ok || math.Abs(v-tc.value) > 1e-5 {
				t.Fatalf("decoded=%v raw=%v expected=%v", value, raw, tc.value)
			}
			if e := <-serverResult; e != nil {
				t.Fatal(e)
			}
		})
	}
}
