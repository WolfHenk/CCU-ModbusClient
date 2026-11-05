// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/interface_registration.go
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
	"strings"
	"sync"
)

// InterfaceRegistration is the isolated XML-RPC entry point for a CCU-owned
// registration. A receiver is recorded only after its arguments are verified.
// Device announcements are separate: no network requests run within init.
type InterfaceRegistration struct {
	mu        sync.RWMutex
	receivers map[string]string
	OnInit    func(receiver, interfaceID string) error
}

func NewInterfaceRegistration() *InterfaceRegistration {
	return &InterfaceRegistration{receivers: make(map[string]string)}
}

func (r *InterfaceRegistration) Register(h *Handler) error {
	if r == nil || h == nil {
		return errors.New("missing interface registration handler")
	}
	return h.Register("init", func(args []Value) (Value, error) {
		if len(args) != 1 && len(args) != 2 {
			return Value{}, fmt.Errorf("init needs receiver address and optional interface ID")
		}
		for _, arg := range args {
			if arg.Kind != "string" {
				return Value{}, errors.New("init arguments must be strings")
			}
		}
		receiver := strings.TrimSpace(args[0].Text)
		if receiver == "" {
			return Value{}, errors.New("empty receiver address")
		}
		if len(args) == 1 {
			r.mu.Lock()
			delete(r.receivers, receiver)
			r.mu.Unlock()
		} else {
			if err := r.Bootstrap(receiver, strings.TrimSpace(args[1].Text)); err != nil {
				return Value{}, err
			}
		}
		return String(""), nil
	})
}

func (r *InterfaceRegistration) Receiver(address string) (string, bool) {
	if r == nil {
		return "", false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.receivers[address]
	return v, ok
}

// Bootstrap uses the same validated registration path as the CCU init call.
func (r *InterfaceRegistration) Bootstrap(receiver, identifier string) error {
	if r == nil {
		return errors.New("nil registration")
	}
	receiver = strings.TrimSpace(receiver)
	identifier = strings.TrimSpace(identifier)
	if receiver == "" || identifier == "" {
		return errors.New("empty interface registration")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.OnInit != nil {
		if err := r.OnInit(receiver, identifier); err != nil {
			return err
		}
	}
	r.receivers[receiver] = identifier
	return nil
}
