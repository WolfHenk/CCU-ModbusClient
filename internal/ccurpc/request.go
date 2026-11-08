// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/request.go
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
	"encoding/xml"
	"errors"
	"fmt"
	"golang.org/x/net/html/charset"
	"io"
	"strings"
)

type methodCallEnvelope struct {
	XMLName xml.Name `xml:"methodCall"`
	Name    string   `xml:"methodName"`
	Params  []struct {
		Value responseValue `xml:"value"`
	} `xml:"params>param"`
}

// ParseCall decodes a bounded XML-RPC request. It rejects unknown roots,
// empty method names and unsupported value types instead of guessing.
func ParseCall(r io.Reader) (string, []Value, error) {
	raw, err := io.ReadAll(io.LimitReader(r, (1<<20)+1))
	if err != nil {
		return "", nil, err
	}
	if len(raw) > 1<<20 {
		return "", nil, errors.New("XML-RPC request too large")
	}
	var call methodCallEnvelope
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&call); err != nil {
		return "", nil, err
	}
	if call.Name == "" || strings.TrimSpace(call.Name) != call.Name || strings.ContainsAny(call.Name, " <>\r\n\t") {
		return "", nil, errors.New("invalid XML-RPC method name")
	}
	args := make([]Value, 0, len(call.Params))
	for index, p := range call.Params {
		v, e := decodeValue(p.Value, 0)
		if e != nil {
			return "", nil, fmt.Errorf("XML-RPC parameter %d: %w", index, e)
		}
		args = append(args, v)
	}
	return call.Name, args, nil
}

// MarshalResponse produces an XML-RPC result with one parameter.
func MarshalResponse(value Value) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="ISO-8859-1"?><methodResponse><params><param>`)
	if err := writeValue(&b, value, 0); err != nil {
		return nil, err
	}
	b.WriteString(`</param></params></methodResponse>`)
	return b.Bytes(), nil
}

// MarshalFault encodes an XML-RPC fault using the standard faultCode and faultString members.
func MarshalFault(code int, message string) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="ISO-8859-1"?><methodResponse><fault>`)
	if err := writeValue(&b, Struct(map[string]Value{"faultCode": Integer(code), "faultString": String(message)}), 0); err != nil {
		return nil, err
	}
	b.WriteString(`</fault></methodResponse>`)
	return b.Bytes(), nil
}
