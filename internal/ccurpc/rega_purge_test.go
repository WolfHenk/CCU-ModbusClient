// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/rega_purge_test.go
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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type fakeReGa struct {
	mu        sync.Mutex
	iface     string
	devices   []string
	ifaceGone bool
	deleted   [][]string
	keep      bool // simulate ReGa ignoring deleteDevices
}

func (f *fakeReGa) script(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s := string(body)
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case strings.Contains(s, "dom.DeleteObject"):
		if !strings.Contains(s, "l.Remove("+f.iface+")") {
			http.Error(w, "interface not removed from list", 500)
			return
		}
		f.ifaceGone = true
		io.WriteString(w, "GONE")
	case strings.Contains(s, "ID_INTERFACES"):
		if f.iface != "" && !f.ifaceGone {
			io.WriteString(w, f.iface+"\r\n")
		}
	case strings.Contains(s, "ID_DEVICES"):
		if !strings.Contains(s, "== "+f.iface+")") {
			http.Error(w, "wrong interface id", 500)
			return
		}
		for _, d := range f.devices {
			io.WriteString(w, d+"\r\n")
		}
	}
	io.WriteString(w, "<xml><exec>/rega.exe</exec></xml>")
}

func (f *fakeReGa) servers(t *testing.T) ReGaEndpoints {
	h := NewHandler()
	if err := h.Register("deleteDevices", func(args []Value) (Value, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if len(args) != 2 || args[0].Text != f.iface {
			return Value{}, rpcInputError("bad interface")
		}
		var addrs []string
		for _, a := range args[1].Array {
			addrs = append(addrs, a.Text)
		}
		f.deleted = append(f.deleted, addrs)
		if !f.keep {
			f.devices = nil
		}
		return String(""), nil
	}); err != nil {
		t.Fatal(err)
	}
	rega := httptest.NewServer(http.HandlerFunc(f.script))
	cb := httptest.NewServer(h)
	t.Cleanup(rega.Close)
	t.Cleanup(cb.Close)
	return ReGaEndpoints{ScriptURL: rega.URL, CallbackURL: cb.URL}
}

func TestPurgeFromReGaRemovesDevicesAndInterface(t *testing.T) {
	f := &fakeReGa{iface: "1049", devices: []string{"MBC03CE01F601", "MBC075905DE01"}}
	res, err := PurgeFromReGa(context.Background(), f.servers(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.InterfaceID != "1049" || !res.InterfaceDeleted || len(res.DeletedDevices) != 2 {
		t.Fatalf("result %+v", res)
	}
	if len(f.deleted) != 1 || strings.Join(f.deleted[0], ",") != "MBC03CE01F601,MBC075905DE01" {
		t.Fatalf("deleteDevices calls %v", f.deleted)
	}
	if !f.ifaceGone {
		t.Fatal("interface object not deleted")
	}
	// A second run finds nothing and changes nothing.
	res, err = PurgeFromReGa(context.Background(), f.servers(t))
	if err != nil || res.InterfaceID != "" || len(f.deleted) != 1 {
		t.Fatalf("second run %+v %v %v", res, err, f.deleted)
	}
}

func TestPurgeFromReGaStopsWhenDevicesRemain(t *testing.T) {
	f := &fakeReGa{iface: "1049", devices: []string{"MBC03CE01F601"}, keep: true}
	if _, err := PurgeFromReGa(context.Background(), f.servers(t)); err == nil {
		t.Fatal("reported success although ReGa kept the device")
	}
	if f.ifaceGone {
		t.Fatal("interface deleted while devices still reference it")
	}
}
