// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/exception_handling_test.go
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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	mb "github.com/WolfHenk/CCU-ModbusClient/internal/modbus"
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
)

// A local-only Modbus TCP server that always returns the selected exception.
// Real Modbus devices and CCU outputs are never addressed by these tests.
func exceptionServer(t *testing.T, code byte, requests *atomic.Int32) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				for {
					req := make([]byte, 12)
					if _, e := io.ReadFull(conn, req); e != nil {
						return
					}
					requests.Add(1)
					ans := make([]byte, 9)
					copy(ans[:7], req[:7])
					binary.BigEndian.PutUint16(ans[4:6], 3)
					ans[7], ans[8] = req[7]|0x80, code
					if _, e := conn.Write(ans); e != nil {
						return
					}
				}
			}()
		}
	}()
	return ln.Addr().String()
}

func TestRegisterExceptionsAllStandardCodesAllReadFunctions(t *testing.T) {
	for _, tc := range []struct {
		fc       byte
		kind     string
		datatype string
		code     byte
	}{
		{1, "coil", "bool", 1},
		{2, "discrete", "bool", 6},
		{3, "holding", "uint16", 2},
		{4, "input", "uint16", 11},
		{3, "holding", "uint16", 3},
		{3, "holding", "uint16", 4},
		{3, "holding", "uint16", 5},
		{4, "input", "uint16", 8},
		{4, "input", "uint16", 10},
		{3, "holding", "uint16", 9}, // preserve unknown code
	} {
		var requests atomic.Int32
		address := exceptionServer(t, tc.code, &requests)
		client := mb.NewClient(address, 1, 500*time.Millisecond)
		d := config.Device{Retries: 2}
		r := config.Register{ID: "register", Name: "Register", Enabled: true,
			Type: tc.kind, DataType: tc.datatype, PollSeconds: 1}
		s := model.DeviceState{Registers: []model.RegisterState{
			{ID: r.ID, Name: r.Name, Value: model.Value{Quality: model.QualityStale}},
		}}
		(&Engine{}).pollRegister(d, client, &s, 0, r)
		recomputeDeviceState(&s)
		ex := s.Registers[0].ModbusException
		if ex == nil || ex.Function != (tc.fc|0x80) || ex.Code != tc.code ||
			!strings.Contains(ex.Message, model.ExceptionMessage(tc.fc, tc.code)) {
			t.Errorf("FC%02X code%02X incorrectly recorded: %+v", tc.fc, tc.code, ex)
		}
		if s.ModbusException != ex || !s.Connected || s.State != "DEGRADED" || s.LastSeen == nil {
			t.Errorf("FC%02X code%02X is an answer, not disconnection: %+v", tc.fc, tc.code, s)
		}
		if got := requests.Load(); got != 1 {
			t.Errorf("FC%02X code%02X retried a complete exception %d times", tc.fc, tc.code, got)
		}
		_ = client.Close()
	}
}

func TestPerRegisterExceptionPersistsUntilItsOwnRecovery(t *testing.T) {
	ex := &model.ModbusException{Function: 0x83, Code: 2, Message: "83 02 Illegal Data Address", Timestamp: time.Now()}
	s := model.DeviceState{Registers: []model.RegisterState{
		{ID: "bad-register", ModbusException: ex, Value: model.Value{Quality: model.QualityReadError}},
		{ID: "good-register", Value: model.Value{Quality: model.QualityGood}},
	}}
	recomputeDeviceState(&s)
	if !s.Connected || s.State != "DEGRADED" || s.ModbusException != ex {
		t.Fatalf("healthy other register masked unresolved exception: %+v", s)
	}
	s.Registers[0].ModbusException = nil
	s.Registers[0].Value.Quality = model.QualityGood
	recomputeDeviceState(&s)
	if s.ModbusException != nil || s.State != "ONLINE" {
		t.Fatalf("recovered register did not clear exception: %+v", s)
	}
}

func TestTransportErrorNotReportedAsModbusException(t *testing.T) {
	s := model.DeviceState{Registers: []model.RegisterState{
		{ID: "register", Value: model.Value{Quality: model.QualityReadError, Error: "connection refused"}},
	}}
	recomputeDeviceState(&s)
	if s.ModbusException != nil || s.Connected || s.State != "OFFLINE" {
		t.Fatalf("transport error conflated with Modbus exception: %+v", s)
	}
}
