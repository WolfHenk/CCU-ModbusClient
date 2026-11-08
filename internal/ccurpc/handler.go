// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/handler.go
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
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

// clientRPCFault marks only deliberately safe, short validation messages for
// transmission to ReGa. Unexpected internal errors remain redacted.
type clientRPCFault struct{ message string }

func (e clientRPCFault) Error() string   { return e.message }
func rpcInputError(message string) error { return clientRPCFault{message: message} }

// Method is a locally registered XML-RPC procedure.
type Method func([]Value) (Value, error)

// Handler is an independent XML-RPC dispatcher, not yet used by OpenCCU.
type Handler struct {
	mu      sync.RWMutex
	methods map[string]Method
}

func NewHandler() *Handler { return &Handler{methods: make(map[string]Method)} }
func (h *Handler) Register(name string, fn Method) error {
	if h == nil || fn == nil {
		return fmt.Errorf("nil XML-RPC handler")
	}
	if _, err := MarshalCall(name); err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.methods[name]; exists {
		return fmt.Errorf("duplicate XML-RPC method %q", name)
	}
	h.methods[name] = fn
	return nil
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name, args, err := ParseCall(r.Body)
	if err != nil {
		http.Error(w, "invalid XML-RPC request", http.StatusBadRequest)
		return
	}
	if h == nil {
		http.Error(w, "handler unavailable", http.StatusServiceUnavailable)
		return
	}
	h.mu.RLock()
	fn := h.methods[name]
	h.mu.RUnlock()
	if fn == nil {
		sendRPCFault(w, -32601, "unknown XML-RPC method")
		return
	}
	result, err := fn(args)
	if err != nil {
		var known clientRPCFault
		if errors.As(err, &known) {
			sendRPCFault(w, -32500, known.message)
		} else {
			sendRPCFault(w, -32500, "RPC method failed")
		}
		return
	}
	wire, err := MarshalResponse(result)
	if err != nil {
		http.Error(w, "RPC encoding failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(wire)))
	_, _ = bytes.NewReader(wire).WriteTo(w)
}

func sendRPCFault(w http.ResponseWriter, code int, msg string) {
	wire, err := MarshalFault(code, msg)
	if err != nil {
		http.Error(w, "RPC encoding failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(wire)))
	_, _ = w.Write(wire)
}
