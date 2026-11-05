// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/model/model.go
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

package model

import (
	"fmt"
	"time"
)

type Quality string

const (
	QualityGood        Quality = "GOOD"
	QualityStale       Quality = "STALE"
	QualityReadError   Quality = "READ_ERROR"
	QualityConfigError Quality = "CONFIG_ERROR"
	QualityDisabled    Quality = "DISABLED"
)

type Value struct {
	Raw       any       `json:"raw,omitempty"`
	Value     any       `json:"value,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Quality   Quality   `json:"quality"`
	Error     string    `json:"error,omitempty"`
}

type RegisterState struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value Value  `json:"value"`
	// The last protocol exception of this register, if not cleared by success.
	ModbusException *ModbusException `json:"modbus_exception,omitempty"`
}

type DeviceState struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Host       string     `json:"host"`
	Port       int        `json:"port"`
	UnitID     uint8      `json:"unit_id"`
	Connected  bool       `json:"connected"`
	State      string     `json:"state"`
	LastSeen   *time.Time `json:"last_seen,omitempty"`
	LastError  string     `json:"last_error,omitempty"`
	ErrorCount uint64     `json:"error_count"`
	// FC05Exception records the last unresolved, valid FC05 exception (01..04).
	// This is not an UNREACH or TCP communication error.
	FC05Exception *ModbusException `json:"fc05_exception,omitempty"`
	// Latest unresolved protocol exception from any configured register.
	ModbusException *ModbusException `json:"modbus_exception,omitempty"`
	Registers       []RegisterState  `json:"registers"`
}

// ModbusException retains the protocol bytes and a human-readable rendering.
// A normal FC05 success clears the record; receiving an exception does not
// imply that the Modbus server is unreachable.
type ModbusException struct {
	Function  byte      `json:"function"`
	Code      byte      `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// FC05ExceptionMessage preserves the wire-format reply (85 XX). The label for
// code 04 follows the requested CCU wording; the Modbus specification calls
// it SERVER DEVICE FAILURE.
func FC05ExceptionMessage(code byte) string {
	var explanation string
	switch code {
	case 1:
		explanation = "Illegal Function"
	case 2:
		explanation = "Illegal Data Address"
	case 3:
		explanation = "Illegal Data Value"
	case 4:
		explanation = "Server Device Error"
	default:
		return ""
	}
	return fmt.Sprintf("85 %02X %s", code, explanation)
}

// FC05ServiceMessage is the intended CCU service-message wording. OpenCCU
// itself must resolve the service datapoint to this label in the WebUI;
// the XML-RPC Service flag alone does not provide a custom text translation.
func FC05ServiceMessage(code byte) string {
	if label := FC05ExceptionMessage(code); label != "" {
		return "Modbus: " + label
	}
	return ""
}
