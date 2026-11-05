// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/codec.go
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

// Package ccurpc contains independently developed CCU XML-RPC primitives.
// This package imports only the Go standard library. It is not wired into
// the production CCU interface until protocol and integration testing passes.
package ccurpc

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Value is an explicitly typed XML-RPC value. Unsupported types fail closed.
type Value struct {
	Kind    string
	Text    string
	Array   []Value
	Members map[string]Value
}

func String(s string) Value                { return Value{Kind: "string", Text: s} }
func Integer(i int) Value                  { return Value{Kind: "int", Text: fmt.Sprint(i)} }
func Array(values ...Value) Value          { return Value{Kind: "array", Array: values} }
func Struct(fields map[string]Value) Value { return Value{Kind: "struct", Members: fields} }
func Boolean(v bool) Value {
	if v {
		return Value{Kind: "boolean", Text: "1"}
	}
	return Value{Kind: "boolean", Text: "0"}
}

// MarshalCall creates a complete XML-RPC methodCall, without copying any
// third-party implementation. Caller must provide valid method identifiers.
func MarshalCall(method string, params ...Value) ([]byte, error) {
	if method == "" || strings.TrimSpace(method) != method || strings.ContainsAny(method, " <>\r\n\t") {
		return nil, errors.New("invalid XML-RPC method name")
	}
	var b bytes.Buffer
	b.WriteString("<?xml version=\"1.0\"?><methodCall><methodName>")
	if err := xml.EscapeText(&b, []byte(method)); err != nil {
		return nil, err
	}
	b.WriteString("</methodName><params>")
	for _, p := range params {
		b.WriteString("<param>")
		if err := writeValue(&b, p, 0); err != nil {
			return nil, err
		}
		b.WriteString("</param>")
	}
	b.WriteString("</params></methodCall>")
	return b.Bytes(), nil
}

// ParseMethodName validates the outer XML-RPC call and extracts the method.
// It deliberately does not claim support for XML-RPC parameter decoding yet.
func ParseMethodName(r io.Reader) (string, error) {
	var c struct {
		XMLName xml.Name `xml:"methodCall"`
		Name    string   `xml:"methodName"`
	}
	d := xml.NewDecoder(io.LimitReader(r, 1024*1024))
	if err := d.Decode(&c); err != nil {
		return "", err
	}
	if c.Name == "" {
		return "", errors.New("missing XML-RPC method")
	}
	return c.Name, nil
}

// writeValue limits recursive nesting to avoid unbounded structures.
func writeValue(b *bytes.Buffer, v Value, depth int) error {
	if depth > 32 {
		return errors.New("XML-RPC nesting exceeds limit")
	}
	b.WriteString("<value>")
	switch v.Kind {
	case "string", "int", "boolean", "double":
		b.WriteString("<" + v.Kind + ">")
		if err := xml.EscapeText(b, []byte(v.Text)); err != nil {
			return err
		}
		b.WriteString("</" + v.Kind + ">")
	case "array":
		b.WriteString("<array><data>")
		for _, item := range v.Array {
			if err := writeValue(b, item, depth+1); err != nil {
				return err
			}
		}
		b.WriteString("</data></array>")
	case "struct":
		b.WriteString("<struct>")
		keys := make([]string, 0, len(v.Members))
		for k := range v.Members {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString("<member><name>")
			if err := xml.EscapeText(b, []byte(k)); err != nil {
				return err
			}
			b.WriteString("</name>")
			if err := writeValue(b, v.Members[k], depth+1); err != nil {
				return err
			}
			b.WriteString("</member>")
		}
		b.WriteString("</struct>")
	default:
		return fmt.Errorf("unsupported XML-RPC value type %q", v.Kind)
	}
	b.WriteString("</value>")
	return nil
}
