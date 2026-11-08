// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/numeric_clamp_wire_test.go
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
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	mb "github.com/WolfHenk/CCU-ModbusClient/internal/modbus"
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
	"io"
	"net"
	"testing"
	"time"
)

// Exercises range limits: input -> queue -> Modbus FC06/FC16 -> independent TCP listener.
func TestClampedNumericWriteWire(t *testing.T) {
	for _, tc := range []struct {
		typ         string
		input, want float64
		fc          byte
		regs        []uint16
	}{
		{"int16", -32768, -32768, 6, []uint16{0x8000}},
		{"uint16", 65535, 65535, 6, []uint16{0xffff}},
		{"int32", 2147483647, 2147483647, 16, []uint16{0x7fff, 0xffff}},
		{"uint32", 0, 0, 16, []uint16{0, 0}},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			ok := make(chan bool, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					ok <- false
					return
				}
				defer conn.Close()
				header := make([]byte, 7)
				if _, err = io.ReadFull(conn, header); err != nil {
					ok <- false
					return
				}
				body := make([]byte, int(binary.BigEndian.Uint16(header[4:]))-1)
				if _, err = io.ReadFull(conn, body); err != nil {
					ok <- false
					return
				}
				valid := body[0] == tc.fc && header[6] == 8 && binary.BigEndian.Uint16(body[1:3]) == 100
				resp := []byte{tc.fc, 0, 100}
				if tc.fc == 6 {
					valid = valid && binary.BigEndian.Uint16(body[3:5]) == tc.regs[0]
					resp = append(resp, body[3:5]...)
				} else {
					valid = valid && binary.BigEndian.Uint16(body[3:5]) == 2 && body[5] == 4 && binary.BigEndian.Uint16(body[6:8]) == tc.regs[0] && binary.BigEndian.Uint16(body[8:10]) == tc.regs[1]
					resp = append(resp, 0, 2)
				}
				packet := make([]byte, 7+len(resp))
				copy(packet, header[:4])
				binary.BigEndian.PutUint16(packet[4:6], uint16(1+len(resp)))
				packet[6] = 8
				copy(packet[7:], resp)
				_, err = conn.Write(packet)
				ok <- valid && err == nil
			}()
			reg := config.Register{ID: "r", Enabled: true, Type: "holding", NumericDirection: "output", DataType: tc.typ, Address: 100, Factor: 1}
			dev := config.Device{ID: "dev", Registers: []config.Register{reg}}
			e := New()
			e.active["dev"] = dev
			e.writers["dev"] = make(chan writeRequest, 1)
			if !e.EnqueueNumeric("dev", "r", tc.input) {
				t.Fatal("enqueue failed")
			}
			req := <-e.writers["dev"]
			if *req.numeric != tc.want {
				t.Fatalf("got %v expected %v", *req.numeric, tc.want)
			}
			client := mb.NewClient(listener.Addr().String(), 8, time.Second)
			defer client.Close()
			state := model.DeviceState{Registers: []model.RegisterState{{ID: "r"}}}
			e.writeNumeric(client, dev, &state, req)
			if !<-ok {
				t.Fatal("wrong FC06/FC16 wire bytes")
			}
			if state.ErrorCount != 0 || state.Registers[0].Value.Quality != model.QualityGood || state.Registers[0].Value.Value != tc.want {
				t.Fatalf("wrong state %+v", state)
			}
		})
	}
}
