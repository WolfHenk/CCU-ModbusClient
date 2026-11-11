// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/channel_plan.go
// Ersteller:    Wolfram Henkel
// Datum:        11. Nov. 2026
// Version:      0.3.7
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
	"sort"

	"github.com/WolfHenk/CCU-ModbusClient/internal/config"
)

// plannedChannel is one register as it appears in the CCU.
type plannedChannel struct {
	Index  int // CCU channel number ":n"
	Binary bool
	Reg    config.Register
}

// channelPlan is the single source of the channel numbering used by the
// catalog, the value snapshot and the write routes. Numbers come from the
// configuration (stable, see config/channels.go); disabled registers and
// registers without a channel type leave their number unused.
func channelPlan(d config.Device) []plannedChannel {
	cp := d
	cp.Registers = append([]config.Register(nil), d.Registers...)
	config.EnsureChannels(&cp) // defensive: configurations built in code
	out := make([]plannedChannel, 0, len(cp.Registers))
	for _, r := range cp.Registers {
		if !r.Enabled {
			continue
		}
		switch {
		case config.IsBinaryChannel(r):
			out = append(out, plannedChannel{Index: r.Channel, Binary: true, Reg: r})
		case config.IsNumericChannel(r):
			out = append(out, plannedChannel{Index: r.Channel, Reg: r})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}
