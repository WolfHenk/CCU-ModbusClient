// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/catalog.go
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
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Point is a CCU data-point definition and its current cached value.
// Read and Write are RPC access capabilities, not Modbus commands.
type Point struct {
	ID, Type, Unit            string
	Value                     Value
	Read, Write, Event        bool
	Flags                     int
	Control                   string
	Min, Max                  *float64
	Default, Minimum, Maximum *Value
	TabOrder                  int
}
type Channel struct {
	Address, Type string
	Version       int
	Points        []Point
}
type Device struct {
	Address, Type string
	Channels      []Channel
}

// MemoryCatalog is an independent in-memory device model for staged CCU
// compatibility tests; its addresses and data-point IDs are caller-defined.
type MemoryCatalog struct {
	mu       sync.RWMutex
	devices  map[string]Device
	channels map[string]Channel
	pending  []PointEvent
}

func NewMemoryCatalog() *MemoryCatalog {
	return &MemoryCatalog{devices: make(map[string]Device), channels: make(map[string]Channel)}
}
func (c *MemoryCatalog) AddDevice(d Device) error {
	if c == nil || d.Address == "" || strings.Contains(d.Address, ":") || d.Type == "" {
		return errors.New("invalid device identity")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.devices[d.Address]; exists {
		return errors.New("duplicate device")
	}
	if len(d.Channels) == 0 {
		return errors.New("device has no channels")
	}
	local := make(map[string]Channel)
	for _, ch := range d.Channels {
		if !strings.HasPrefix(ch.Address, d.Address+":") || ch.Type == "" {
			return fmt.Errorf("invalid channel %q", ch.Address)
		}
		if _, exists := local[ch.Address]; exists {
			return fmt.Errorf("duplicate channel %q", ch.Address)
		}
		if _, exists := c.channels[ch.Address]; exists {
			return fmt.Errorf("existing channel %q", ch.Address)
		}
		seen := make(map[string]bool)
		for _, p := range ch.Points {
			if p.ID == "" || p.Type == "" || seen[p.ID] {
				return fmt.Errorf("invalid point on %s", ch.Address)
			}
			seen[p.ID] = true
		}
		local[ch.Address] = ch
	}
	copyDev := cloneDevice(d)
	c.devices[d.Address] = copyDev
	for _, ch := range copyDev.Channels {
		c.channels[ch.Address] = ch
	}
	return nil
}
func cloneDevice(d Device) Device {
	d.Channels = append([]Channel(nil), d.Channels...)
	for i := range d.Channels {
		d.Channels[i].Points = append([]Point(nil), d.Channels[i].Points...)
	}
	return d
}
func (c *MemoryCatalog) ListDevices() ([]Value, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.devices))
	for addr := range c.devices {
		keys = append(keys, addr)
	}
	sort.Strings(keys)
	var values []Value
	for _, addr := range keys {
		d := c.devices[addr]
		values = append(values, deviceDescription(d))
		for _, ch := range d.Channels {
			values = append(values, channelDescription(d, ch))
		}
	}
	return values, nil
}
func description(address, typ string) Value {
	return Struct(map[string]Value{"ADDRESS": String(address), "TYPE": String(typ)})
}

