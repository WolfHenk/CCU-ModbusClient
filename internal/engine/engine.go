// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/engine/engine.go
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
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	mb "github.com/WolfHenk/CCU-ModbusClient/internal/modbus"
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
)

type testRequest struct {
	register *config.Register
	result   chan TestResult
}

type writeRequest struct {
	registerID string
	value      bool
	numeric    *float64
}

type Engine struct {
	mu      sync.RWMutex
	states  map[string]model.DeviceState
	writers map[string]chan writeRequest
	testers map[string]chan testRequest
	active  map[string]config.Device
	cancel  context.CancelFunc
}

func New() *Engine {
	return &Engine{
		states:  map[string]model.DeviceState{},
		writers: map[string]chan writeRequest{},
		testers: map[string]chan testRequest{},
		active:  map[string]config.Device{},
	}
}

func (e *Engine) Start(parent context.Context, cfg *config.Config) {
	ctx, cancel := context.WithCancel(parent)
	e.cancel = cancel
	for _, d := range cfg.Devices {
		d := d
		if !d.Enabled {
			continue
		}
		ch := make(chan writeRequest, 32)
		tests := make(chan testRequest)
		e.mu.Lock()
		e.writers[d.ID] = ch
		e.testers[d.ID] = tests
		e.active[d.ID] = d
		e.mu.Unlock()
		go e.runDevice(ctx, d, ch, tests)
	}
}

func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
}

func (e *Engine) EnqueueCoil(deviceID, registerID string, value bool) bool {
	e.mu.RLock()
	ch := e.writers[deviceID]
	e.mu.RUnlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- writeRequest{registerID: registerID, value: value}:
		return true
	default:
		return false
	}
}

// EnqueueNumeric schedules a write only for a configured holding output.
func (e *Engine) EnqueueNumeric(deviceID, registerID string, value float64) bool {
	e.mu.RLock()
	ch := e.writers[deviceID]
	device, ok := e.active[deviceID]
	e.mu.RUnlock()
	if ch == nil || !ok {
		return false
	}
	for _, reg := range device.Registers {
		if reg.ID == registerID {
			if !reg.Enabled || reg.Type != "holding" || reg.NumericDirection != "output" {
				return false
			}
			// Values outside the register range are rejected, never clamped:
			// writing a limit value the user did not request could move an actuator.
			if _, err := encodeNumericWrite(reg, value); err != nil {
				return false
			}
			v := value
			select {
			case ch <- writeRequest{registerID: registerID, numeric: &v}:
				return true
			default:
				return false
			}
		}
	}
	return false
}

// CheckNumeric reports why a numeric write would be rejected, nil if valid.
func (e *Engine) CheckNumeric(deviceID, registerID string, value float64) error {
	e.mu.RLock()
	device, ok := e.active[deviceID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("Geraet %s nicht aktiv", deviceID)
	}
	for _, reg := range device.Registers {
		if reg.ID == registerID {
			if !reg.Enabled || reg.Type != "holding" || reg.NumericDirection != "output" {
				return fmt.Errorf("Register %s ist nicht beschreibbar", registerID)
			}
			_, err := encodeNumericWrite(reg, value)
			return err
		}
	}
	return fmt.Errorf("Register %s unbekannt", registerID)
}

