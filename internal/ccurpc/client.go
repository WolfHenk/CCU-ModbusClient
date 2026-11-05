// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/client.go
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
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"golang.org/x/net/html/charset"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client is an independent, bounded HTTP/XML-RPC transport.
type Client struct {
	URL  string
	HTTP *http.Client
}

type responseEnvelope struct {
	XMLName xml.Name `xml:"methodResponse"`
	Params  []struct {
		Value responseValue `xml:"value"`
	} `xml:"params>param"`
	Fault *struct {
		Value responseValue `xml:"value"`
	} `xml:"fault"`
}
type responseValue struct {
	Text    string  `xml:",chardata"`
	String  *string `xml:"string"`
	Int     *string `xml:"int"`
	I4      *string `xml:"i4"`
	Boolean *string `xml:"boolean"`
	Double  *string `xml:"double"`
	Array   *struct {
		Items []responseValue `xml:"data>value"`
	} `xml:"array"`
	Struct *struct {
		Members []struct {
			Name  string        `xml:"name"`
			Value responseValue `xml:"value"`
		} `xml:"member"`
	} `xml:"struct"`
}

func simpleValue(v responseValue) (Value, error) { return decodeValue(v, 0) }
func decodeValue(v responseValue, depth int) (Value, error) {
	if depth > 32 {
		return Value{}, errors.New("XML-RPC nesting exceeds limit")
	}
	switch {
	case v.String != nil:
		return String(*v.String), nil
	case v.Int != nil:
		if _, e := strconv.ParseInt(*v.Int, 10, 32); e != nil {
			return Value{}, e
		}
		return Value{Kind: "int", Text: *v.Int}, nil
	case v.I4 != nil:
		if _, e := strconv.ParseInt(*v.I4, 10, 32); e != nil {
			return Value{}, e
		}
		return Value{Kind: "int", Text: *v.I4}, nil
	case v.Boolean != nil:
		if *v.Boolean != "0" && *v.Boolean != "1" {
			return Value{}, errors.New("invalid boolean")
		}
		return Value{Kind: "boolean", Text: *v.Boolean}, nil
	case v.Double != nil:
		if _, e := strconv.ParseFloat(*v.Double, 64); e != nil {
			return Value{}, e
		}
		return Value{Kind: "double", Text: *v.Double}, nil
	case v.Array != nil:
		items := make([]Value, 0, len(v.Array.Items))
		for _, a := range v.Array.Items {
			x, e := decodeValue(a, depth+1)
			if e != nil {
				return Value{}, e
			}
			items = append(items, x)
		}
		return Array(items...), nil
	case v.Struct != nil:
		members := make(map[string]Value, len(v.Struct.Members))
		for _, m := range v.Struct.Members {
			if _, ok := members[m.Name]; ok {
				return Value{}, errors.New("duplicate XML-RPC member")
			}
			x, e := decodeValue(m.Value, depth+1)
			if e != nil {
				return Value{}, e
			}
			members[m.Name] = x
		}
		return Struct(members), nil
	default:
		return String(strings.TrimSpace(v.Text)), nil
	}
}

// ParseResponse returns an explicitly typed result or an XML-RPC fault.
func ParseResponse(r io.Reader) (Value, error) {
	data, err := io.ReadAll(io.LimitReader(r, 1<<20+1))
	if err != nil {
		return Value{}, err
	}
	if len(data) > 1<<20 {
		return Value{}, errors.New("XML-RPC response too large")
	}
	var out responseEnvelope
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&out); err != nil {
		return Value{}, err
	}
	if out.Fault != nil {
		parts := make([]string, 0)
		if out.Fault.Value.Struct != nil {
			for _, m := range out.Fault.Value.Struct.Members {
				v, e := simpleValue(m.Value)
				if e == nil {
					parts = append(parts, m.Name+"="+v.Text)
				}
			}
		}
		return Value{}, fmt.Errorf("XML-RPC fault: %s", strings.Join(parts, ", "))
	}
	if len(out.Params) != 1 {
		return Value{}, fmt.Errorf("expected one XML-RPC result, got %d", len(out.Params))
	}
	return simpleValue(out.Params[0].Value)
}

func (c Client) Call(ctx context.Context, method string, params ...Value) (Value, error) {
	wire, err := MarshalCall(method, params...)
	if err != nil {
		return Value{}, err
	}
	if !strings.HasPrefix(c.URL, "http://") && !strings.HasPrefix(c.URL, "https://") {
		return Value{}, errors.New("explicit HTTP URL required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(wire))
	if err != nil {
		return Value{}, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return Value{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return Value{}, fmt.Errorf("XML-RPC HTTP status %d", res.StatusCode)
	}
	return ParseResponse(res.Body)
}
