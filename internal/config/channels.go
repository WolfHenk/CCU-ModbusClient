// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/config/channels.go
// Ersteller:    Wolfram Henkel
// Datum:        11. Nov. 2026
// Version:      0.3.7
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
//

package config

import (
	"fmt"
	"strings"
)

// Stable CCU channel numbers.
//
// Every register owns a channel number (Register.Channel, ":n" of the CCU
// device). It is assigned once and never changes: adding, removing,
// disabling or reordering registers does not move any other register to
// a different channel, so ReGa programs and links keep their meaning.
// Numbers of deleted registers are not handed out again
// (Device.NextChannel is the high-water mark).
//
// Configurations written before channel numbers existed are migrated with
// exactly the numbering the CCU already knows (legacy rule: enabled
// binary registers first, then enabled numeric registers, in file order);
// all other registers follow in file order.

// IsBinaryChannel reports whether r appears as a STATE channel.
func IsBinaryChannel(r Register) bool {
	return strings.EqualFold(r.DataType, "bool") && (r.Type == "coil" || r.Type == "discrete")
}

// IsNumericChannel reports whether r appears as a numeric channel.
func IsNumericChannel(r Register) bool {
	return !IsBinaryChannel(r) && (r.Type == "holding" || r.Type == "input")
}

// EnsureChannels gives every register of d a channel number. Registers
// that already have one keep it. A device without any numbers is migrated
// with the legacy rule. Afterwards NextChannel is above every used number.
func EnsureChannels(d *Device) {
	used := false
	for _, r := range d.Registers {
		if r.Channel > 0 {
			used = true
			break
		}
	}
	next := 1
	if !used {
		for _, pass := range []func(Register) bool{
			func(r Register) bool { return r.Enabled && IsBinaryChannel(r) },
			func(r Register) bool { return r.Enabled && IsNumericChannel(r) },
			func(r Register) bool { return true },
		} {
			for i := range d.Registers {
				if d.Registers[i].Channel == 0 && pass(d.Registers[i]) {
					d.Registers[i].Channel = next
					next++
				}
			}
		}
	}
	for _, r := range d.Registers {
		if r.Channel >= next {
			next = r.Channel + 1
		}
	}
	if d.NextChannel > next {
		next = d.NextChannel
	}
	for i := range d.Registers {
		if d.Registers[i].Channel == 0 {
			d.Registers[i].Channel = next
			next++
		}
	}
	d.NextChannel = next
}

// EnsureAllChannels runs EnsureChannels for every device.
func EnsureAllChannels(c *Config) {
	for i := range c.Devices {
		EnsureChannels(&c.Devices[i])
	}
}

// AssignChannels sets the channel numbers of a configuration that is about
// to be saved. The saved file is the authority: a register keeps the
// number it has in old (matched by device ID and register ID), whatever
// the incoming data says; new registers get numbers above everything the
// device ever used. old may be nil (first configuration).
func AssignChannels(c *Config, old *Config) {
	prev := map[string]Device{}
	if old != nil {
		o := *old
		o.Devices = append([]Device(nil), old.Devices...)
		for i := range o.Devices {
			d := o.Devices[i]
			d.Registers = append([]Register(nil), d.Registers...)
			EnsureChannels(&d)
			prev[d.ID] = d
		}
	}
	for i := range c.Devices {
		d := &c.Devices[i]
		p, known := prev[d.ID]
		ch := map[string]int{}
		next := 1
		if known {
			for _, r := range p.Registers {
				ch[r.ID] = r.Channel
			}
			next = p.NextChannel
		}
		for j := range d.Registers {
			d.Registers[j].Channel = ch[d.Registers[j].ID] // 0 for new registers
		}
		d.NextChannel = next
		if known {
			// numbers come only from the old file; new ones above its mark
			for j := range d.Registers {
				if d.Registers[j].Channel == 0 {
					d.Registers[j].Channel = d.NextChannel
					d.NextChannel++
				}
			}
		} else {
			EnsureChannels(d)
		}
	}
}

// ValidateChannels checks that the numbers of a device are set and unique.
func ValidateChannels(d Device) error {
	seen := map[int]string{}
	for _, r := range d.Registers {
		if r.Channel < 1 {
			return fmt.Errorf("Register %q hat keine Kanalnummer", r.ID)
		}
		if other, ok := seen[r.Channel]; ok {
			return fmt.Errorf("Register %q und %q haben dieselbe Kanalnummer %d", other, r.ID, r.Channel)
		}
		seen[r.Channel] = r.ID
		if d.NextChannel > 0 && r.Channel >= d.NextChannel {
			return fmt.Errorf("Kanalnummer %d von Register %q liegt nicht unter ccu_next_channel %d", r.Channel, r.ID, d.NextChannel)
		}
	}
	return nil
}
