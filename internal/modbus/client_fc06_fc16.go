// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/modbus/client_fc06_fc16.go
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

package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
)

// WriteSingleRegister implements FC06 (one holding register).
func (c *Client) WriteSingleRegister(address, value uint16) error {
	var data [2]byte
	binary.BigEndian.PutUint16(data[:], value)
	return c.writeRegisters(6, address, data[:], 1)
}

// WriteMultipleRegisters implements FC16 (1..123 consecutive holding registers).
func (c *Client) WriteMultipleRegisters(address uint16, values []uint16) error {
	if len(values) < 1 || len(values) > 123 {
		return fmt.Errorf("FC16 quantity %d outside 1..123", len(values))
	}
	if uint32(address)+uint32(len(values)) > 65536 {
		return errors.New("FC16 register address overflow")
	}
	data := make([]byte, len(values)*2)
	for n, v := range values {
		binary.BigEndian.PutUint16(data[2*n:], v)
	}
	return c.writeRegisters(16, address, data, uint16(len(values)))
}

func (c *Client) writeRegisters(function byte, address uint16, data []byte, quantity uint16) error {
	if function != 6 && function != 16 {
		return fmt.Errorf("invalid register write function %d", function)
	}
	if quantity == 0 || uint32(address)+uint32(quantity) > 65536 {
		return errors.New("register address range invalid")
	}
	if function == 6 && (quantity != 1 || len(data) != 2) {
		return errors.New("FC06 requires exactly one register")
	}
	if function == 16 && (quantity > 123 || len(data) != int(quantity)*2) {
		return errors.New("FC16 invalid register quantity")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.connectLocked(); err != nil {
		return err
	}
	_ = c.conn.SetDeadline(time.Now().Add(c.Timeout))
	c.tx++
	if c.tx == 0 {
		c.tx++
	}
	tx := c.tx
	pdu := make([]byte, 5)
	pdu[0] = function
	binary.BigEndian.PutUint16(pdu[1:], address)
	if function == 6 {
		copy(pdu[3:], data)
	} else {
		binary.BigEndian.PutUint16(pdu[3:], quantity)
		pdu = append(pdu, byte(len(data)))
		pdu = append(pdu, data...)
	}
	adu := make([]byte, 7+len(pdu))
	binary.BigEndian.PutUint16(adu[0:], tx)
	binary.BigEndian.PutUint16(adu[4:], uint16(1+len(pdu)))
	adu[6] = c.UnitID
	copy(adu[7:], pdu)
	if _, err := c.conn.Write(adu); err != nil {
		return c.failLocked(err)
	}
	hdr := make([]byte, 7)
	if _, err := io.ReadFull(c.conn, hdr); err != nil {
		return c.failLocked(err)
	}
	if binary.BigEndian.Uint16(hdr[0:]) != tx {
		return c.failLocked(errors.New("transaction id mismatch"))
	}
	if binary.BigEndian.Uint16(hdr[2:]) != 0 {
		return c.failLocked(errors.New("protocol id mismatch"))
	}
	if hdr[6] != c.UnitID {
		return c.failLocked(errors.New("unit id mismatch"))
	}
	length := int(binary.BigEndian.Uint16(hdr[4:]))
	if length != 3 && length != 6 {
		return c.failLocked(fmt.Errorf("FC%02d invalid response length %d", function, length))
	}
	body := make([]byte, length-1)
	if _, err := io.ReadFull(c.conn, body); err != nil {
		return c.failLocked(err)
	}
	if body[0] == function|0x80 {
		if length != 3 {
			return c.failLocked(errors.New("malformed write exception"))
		}
		return &ExceptionError{Function: function, Code: body[1]}
	}
	if body[0] != function || length != 6 {
		return c.failLocked(errors.New("unexpected write response function or length"))
	}
	if binary.BigEndian.Uint16(body[1:3]) != address {
		return c.failLocked(errors.New("write echo address mismatch"))
	}
	if function == 6 {
		if binary.BigEndian.Uint16(body[3:5]) != binary.BigEndian.Uint16(data) {
			return c.failLocked(errors.New("write echo value mismatch"))
		}
	} else if binary.BigEndian.Uint16(body[3:5]) != quantity {
		return c.failLocked(errors.New("write echo quantity mismatch"))
	}
	return nil
}
