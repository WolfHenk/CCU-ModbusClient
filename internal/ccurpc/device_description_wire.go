// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/device_description_wire.go
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

// DeviceDescriptionWire contains the fields of a standard ReGa XML-RPC device
// description. It deliberately has no dependency on the legacy Go library.
type DeviceDescriptionWire struct {
	Type, Address, Parent, ParentType, Firmware, AvailableFirmware          string
	LinkSourceRoles, LinkTargetRoles, Group, Team, TeamTag, Interface       string
	Children, Paramsets, TeamChannels                                       []string
	RFAddress, Index, AESActive, Version, Flags, Direction, Roaming, RXMode int
}

func (d DeviceDescriptionWire) RPCValue() Value {
	stringsArray := func(items []string) Value {
		out := make([]Value, 0, len(items))
		for _, s := range items {
			out = append(out, String(s))
		}
		return Array(out...)
	}
	return Struct(map[string]Value{
		"TYPE": String(d.Type), "ADDRESS": String(d.Address), "PARENT": String(d.Parent), "PARENT_TYPE": String(d.ParentType),
		"FIRMWARE": String(d.Firmware), "AVAILABLE_FIRMWARE": String(d.AvailableFirmware),
		"LINK_SOURCE_ROLES": String(d.LinkSourceRoles), "LINK_TARGET_ROLES": String(d.LinkTargetRoles),
		"GROUP": String(d.Group), "TEAM": String(d.Team), "TEAM_TAG": String(d.TeamTag), "INTERFACE": String(d.Interface),
		"CHILDREN": stringsArray(d.Children), "PARAMSETS": stringsArray(d.Paramsets), "TEAM_CHANNELS": stringsArray(d.TeamChannels),
		"RF_ADDRESS": Integer(d.RFAddress), "INDEX": Integer(d.Index), "AES_ACTIVE": Integer(d.AESActive),
		"VERSION": Integer(d.Version), "FLAGS": Integer(d.Flags), "DIRECTION": Integer(d.Direction),
		"ROAMING": Integer(d.Roaming), "RX_MODE": Integer(d.RXMode),
	})
}
