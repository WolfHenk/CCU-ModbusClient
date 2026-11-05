// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        cmd/ccu-modbusclientd/config_identity.go
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
	"fmt"
	"github.com/WolfHenk/CCU-ModbusClient/internal/ccurpc"
	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
	"os"
	"path/filepath"
	"strings"
)

// loadClientConfig assigns MBC identities before any RPC registration.
// Check mode is strictly read-only. The caller must use only the new
// CCU-ModbusClient's dedicated configuration path, never the legacy addon file.
const clientConfigRoot = "/usr/local/etc/config/addons/ccu-modbusclient"
const legacyConfigRoot = "/usr/local/etc/config/addons/ccu-modbus"

// Production paths must be contained in this add-on's own persistent directory.
// Outside the CCU, an explicit config path supports isolated local tests.
func validateClientConfigPath(path string) error {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return fmt.Errorf("absolute client config path required")
	}
	if clean == legacyConfigRoot || strings.HasPrefix(clean, legacyConfigRoot+string(filepath.Separator)) {
		return fmt.Errorf("refusing legacy CCU-Modbus config path %q", path)
	}
	if strings.HasPrefix(clean, "/usr/local/etc/config/addons/") && clean != clientConfigRoot && !strings.HasPrefix(clean, clientConfigRoot+string(filepath.Separator)) {
		return fmt.Errorf("refusing another addon's configuration %q", path)
	}
	// Reject symlinks on the file and every existing ancestor, preventing
	// redirection of persistent writes to another add-on.
	for cur := clean; ; cur = filepath.Dir(cur) {
		info, err := os.Lstat(cur)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("client configuration path contains symlink %q", cur)
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if cur == string(filepath.Separator) {
			break
		}
	}
	return nil
}

func loadClientConfig(path string, checkOnly bool) (*config.Config, error) {
	if err := validateClientConfigPath(path); err != nil {
		return nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	if errs := config.Validate(cfg); len(errs) != 0 {
		return nil, fmt.Errorf("invalid configuration: %w", errs[0])
	}
	assigned, changed, err := ccurpc.AllocateClientSerials(cfg)
	if err != nil {
		return nil, err
	}
	if changed && !checkOnly {
		if err := config.WriteAtomic(path, assigned); err != nil {
			return nil, fmt.Errorf("persist MBC serials before registration: %w", err)
		}
		assigned, err = config.Load(path)
		if err != nil {
			return nil, fmt.Errorf("re-read saved MBC serials: %w", err)
		}
	}
	return assigned, nil
}