func (e *Engine) Snapshot() []model.DeviceState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]model.DeviceState, 0, len(e.states))
	for _, s := range e.states {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (e *Engine) setState(s model.DeviceState) {
	e.mu.Lock()
	e.states[s.ID] = s
	e.mu.Unlock()
}

func registerInterval(r config.Register) time.Duration {
	sec := r.PollSeconds
	if sec <= 0 {
		sec = 10
	}
	if sec < 1 {
		sec = 1
	}
	return time.Duration(sec) * time.Second
}

func (e *Engine) runDevice(ctx context.Context, d config.Device, writes <-chan writeRequest, tests <-chan testRequest) {
	state := model.DeviceState{ID: d.ID, Name: d.Name, Host: d.Host, Port: d.Port, UnitID: uint8(d.UnitID), State: "STARTING"}
	nextPoll := make([]time.Time, len(d.Registers))

	now := time.Now()
	for i, r := range d.Registers {
		q := model.QualityDisabled
		if r.Enabled {
			q = model.QualityStale
			nextPoll[i] = now
		}
		if err := config.ValidateRegister(r); err != nil {
			q = model.QualityConfigError
		}
		state.Registers = append(state.Registers, model.RegisterState{ID: r.ID, Name: r.Name, Value: model.Value{Quality: q}})
	}

	if errs := config.ValidateDevice(d); len(errs) > 0 {
		state.State = "CONFIG_ERROR"
		state.LastError = errs[0].Error()
		e.setState(state)
		return
	}
	e.setState(state)

	client := mb.NewClient(net.JoinHostPort(d.Host, fmt.Sprint(d.Port)), byte(d.UnitID), d.Timeout())
	defer client.Close()

	// 250 ms is only the local scheduler resolution. It does not generate
	// Modbus traffic unless a register is due.
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		now = time.Now()
		changed := false
		for i, r := range d.Registers {
			if !r.Enabled || nextPoll[i].IsZero() || now.Before(nextPoll[i]) {
				continue
			}
			e.pollRegister(d, client, &state, i, r)
			nextPoll[i] = now.Add(registerInterval(r))
			changed = true
		}
		if changed {
			recomputeDeviceState(&state)
			e.setState(state)
		}

		select {
		case <-ctx.Done():
			state.State = "STOPPED"
			state.Connected = false
			e.setState(state)
			return
		case t := <-tests:
			if t.register == nil {
				// The existing client has been used successfully only if its state is ONLINE.
				t.result <- TestResult{OK: state.State == "ONLINE", Error: func() string {
					if state.State != "ONLINE" {
						return "Geraet nicht online: " + state.State
					}
					return ""
				}()}
			} else {
				t.result <- testRegisterWithClient(client, *t.register)
			}
		case wr := <-writes:
			if wr.numeric != nil {
				e.writeNumeric(client, d, &state, wr)
			} else {
				e.writeCoil(client, d, &state, wr)
			}
			recomputeDeviceState(&state)
			e.setState(state)
		case <-ticker.C:
		}
	}
}

func (e *Engine) writeCoil(client *mb.Client, d config.Device, state *model.DeviceState, wr writeRequest) {
	idx := -1
	var reg config.Register
	for i, r := range d.Registers {
		if r.ID == wr.registerID {
			idx = i
			reg = r
			break
		}
	}
	if idx < 0 {
		state.LastError = "Unbekanntes Register fuer Schreibzugriff: " + wr.registerID
		state.ErrorCount++
		return
	}
	if !reg.Enabled || reg.Type != "coil" || reg.DataType != "bool" || reg.CoilDirection == "input" {
		state.LastError = "Register ist nicht als schaltbarer Coil konfiguriert: " + reg.Name
		state.ErrorCount++
		return
	}
	if err := client.WriteSingleCoil(reg.Address, wr.value); err != nil {
		var ex *mb.ExceptionError
		if errors.As(err, &ex) && ex.Function == 5 {
			when := time.Now()
			state.LastSeen = &when
			info := &model.ModbusException{
				Function: 0x85, Code: ex.Code, Message: model.ExceptionMessage(ex.Function, ex.Code), Timestamp: time.Now(),
			}
			state.Registers[idx].ModbusException = info
			// Legacy FC05 status retains the original representation for 01..04.
			if model.FC05ExceptionMessage(ex.Code) != "" {
				state.FC05Exception = info
			} else {
				state.FC05Exception = nil
			}
		} else {
			state.Registers[idx].ModbusException = nil
		}
		state.Registers[idx].Value.Quality = model.QualityReadError
		state.Registers[idx].Value.Error = humanError(err)
		state.LastError = humanError(err)
		state.ErrorCount++
		return
	}

	// A successful write response resolves the preceding FC05 exception.
	state.FC05Exception = nil
	state.Registers[idx].ModbusException = nil
	now := time.Now()
	state.Registers[idx].Value = model.Value{
		Raw:       wr.value,
		Value:     wr.value,
		Timestamp: now,
		Quality:   model.QualityGood,
	}
	state.LastSeen = &now
}

