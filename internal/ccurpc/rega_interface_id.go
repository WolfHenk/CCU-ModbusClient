// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/rega_interface_id.go
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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ResolveClientInterfaceID queries the existing ReGa interface object by name.
// This is a read-only ReGa script, never a mutation of OpenCCU system files.
func ResolveClientInterfaceID(ctx context.Context, url string) (string, error) {
	script := `object o=dom.GetObject("CCU-ModbusClient"); if(o){Write(o.ID());}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(script))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	client := &http.Client{Timeout: 4 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ReGa script HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 4097))
	if err != nil {
		return "", err
	}
	if len(body) > 4096 {
		return "", errors.New("ReGa script response too large")
	}
	id := strings.TrimSpace(strings.SplitN(string(body), "<xml>", 2)[0])
	number, err := strconv.ParseUint(id, 10, 32)
	if err != nil || number == 0 {
		return "", fmt.Errorf("CCU-ModbusClient interface ID unavailable: %q", id)
	}
	return id, nil
}
