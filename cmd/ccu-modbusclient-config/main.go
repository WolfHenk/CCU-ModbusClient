// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        cmd/ccu-modbusclient-config/main.go
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

package main

import (
	"encoding/json"
	"fmt"
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccurpc"
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"github.com/WolfHenk/CCU-ModbusClient/internal/engine"
	"io"
	"os"
	"path/filepath"
)

var version = "0.3.5"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "-version" {
		fmt.Println(version)
		return
	}
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string, stdin io.Reader, stdout io.Writer) error {
	// Test requests are read-only: only a user-clicked Test button invokes
	// an isolated Modbus/TCP connection; the service configuration is untouched.
	if len(args) == 1 && (args[0] == "test_connection" || args[0] == "test_register") {
		body, err := io.ReadAll(io.LimitReader(stdin, 1048577))
		if err != nil {
			return err
		}
		if len(body) > 1048576 {
			return fmt.Errorf("test request exceeds 1 MiB")
		}
		var req struct {
			Device   config.Device   `json:"device"`
			Register config.Register `json:"register"`
		}
		if args[0] == "test_connection" {
			if err := json.Unmarshal(body, &req.Device); err != nil {
				return err
			}
			result := engine.TestConnection(req.Device)
			return json.NewEncoder(stdout).Encode(result)
		}
		if err := json.Unmarshal(body, &req); err != nil {
			return err
		}
		result := engine.TestRegister(req.Device, req.Register)
		return json.NewEncoder(stdout).Encode(result)
	}
	if len(args) != 2 {
		return fmt.Errorf("usage: ccu-modbusclient-config read|write /absolute/path/config.json")
	}
	action, path := args[0], args[1]
	if path != "/usr/local/etc/config/addons/ccu-modbusclient/config.json" && os.Getenv("CCU_MODBUSCLIENT_CONFIG_TEST") == "" {
		return fmt.Errorf("configuration path must belong to CCU-ModbusClient")
	}
	// Refuse symlinks in both the target file and its parents; a
	// redirected persistent directory must never write another add-on.
	for at := filepath.Clean(path); ; at = filepath.Dir(at) {
		info, err := os.Lstat(at)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("configuration path contains symlink: %s", at)
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if at == "/" {
			break
		}
	}
	switch action {
	case "read":
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}
		return json.NewEncoder(stdout).Encode(cfg)
	case "write":
		reader := io.LimitReader(stdin, 1048577)
		body, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		if len(body) > 1048576 {
			return fmt.Errorf("configuration exceeds 1 MiB")
		}
		var cfg config.Config
		if err := json.Unmarshal(body, &cfg); err != nil {
			return err
		}
		config.ApplyDefaults(&cfg)
		if errs := config.Validate(&cfg); len(errs) > 0 {
			return errs[0]
		}
		assigned, _, err := ccurpc.AllocateClientSerials(&cfg)
		if err != nil {
			return err
		}
		// Preserve identities supplied for existing devices; replacing one requires an explicit decision.
		old, err := config.Load(path)
		if err != nil {
			return err
		}
		ids := map[string]string{}
		for _, d := range old.Devices {
			ids[d.ID] = d.CCUAddress
		}
		for _, d := range assigned.Devices {
			if before, ok := ids[d.ID]; ok && before != "" && d.CCUAddress != before {
				return fmt.Errorf("device %s: changing serial is prohibited", d.ID)
			}
		}
		if err := config.WriteAtomic(path, assigned); err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, "OK")
		return err
	default:
		return fmt.Errorf("unknown action")
	}
}