func (e *Engine) pollRegister(d config.Device, client *mb.Client, state *model.DeviceState, i int, r config.Register) {
	if err := config.ValidateRegister(r); err != nil {
		state.Registers[i].Value = model.Value{Quality: model.QualityConfigError, Error: err.Error()}
		return
	}

	fc := byte(0)
	switch r.Type {
	case "coil":
		fc = 1
	case "discrete":
		fc = 2
	case "holding":
		fc = 3
	case "input":
		fc = 4
	}

	tries := d.Retries + 1
	if tries < 1 {
		tries = 1
	}
	if tries > 3 {
		tries = 3
	}

	var data []byte
	var err error
	for n := 0; n < tries; n++ {
		data, err = client.Read(fc, r.Address, quantity(r))
		if err == nil {
			break
		}
		// A complete Modbus exception is a valid response. Repeating the same
		// invalid request cannot fix an illegal address/function/value.
		var ex *mb.ExceptionError
		if errors.As(err, &ex) {
			break
		}
		if n+1 < tries {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if err != nil {
		var ex *mb.ExceptionError
		if errors.As(err, &ex) {
			when := time.Now()
			state.LastSeen = &when
			state.Registers[i].ModbusException = &model.ModbusException{
				Function: ex.Function | 0x80, Code: ex.Code,
				Message: model.ExceptionMessage(ex.Function, ex.Code), Timestamp: time.Now(),
			}
		} else {
			state.Registers[i].ModbusException = nil
		}
		state.Registers[i].Value = model.Value{Quality: model.QualityReadError, Error: humanError(err)}
		state.ErrorCount++
		state.LastError = humanError(err)
		return
	}

	raw, val, err := decode(r, data)
	if err != nil {
		// The Modbus reply was successful; a decoding problem is local, not
		// an outstanding protocol exception from an earlier poll.
		state.Registers[i].ModbusException = nil
		state.Registers[i].Value = model.Value{Quality: model.QualityReadError, Error: err.Error()}
		state.ErrorCount++
		state.LastError = err.Error()
		return
	}

	now := time.Now()
	state.Registers[i].ModbusException = nil
	state.Registers[i].Value = model.Value{Raw: raw, Value: val, Timestamp: now, Quality: model.QualityGood}
	state.LastSeen = &now
}

func recomputeDeviceState(state *model.DeviceState) {
	anyGood := false
	anyError := false
	active := false
	protocolResponded := false
	state.ModbusException = nil

	for _, r := range state.Registers {
		if ex := r.ModbusException; ex != nil {
			protocolResponded = true
			if state.ModbusException == nil || ex.Timestamp.After(state.ModbusException.Timestamp) {
				state.ModbusException = ex
			}
		}
		switch r.Value.Quality {
		case model.QualityGood:
			anyGood = true
			active = true
		case model.QualityReadError, model.QualityConfigError:
			anyError = true
			active = true
		case model.QualityStale:
			active = true
		}
	}

	// A complete exception reply proves the server responded on the wire.
	state.Connected = anyGood || protocolResponded
	switch {
	case !active:
		state.State = "IDLE"
	case anyGood && anyError, protocolResponded && anyError:
		state.State = "DEGRADED"
	case anyGood:
		state.State = "ONLINE"
		state.LastError = ""
	case anyError:
		state.State = "OFFLINE"
	default:
		state.State = "STARTING"
	}
}

func humanError(err error) string {
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return "Zeitueberschreitung"
	}
	var ex *mb.ExceptionError
	if errors.As(err, &ex) {
		return "Modbus: " + model.ExceptionMessage(ex.Function, ex.Code)
	}
	return err.Error()
}

func LogSnapshot(states []model.DeviceState) {
	for _, s := range states {
		log.Printf("%s: status=%s connected=%v errors=%d", s.Name, s.State, s.Connected, s.ErrorCount)
	}
}
