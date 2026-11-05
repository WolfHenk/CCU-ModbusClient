// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/content_length_test.go
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
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestOpenCCURequiresContentLengthForLargeResponse(t *testing.T) {
	h := NewHandler()
	if err := h.Register("getParamsetDescription", func([]Value) (Value, error) {
		return Struct(map[string]Value{"DETAILS": String(strings.Repeat("x", 8000))}), nil
	}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	for _, method := range []string{"getParamsetDescription", "unknown"} {
		body, err := MarshalCall(method)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.Post(srv.URL, "text/xml", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		length := resp.ContentLength
		if resp.Header.Get("Content-Length") != strconv.FormatInt(length, 10) || length < 10 {
			t.Fatalf("%s: missing/incorrect content-length: %q length=%d transfer=%v", method, resp.Header.Get("Content-Length"), length, resp.TransferEncoding)
		}
		resp.Body.Close()
	}
}