// localVirtualAttributes contains fields verified against the legacy local RPC fixture.
func localVirtualAttributes(v Value) Value {
	m := v.Members
	m["AES_ACTIVE"] = Integer(0)
	m["AVAILABLE_FIRMWARE"] = String("")
	m["FIRMWARE"] = String("")
	m["GROUP"] = String("")
	m["LINK_SOURCE_ROLES"] = String("")
	m["LINK_TARGET_ROLES"] = String("")
	m["RF_ADDRESS"] = Integer(0)
	m["ROAMING"] = Integer(0)
	m["TEAM"] = String("")
	m["TEAM_CHANNELS"] = Array()
	m["TEAM_TAG"] = String("")
	return v
}
func deviceDescription(d Device) Value {
	children := make([]Value, 0, len(d.Channels))
	for _, ch := range d.Channels {
		children = append(children, String(ch.Address))
	}
	return localVirtualAttributes(Struct(map[string]Value{"PARENT_TYPE": String(""), "ADDRESS": String(d.Address), "TYPE": String(d.Type), "CHILDREN": Array(children...), "PARENT": String(""), "INDEX": Integer(0), "FLAGS": Integer(1), "DIRECTION": Integer(0), "RX_MODE": Integer(0), "INTERFACE": String(""), "VERSION": Integer(1), "PARAMSETS": Array(String("MASTER"))}))
}
func channelDescription(d Device, ch Channel) Value {
	// INDEX is the channel number of the address ("ADDR:n"), not the
	// position in the list: stable numbers may leave gaps.
	idx := 0
	if p := strings.LastIndexByte(ch.Address, ':'); p >= 0 {
		if n, err := strconv.Atoi(ch.Address[p+1:]); err == nil {
			idx = n
		}
	}
	return localVirtualAttributes(Struct(map[string]Value{"ADDRESS": String(ch.Address), "TYPE": String(ch.Type), "PARENT": String(d.Address), "PARENT_TYPE": String(d.Type), "INDEX": Integer(idx), "CHILDREN": Array(), "PARAMSETS": Array(String("MASTER"), String("VALUES")), "VERSION": Integer(max(1, ch.Version)), "FLAGS": Integer(func() int {
		if ch.Type == "MAINTENANCE" {
			return 3
		}
		return 1
	}()), "DIRECTION": Integer(0), "RX_MODE": Integer(0), "INTERFACE": String("")}))
}
func (c *MemoryCatalog) DeviceDescription(address string) (Value, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if d, ok := c.devices[address]; ok {
		return deviceDescription(d), nil
	}
	if ch, ok := c.channels[address]; ok {
		for _, d := range c.devices {
			for _, candidate := range d.Channels {
				if candidate.Address == ch.Address {
					return channelDescription(d, ch), nil
				}
			}
		}
		return Value{}, fmt.Errorf("orphan channel %s", address)
	}
	return Value{}, fmt.Errorf("unknown address %s", address)
}
func (c *MemoryCatalog) channel(address, set string) (Channel, error) {
	if set != "VALUES" && set != "MASTER" {
		return Channel{}, fmt.Errorf("unsupported paramset %q", set)
	}
	ch, ok := c.channels[address]
	if !ok {
		return Channel{}, fmt.Errorf("unknown channel %s", address)
	}
	return ch, nil
}
func (c *MemoryCatalog) ParamsetDescription(address, set string) (Value, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if set == "MASTER" {
		if _, exists := c.devices[address]; exists {
			return Struct(map[string]Value{}), nil
		}
	}
	ch, e := c.channel(address, set)
	if e != nil {
		return Value{}, e
	}
	if set == "MASTER" {
		return Struct(map[string]Value{}), nil
	}
	fields := make(map[string]Value)
	for _, p := range ch.Points {
		ops := 0
		if p.Read {
			ops |= 1
		}
		if p.Write {
			ops |= 2
		}
		if p.Event {
			ops |= 4
		}
		meta := map[string]Value{"ID": String(p.ID), "TYPE": String(p.Type), "UNIT": String(p.Unit), "OPERATIONS": Integer(ops), "FLAGS": Integer(p.Flags), "CONTROL": String(p.Control), "TAB_ORDER": Integer(p.TabOrder)}
		if p.Type == "INTEGER" || p.Type == "FLOAT" {
			meta["SPECIAL"] = Array()
		}
		if p.Default != nil {
			meta["DEFAULT"] = *p.Default
		}
		if p.Minimum != nil {
			meta["MIN"] = *p.Minimum
		}
		if p.Maximum != nil {
			meta["MAX"] = *p.Maximum
		}
		if p.Control != "" {
			meta["CONTROL"] = String(p.Control)
		}
		if p.Min != nil && p.Minimum == nil {
			meta["MIN"] = Value{Kind: "double", Text: fmt.Sprint(*p.Min)}
		}
		if p.Max != nil && p.Maximum == nil {
			meta["MAX"] = Value{Kind: "double", Text: fmt.Sprint(*p.Max)}
		}
		fields[p.ID] = Struct(meta)
	}
	return Struct(fields), nil
}
func (c *MemoryCatalog) Paramset(address, set string) (Value, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if set == "MASTER" {
		if _, exists := c.devices[address]; exists {
			return Struct(map[string]Value{}), nil
		}
	}
	ch, e := c.channel(address, set)
	if e != nil {
		return Value{}, e
	}
	if set == "MASTER" {
		return Struct(map[string]Value{}), nil
	}
	fields := make(map[string]Value)
	for _, p := range ch.Points {
		// Match go-hmccu's GetParamset: read every parameter value,
		// including write-only ACTION parameters.
		fields[p.ID] = p.Value
	}
	return Struct(fields), nil
}
