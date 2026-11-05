// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/numeric_write_engine.go
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
	"errors"
	"time"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	mb "github.com/WolfHenk/CCU-ModbusClient/internal/modbus"
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
)

func (e *Engine) writeNumeric(client *mb.Client, d config.Device, state *model.DeviceState, wr writeRequest) {
	idx := -1
	var reg config.Register
	for n, r := range d.Registers {
		if r.ID == wr.registerID {
			idx = n
			reg = r
			break
		}
	}
	if idx < 0 || !reg.Enabled || reg.Type != "holding" || reg.NumericDirection != "output" {
		state.LastError = "Numerischer Schreibzugriff nicht erlaubt: " + wr.registerID
		state.ErrorCount++
		return
	}
	values, err := encodeNumericWrite(reg, *wr.numeric)
	if err != nil {
		state.LastError = err.Error()
		state.ErrorCount++
		return
	}
	if len(values) == 1 {
		err = client.WriteSingleRegister(reg.Address, values[0])
	} else {
		err = client.WriteMultipleRegisters(reg.Address, values)
	}
	if err != nil {
		var ex *mb.ExceptionError
		if errors.As(err, &ex) {
			state.Registers[idx].ModbusException = &model.ModbusException{Function: ex.Function | 0x80, Code: ex.Code, Message: model.ExceptionMessage(ex.Function, ex.Code), Timestamp: time.Now()}
		} else {
			state.Registers[idx].ModbusException = nil
		}
		state.Registers[idx].Value.Quality = model.QualityReadError
		state.Registers[idx].Value.Error = humanError(err)
		state.LastError = humanError(err)
		state.ErrorCount++
		return
	}
	state.Registers[idx].ModbusException = nil
	now := time.Now()
	state.Registers[idx].Value = model.Value{Raw: values, Value: *wr.numeric, Timestamp: now, Quality: model.QualityGood}
	state.LastSeen = &now
}
