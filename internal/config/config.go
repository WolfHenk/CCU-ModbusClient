// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/config/config.go
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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Schema  int      `json:"schema"`
	Listen  string   `json:"listen"`
	Devices []Device `json:"devices"`
}

type Device struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Enabled    bool       `json:"enabled"`
	Host       string     `json:"host"`
	Port       int        `json:"port"`
	UnitID     int        `json:"unit_id"`
	CCUAddress string     `json:"ccu_address,omitempty"`
	TimeoutMS  int        `json:"timeout_ms"`
	Retries    int        `json:"retries"`
	Registers  []Register `json:"registers"`
	// NextChannel is the next CCU channel number to hand out; numbers
	// below it are taken or retired (see channels.go).
	NextChannel int `json:"ccu_next_channel,omitempty"`
}

type Register struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Enabled          bool    `json:"enabled"`
	Type             string  `json:"type"`
	CoilDirection    string  `json:"coil_direction,omitempty"`
	NumericDirection string  `json:"numeric_direction,omitempty"`
	Address          uint16  `json:"address"`
	DataType         string  `json:"datatype"`
	Factor           float64 `json:"factor"`
	Offset           float64 `json:"offset"`
	ByteSwap         bool    `json:"byte_swap"`
	WordSwap         bool    `json:"word_swap"`
	DWordSwap        bool    `json:"dword_swap,omitempty"`
	PollSeconds      int     `json:"poll_seconds"`
	Unit             string  `json:"unit"`
	Measurement      string  `json:"measurement,omitempty"`
	TrueMeansOpen    bool    `json:"true_means_open,omitempty"`
	InvertInput      bool    `json:"invert_input,omitempty"`
	// Channel is the stable CCU channel number (":n"), see channels.go.
	Channel int `json:"ccu_channel,omitempty"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	ApplyDefaults(&c)
	EnsureAllChannels(&c) // files without numbers: legacy numbering
	return &c, nil
}

func ApplyDefaults(c *Config) {
	if c.Schema == 0 {
		c.Schema = 1
	}
	if c.Listen == "" {
		c.Listen = "127.0.0.1:18702"
	}
	for i := range c.Devices {
		if c.Devices[i].Port == 0 {
			c.Devices[i].Port = 502
		}
		if c.Devices[i].TimeoutMS <= 0 {
			c.Devices[i].TimeoutMS = 1500
		}
	}
}

func WriteAtomic(path string, c *Config) error {
	ApplyDefaults(c)
	if errs := Validate(c); len(errs) > 0 {
		return errs[0]
	}
	EnsureAllChannels(c)
	for _, d := range c.Devices {
		if err := ValidateChannels(d); err != nil {
			return fmt.Errorf("%s: %w", d.Name, err)
		}
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// CreateTemp uses exclusive creation in the target directory: no fixed
	// temporary path can be pre-planted as a symlink.
	f, err := os.CreateTemp(dir, ".ccu-modbusclient-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil {
		backup, err := os.CreateTemp(dir, ".ccu-modbusclient-backup-*.tmp")
		if err != nil {
			return err
		}
		backupPath := backup.Name()
		defer os.Remove(backupPath)
		if err := backup.Chmod(0644); err != nil {
			backup.Close()
			return err
		}
		if _, err := backup.Write(old); err != nil {
			backup.Close()
			return err
		}
		if err := backup.Sync(); err != nil {
			backup.Close()
			return err
		}
		if err := backup.Close(); err != nil {
			return err
		}
		if err := os.Rename(backupPath, path+".bak"); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmp, path)
}

func (d Device) Timeout() time.Duration {
	ms := d.TimeoutMS
	if ms <= 0 {
		ms = 1500
	}
	return time.Duration(ms) * time.Millisecond
}

func Validate(c *Config) []error {
	var out []error
	ids := map[string]bool{}
	addresses := map[string]bool{}
	for i, d := range c.Devices {
		for _, err := range ValidateDevice(d) {
			out = append(out, fmt.Errorf("Geraet %d: %w", i+1, err))
		}
		if ids[d.ID] {
			out = append(out, fmt.Errorf("Geraete-ID %q ist doppelt", d.ID))
		}
		ids[d.ID] = true
		if addr := strings.TrimSpace(d.CCUAddress); addr != "" {
			if addresses[addr] {
				out = append(out, fmt.Errorf("CCU-Geraeteadresse %q ist doppelt", addr))
			}
			addresses[addr] = true
		}
		rids := map[string]bool{}
		for n, r := range d.Registers {
			if err := ValidateRegister(r); err != nil {
				out = append(out, fmt.Errorf("%s, Register %d: %w", d.Name, n+1, err))
			}
			if rids[r.ID] {
				out = append(out, fmt.Errorf("%s: Register-ID %q ist doppelt", d.Name, r.ID))
			}
			rids[r.ID] = true
		}
	}
	return out
}

func ValidateDevice(d Device) []error {
	var out []error
	if strings.TrimSpace(d.ID) == "" {
		out = append(out, errors.New("device id fehlt"))
	}
	if strings.TrimSpace(d.Name) == "" {
		out = append(out, errors.New("device name fehlt"))
	}
	if strings.TrimSpace(d.Host) == "" {
		out = append(out, errors.New("host fehlt"))
	}
	if d.Port < 1 || d.Port > 65535 {
		out = append(out, fmt.Errorf("port %d ist ungueltig", d.Port))
	}
	if d.UnitID < 0 || d.UnitID > 255 {
		out = append(out, fmt.Errorf("unit_id %d ist ungueltig", d.UnitID))
	}
	if len(strings.TrimSpace(d.CCUAddress)) > 14 {
		out = append(out, fmt.Errorf("ccu_address %q ist laenger als 14 Zeichen", d.CCUAddress))
	}
	if d.TimeoutMS < 100 || d.TimeoutMS > 30000 {
		out = append(out, fmt.Errorf("timeout_ms %d ist ungueltig", d.TimeoutMS))
	}
	if d.Retries < 0 || d.Retries > 2 {
		out = append(out, fmt.Errorf("retries %d ist ungueltig", d.Retries))
	}
	return out
}

func ValidateRegister(r Register) error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("register id fehlt")
	}
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("register name fehlt")
	}
	if r.NumericDirection != "" && r.NumericDirection != "input" && r.NumericDirection != "output" {
		return fmt.Errorf("unbekannte numerische richtung %q", r.NumericDirection)
	}
	if r.NumericDirection != "" && r.Type != "holding" {
		return errors.New("numeric_direction nur fuer holding register erlaubt")
	}
	if r.NumericDirection == "output" {
		switch strings.ToLower(r.DataType) {
		case "uint8", "int8", "uint16", "int16", "uint32", "int32", "float32":
		default:
			return errors.New("numerischer ausgang erfordert unterstützten datentyp bis 32 bit")
		}
		if r.Factor == 0 {
			return errors.New("numerischer ausgang: factor darf nicht null sein")
		}
	}
	if r.CoilDirection != "" && r.CoilDirection != "input" && r.CoilDirection != "output" {
		return fmt.Errorf("unbekannte coil-richtung %q", r.CoilDirection)
	}
	if r.Type != "coil" && r.CoilDirection != "" {
		return errors.New("coil_direction ist nur fuer coils erlaubt")
	}
	switch r.Type {
	case "coil", "discrete", "holding", "input":
	default:
		return fmt.Errorf("unbekannter registertyp %q", r.Type)
	}
	switch strings.ToLower(r.DataType) {
	case "bool", "uint8", "int8", "uint16", "int16", "uint32", "int32", "uint64", "int64", "float32":
	default:
		return fmt.Errorf("unbekannter datentyp %q", r.DataType)
	}
	if (r.Type == "coil" || r.Type == "discrete") && strings.ToLower(r.DataType) != "bool" {
		return errors.New("coil/discrete erfordert datatype bool")
	}
	words := 1
	if r.Type == "holding" || r.Type == "input" {
		switch strings.ToLower(r.DataType) {
		case "uint32", "int32", "float32":
			words = 2
		case "uint64", "int64":
			words = 4
		}
	}
	if int(r.Address)+words-1 > 65535 {
		return fmt.Errorf("register ab adresse %d mit %d wort(en) ueberschreitet 65535", r.Address, words)
	}
	if r.PollSeconds < 1 || r.PollSeconds > 86400 {
		return fmt.Errorf("poll_seconds %d ist ungueltig", r.PollSeconds)
	}
	if r.Measurement != "" && r.Measurement != "generic" {
		valid := map[string]bool{"number": true, "temperature": true, "humidity": true, "voltage": true, "current": true, "power": true, "energy": true, "frequency": true}
		if !valid[r.Measurement] {
			return fmt.Errorf("unbekannte messgroesse %q", r.Measurement)
		}
		if r.Type != "holding" && r.Type != "input" {
			return errors.New("messgroesse erfordert numerisches register")
		}
		if strings.ToLower(r.DataType) == "bool" {
			return errors.New("messgroesse erfordert einen numerischen datentyp")
		}
		if strings.ToLower(r.DataType) == "int64" || strings.ToLower(r.DataType) == "uint64" {
			return errors.New("64-bit-werte werden noch nicht als spezialisierter Messkanal abgebildet")
		}
	}
	return nil
}
