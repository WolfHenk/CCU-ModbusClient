// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/rega_purge.go
// Ersteller:    Wolfram Henkel
// Datum:        08. Nov. 2026
// Version:      0.3.2
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

// ReGaEndpoints are the local OpenCCU endpoints used to remove the add-on.
type ReGaEndpoints struct {
	ScriptURL   string // ReGa script interface, e.g. http://127.0.0.1:8181/rega.exe
	CallbackURL string // ReGa XML-RPC callback, e.g. http://127.0.0.1:1999/
}

// PurgeResult reports what was removed from ReGa.
type PurgeResult struct {
	InterfaceID      string
	DeletedDevices   []string
	InterfaceDeleted bool
}

// The interface object is located by name and type, so a device or program
// that happens to carry the same name is never touched.
const purgeFindInterface = `string s; foreach(s, dom.GetObject(ID_INTERFACES).EnumIDs()) { object o = dom.GetObject(s); if (o) { if (o.Name() == "CCU-ModbusClient") { WriteLine(o.ID()); } } }`

const purgeListDevices = `string s; foreach(s, dom.GetObject(ID_DEVICES).EnumIDs()) { object d = dom.GetObject(s); if (d) { if (d.Interface() == %s) { WriteLine(d.Address()); } } }`

// The ID must also leave the ReGa interface list; DeleteObject alone leaves a
// dangling entry there (verified on OpenCCU 3.89.11).
const purgeDeleteInterface = `object l = dom.GetObject(ID_INTERFACES); object o = dom.GetObject(%[1]s); if (o) { if (o.Name() == "CCU-ModbusClient") { l.Remove(%[1]s); dom.DeleteObject(%[1]s); } } boolean listed = false; string s; foreach(s, l.EnumIDs()) { if (s == "%[1]s") { listed = true; } } object g = dom.GetObject(%[1]s); if (g) { listed = true; } if (listed) { Write("REMAINS"); } else { Write("GONE"); }`

// PurgeFromReGa removes every CCU-ModbusClient object from the ReGa logic
// layer. Devices (with their channels and datapoints) are removed by ReGa
// itself through its deleteDevices callback, the same path an interface
// process uses when a device disappears. Afterwards the now empty interface
// object is deleted. OpenCCU files are not touched. Calling it again when
// nothing is left is a no-op.
func PurgeFromReGa(ctx context.Context, ep ReGaEndpoints) (PurgeResult, error) {
	var res PurgeResult
	if ctx == nil || ep.ScriptURL == "" || ep.CallbackURL == "" {
		return res, errors.New("missing ReGa endpoints")
	}
	out, err := regaScript(ctx, ep.ScriptURL, purgeFindInterface)
	if err != nil {
		return res, err
	}
	ids := scriptLines(out)
	if len(ids) == 0 {
		return res, nil // Interface never registered or already removed.
	}
	if len(ids) > 1 {
		return res, fmt.Errorf("several ReGa interfaces named CCU-ModbusClient: %v", ids)
	}
	if _, err := strconv.ParseUint(ids[0], 10, 32); err != nil {
		return res, fmt.Errorf("invalid ReGa interface ID %q", ids[0])
	}
	res.InterfaceID = ids[0]
	addresses, err := purgeDevices(ctx, ep, res.InterfaceID)
	if err != nil {
		return res, err
	}
	if len(addresses) > 0 {
		values := make([]Value, len(addresses))
		for i, a := range addresses {
			values[i] = String(a)
		}
		if _, err := (Client{URL: ep.CallbackURL}).Call(ctx, "deleteDevices", String(res.InterfaceID), Array(values...)); err != nil {
			return res, fmt.Errorf("ReGa deleteDevices: %w", err)
		}
		remaining, err := purgeDevices(ctx, ep, res.InterfaceID)
		if err != nil {
			return res, err
		}
		if len(remaining) > 0 {
			return res, fmt.Errorf("ReGa kept devices after deleteDevices: %v", remaining)
		}
		res.DeletedDevices = addresses
	}
	out, err = regaScript(ctx, ep.ScriptURL, fmt.Sprintf(purgeDeleteInterface, res.InterfaceID))
	if err != nil {
		return res, err
	}
	if strings.TrimSpace(out) != "GONE" {
		return res, fmt.Errorf("ReGa interface object %s not deleted: %q", res.InterfaceID, out)
	}
	res.InterfaceDeleted = true
	return res, nil
}

func purgeDevices(ctx context.Context, ep ReGaEndpoints, id string) ([]string, error) {
	out, err := regaScript(ctx, ep.ScriptURL, fmt.Sprintf(purgeListDevices, id))
	if err != nil {
		return nil, err
	}
	return scriptLines(out), nil
}

// regaScript runs a ReGa script and returns its output without the trailing
// <xml> variable dump.
func regaScript(ctx context.Context, url, script string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(script))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ReGa script HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return strings.SplitN(string(body), "<xml>", 2)[0], nil
}

func scriptLines(out string) []string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}
