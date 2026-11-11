// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/snapshot.go
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

package ccurpc

import (
	"fmt"
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"github.com/WolfHenk/CCU-ModbusClient/internal/model"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// ApplySnapshot updates cached RPC values only. It performs no Modbus writes,
// ReGa calls or events; failed-quality register values are not propagated.
func (c *MemoryCatalog) ApplySnapshot(cfg *config.Config, states []model.DeviceState) error {
	if c == nil || cfg == nil {
		return fmt.Errorf("missing catalog/configuration")
	}
	byID := make(map[string]model.DeviceState, len(states))
	for _, s := range states {
		byID[s.ID] = s
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range cfg.Devices {
		if !d.Enabled {
			continue
		}
		ds, ok := byID[d.ID]
		if !ok {
			continue
		}
		addr := strings.TrimSpace(d.CCUAddress)
		if addr == "" {
			return fmt.Errorf("device %s missing stable address", d.ID)
		}
		if _, ok := c.devices[addr]; !ok {
			return fmt.Errorf("unknown device %s", addr)
		}
		update := func(index int, key string, v Value) {
			chAddr := fmt.Sprintf("%s:%d", addr, index)
			ch, ok := c.channels[chAddr]
			if !ok {
				return
			}
			for j := range ch.Points {
				if ch.Points[j].ID == key {
					if ch.Points[j].Read && ch.Points[j].Event && !reflect.DeepEqual(ch.Points[j].Value, v) {
						c.pending = append(c.pending, PointEvent{Address: chAddr, Key: key, Value: v})
					}
					ch.Points[j].Value = v
					break
				}
			}
			c.channels[chAddr] = ch
		}
		ex := ds.ModbusException
		if ex == nil {
			ex = ds.FC05Exception
		}
		code := 0
		if ex != nil {
			code = int(ex.Function)<<8 | int(ex.Code)
		}
		update(0, "UNREACH", Boolean(ds.State == "OFFLINE"))
		update(0, "MODBUS_ERROR", Integer(code))
		update(0, "MODBUS_EXCEPTION", Boolean(ex != nil))
		byReg := make(map[string]model.RegisterState, len(ds.Registers))
		for _, rs := range ds.Registers {
			byReg[rs.ID] = rs
		}
		for _, pc := range channelPlan(d) {
			r, index := pc.Reg, pc.Index
			rs, ok := byReg[r.ID]
			if !pc.Binary {
				if ok && rs.Value.Quality == model.QualityGood {
					applyNumeric(update, r, index, rs)
				}
				continue
			}
			if ok && rs.Value.Quality == model.QualityGood {
				if v, ok := rs.Value.Value.(bool); ok {
					if r.Type == "discrete" || r.CoilDirection == "input" {
						if r.InvertInput {
							v = !v
						}
					}
					update(index, "STATE", Boolean(v))
				}
			}
		}
	}
	return nil
}

// applyNumeric publishes one good numeric register value.
func applyNumeric(update func(int, string, Value), r config.Register, index int, rs model.RegisterState) {
	nc := numericChannelFor(r)
	switch v := rs.Value.Value.(type) {
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			if nc.DataType == "INTEGER" {
				update(index, nc.Key, Integer(int(math.Round(v))))
			} else {
				update(index, nc.Key, Value{Kind: "double", Text: strconv.FormatFloat(v, 'g', -1, 64)})
			}
		}
	case string:
		if r.DataType == "uint64" || r.DataType == "int64" {
			update(index, nc.Key, String(v))
		}
	}
}
