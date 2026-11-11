// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/config/channels_test.go
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
	"path/filepath"
	"testing"
)

func reg(id, typ, dt string) Register {
	return Register{ID: id, Name: id, Enabled: true, Type: typ, DataType: dt, Factor: 1, PollSeconds: 10}
}

// house36 is the layout of the test CCU before 2026-11-11 18:14:
// 3 coils, 4 input registers, 29 holding registers (file order).
func house36() Device {
	d := Device{ID: "dev", Name: "Haus", Enabled: true, Host: "192.168.3.200", Port: 502, UnitID: 1, CCUAddress: "MBC03C801F601", TimeoutMS: 1500}
	for i := 0; i < 3; i++ {
		d.Registers = append(d.Registers, reg(fmt.Sprintf("coil%d", i), "coil", "bool"))
	}
	for i := 0; i < 4; i++ {
		d.Registers = append(d.Registers, reg(fmt.Sprintf("ir%d", i), "input", "float32"))
	}
	for i := 4; i <= 32; i++ {
		d.Registers = append(d.Registers, reg(fmt.Sprintf("cs%02d", i), "holding", "float32"))
	}
	return d
}

func chans(d Device) map[string]int {
	m := map[string]int{}
	for _, r := range d.Registers {
		m[r.ID] = r.Channel
	}
	return m
}

func TestLegacyMigrationKeepsKnownNumbering(t *testing.T) {
	// numeric first in the file, binary later: the CCU knows binary first
	d := Device{Registers: []Register{reg("num", "holding", "int16"), {ID: "off", Type: "coil", DataType: "bool"}, reg("bin", "discrete", "bool"), reg("num2", "input", "uint16")}}
	EnsureChannels(&d)
	c := chans(d)
	if c["bin"] != 1 || c["num"] != 2 || c["num2"] != 3 || c["off"] != 4 || d.NextChannel != 5 {
		t.Fatalf("legacy numbering %v next %d", c, d.NextChannel)
	}
}

// Regression (test CCU 2026-11-11 18:19): appending a discrete input, a
// coil and four numbers moved existing channels (:4 became the new DI).
func TestAppendKeepsExistingChannels(t *testing.T) {
	old := &Config{Devices: []Device{house36()}}
	cfg := Config{Devices: []Device{house36()}}
	d := &cfg.Devices[0]
	d.Registers = append(d.Registers, reg("di37", "discrete", "bool"), reg("coil38", "coil", "bool"))
	for i := 39; i <= 42; i++ {
		d.Registers = append(d.Registers, reg(fmt.Sprintf("n%d", i), "input", "float32"))
	}
	AssignChannels(&cfg, old)
	c := chans(*d)
	if c["coil0"] != 1 || c["coil2"] != 3 || c["ir0"] != 4 || c["ir3"] != 7 || c["cs04"] != 8 || c["cs32"] != 36 {
		t.Fatalf("existing channels moved: %v", c)
	}
	if c["di37"] != 37 || c["coil38"] != 38 || c["n39"] != 39 || c["n42"] != 42 || d.NextChannel != 43 {
		t.Fatalf("new channels: %v next %d", c, d.NextChannel)
	}
	if err := ValidateChannels(*d); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteDisableReorderAndForeignNumbers(t *testing.T) {
	old := &Config{Devices: []Device{house36()}}
	EnsureAllChannels(old)
	// delete ir1, disable coil1, reverse the order, page sends bogus numbers
	cfg := Config{Devices: []Device{house36()}}
	d := &cfg.Devices[0]
	var regs []Register
	for _, r := range d.Registers {
		if r.ID == "ir1" {
			continue
		}
		if r.ID == "coil1" {
			r.Enabled = false
		}
		r.Channel = 99 // must be ignored
		regs = append([]Register{r}, regs...)
	}
	d.Registers = append(regs, reg("new", "holding", "int16"))
	AssignChannels(&cfg, old)
	c := chans(*d)
	if c["coil1"] != 2 || c["ir0"] != 4 || c["ir2"] != 6 || c["cs32"] != 36 {
		t.Fatalf("numbers changed: %v", c)
	}
	if c["new"] != 37 {
		t.Fatalf("new register got %d, deleted number must not be reused", c["new"])
	}
	// next save: the retired number 5 (ir1) stays retired
	old2 := &Config{Devices: []Device{*d}}
	cfg2 := Config{Devices: []Device{*d}}
	cfg2.Devices[0].Registers = append(append([]Register(nil), d.Registers...), reg("new2", "input", "int16"))
	AssignChannels(&cfg2, old2)
	if got := chans(cfg2.Devices[0])["new2"]; got != 38 {
		t.Fatalf("new2 got %d", got)
	}
}

func TestNewDeviceAndValidation(t *testing.T) {
	cfg := Config{Devices: []Device{{ID: "x", Registers: []Register{reg("a", "coil", "bool"), reg("b", "input", "int16")}}}}
	AssignChannels(&cfg, &Config{})
	if c := chans(cfg.Devices[0]); c["a"] != 1 || c["b"] != 2 {
		t.Fatalf("%v", c)
	}
	bad := Device{NextChannel: 3, Registers: []Register{{ID: "a", Channel: 1}, {ID: "b", Channel: 1}}}
	if ValidateChannels(bad) == nil {
		t.Fatal("duplicate accepted")
	}
	bad = Device{NextChannel: 2, Registers: []Register{{ID: "a", Channel: 2}}}
	if ValidateChannels(bad) == nil {
		t.Fatal("number above next accepted")
	}
}

// A file without numbers is migrated on load and the numbers are written
// with the next save.
func TestLoadMigratesAndWritePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := &Config{Devices: []Device{house36()}}
	cfg.Devices[0].Registers[0], cfg.Devices[0].Registers[5] = cfg.Devices[0].Registers[5], cfg.Devices[0].Registers[0] // ir2 first in the file
	for i := range cfg.Devices[0].Registers {
		cfg.Devices[0].Registers[i].Channel = 0
	}
	if err := WriteAtomic(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	c := chans(got.Devices[0])
	if c["coil1"] != 1 || c["coil0"] != 3 || c["ir2"] != 4 || c["ir3"] != 7 || c["cs04"] != 8 {
		t.Fatalf("persisted numbers %v", c)
	}
}
