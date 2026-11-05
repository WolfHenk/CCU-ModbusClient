// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/numeric_write_integration_test.go
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
	"io"
	"net"
	"testing"
	"time"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	mb "github.com/WolfHenk/CCU-ModbusClient/internal/modbus"
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
)

func TestEngineNumericWriteEndToEndSimulator(t *testing.T) {
	for _, tc := range []struct {
		typ   string
		fc    byte
		value float64
		want  []uint16
	}{
		{"uint16", 6, 1234, []uint16{1234}},
		{"float32", 16, 12.5, []uint16{0x4148, 0}},
	} {
		l, e := net.Listen("tcp", "127.0.0.1:0")
		if e != nil {
			t.Fatal(e)
		}
		result := make(chan bool, 1)
		go func() {
			cn, e := l.Accept()
			if e != nil {
				result <- false
				return
			}
			defer cn.Close()
			h := make([]byte, 7)
			if _, e = io.ReadFull(cn, h); e != nil {
				result <- false
				return
			}
			n := int(binary.BigEndian.Uint16(h[4:]))
			b := make([]byte, n-1)
			if _, e = io.ReadFull(cn, b); e != nil {
				result <- false
				return
			}
			valid := b[0] == tc.fc && binary.BigEndian.Uint16(b[1:3]) == 100 && h[6] == 2
			if tc.fc == 6 {
				valid = valid && binary.BigEndian.Uint16(b[3:5]) == tc.want[0]
			} else {
				valid = valid && binary.BigEndian.Uint16(b[3:5]) == 2 && b[5] == 4 && binary.BigEndian.Uint16(b[6:8]) == tc.want[0] && binary.BigEndian.Uint16(b[8:10]) == tc.want[1]
			}
			response := []byte{tc.fc, b[1], b[2]}
			if tc.fc == 6 {
				response = append(response, b[3:5]...)
			} else {
				response = append(response, 0, 2)
			}
			packet := make([]byte, 7+len(response))
			copy(packet[0:4], h[0:4])
			binary.BigEndian.PutUint16(packet[4:6], uint16(1+len(response)))
			packet[6] = 2
			copy(packet[7:], response)
			_, e = cn.Write(packet)
			result <- valid && e == nil
		}()
		reg := config.Register{ID: "r", Name: "Setpoint", Enabled: true, Type: "holding", NumericDirection: "output", DataType: tc.typ, Address: 100, Factor: 1, PollSeconds: 10}
		dev := config.Device{ID: "dev", Registers: []config.Register{reg}}
		state := model.DeviceState{Registers: []model.RegisterState{{ID: "r", Name: "Setpoint"}}}
		client := mb.NewClient(l.Addr().String(), 2, time.Second)
		v := tc.value
		(&Engine{}).writeNumeric(client, dev, &state, writeRequest{registerID: "r", numeric: &v})
		if !<-result {
			t.Errorf("FC%d mismatch in simulated server", tc.fc)
		}
		if state.ErrorCount != 0 || state.Registers[0].Value.Quality != model.QualityGood {
			t.Errorf("FC%d engine error %+v", tc.fc, state)
		}
		client.Close()
		l.Close()
	}
}
