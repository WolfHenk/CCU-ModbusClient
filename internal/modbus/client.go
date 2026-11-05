// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/modbus/client.go
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
	"net"
	"sync"
	"time"
)

// ExceptionError represents a Modbus exception response.
type ExceptionError struct {
	Function byte
	Code     byte
}

func (e *ExceptionError) Error() string {
	return fmt.Sprintf("modbus exception %d", e.Code)
}

// Client maintains one TCP connection for one Modbus device.
// Any I/O error invalidates the connection. The next call reconnects
// automatically. One client is used by one device worker, but the mutex
// keeps the type safe if that changes later.
type Client struct {
	Address string
	UnitID  byte
	Timeout time.Duration

	mu   sync.Mutex
	conn net.Conn
	tx   uint16
}

func NewClient(address string, unitID byte, timeout time.Duration) *Client {
	return &Client{Address: address, UnitID: unitID, Timeout: timeout}
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeLocked()
}

func (c *Client) closeLocked() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) connectLocked() error {
	if c.conn != nil {
		return nil
	}
	conn, err := net.DialTimeout("tcp", c.Address, c.Timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) failLocked(err error) error {
	_ = c.closeLocked()
	return err
}

func (c *Client) Read(function byte, address, quantity uint16) ([]byte, error) {
	if function < 1 || function > 4 {
		return nil, fmt.Errorf("unsupported function %d", function)
	}
	// Modbus Application Protocol v1.1b3, section 6:
	// FC01/02: 1..2000 bits; FC03/04: 1..125 registers.
	maxQuantity := uint16(2000)
	if function >= 3 {
		maxQuantity = 125
	}
	if quantity == 0 || quantity > maxQuantity {
		return nil, fmt.Errorf("quantity %d out of range for FC%02d (1..%d)", quantity, function, maxQuantity)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.connectLocked(); err != nil {
		return nil, err
	}
	_ = c.conn.SetDeadline(time.Now().Add(c.Timeout))

	c.tx++
	if c.tx == 0 {
		c.tx++
	}
	tx := c.tx

	pdu := make([]byte, 5)
	pdu[0] = function
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], quantity)

	adu := make([]byte, 7+len(pdu))
	binary.BigEndian.PutUint16(adu[0:2], tx)
	binary.BigEndian.PutUint16(adu[2:4], 0)
	binary.BigEndian.PutUint16(adu[4:6], uint16(1+len(pdu)))
	adu[6] = c.UnitID
	copy(adu[7:], pdu)

	if _, err := c.conn.Write(adu); err != nil {
		return nil, c.failLocked(err)
	}

	hdr := make([]byte, 7)
	if _, err := io.ReadFull(c.conn, hdr); err != nil {
		return nil, c.failLocked(err)
	}
	if binary.BigEndian.Uint16(hdr[0:2]) != tx {
		return nil, c.failLocked(errors.New("transaction id mismatch"))
	}
	if binary.BigEndian.Uint16(hdr[2:4]) != 0 {
		return nil, c.failLocked(errors.New("invalid protocol id"))
	}
	if hdr[6] != c.UnitID {
		return nil, c.failLocked(errors.New("unit id mismatch"))
	}

	length := int(binary.BigEndian.Uint16(hdr[4:6]))
	if length < 2 || length > 260 {
		return nil, c.failLocked(fmt.Errorf("invalid response length %d", length))
	}
	body := make([]byte, length-1)
	if _, err := io.ReadFull(c.conn, body); err != nil {
		return nil, c.failLocked(err)
	}
	if len(body) < 2 {
		return nil, c.failLocked(errors.New("short modbus response"))
	}
	if body[0] == function|0x80 {
		if len(body) != 2 {
			return nil, c.failLocked(fmt.Errorf("invalid FC%02d exception length %d", function, len(body)))
		}
		// A complete Modbus exception is a valid response: keep the
		// TCP connection alive without retrying the failed operation.
		return nil, &ExceptionError{Function: function, Code: body[1]}
	}
	if body[0] != function {
		return nil, c.failLocked(fmt.Errorf("unexpected function %d", body[0]))
	}

	byteCount := int(body[1])
	expectedBytes := int(quantity) * 2
	if function < 3 {
		expectedBytes = (int(quantity) + 7) / 8
	}
	// Check both the server's Byte Count and the payload length against
	// the original request. This prevents silently accepting extra values.
	if byteCount != expectedBytes || len(body)-2 != expectedBytes {
		return nil, c.failLocked(fmt.Errorf("FC%02d response byte count mismatch: got %d (%d bytes), want %d", function, byteCount, len(body)-2, expectedBytes))
	}
	return body[2:], nil
}

func (c *Client) WriteSingleCoil(address uint16, value bool) error {
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

	raw := uint16(0x0000)
	if value {
		raw = 0xFF00
	}
	pdu := make([]byte, 5)
	pdu[0] = 5
	binary.BigEndian.PutUint16(pdu[1:3], address)
	binary.BigEndian.PutUint16(pdu[3:5], raw)

	adu := make([]byte, 12)
	binary.BigEndian.PutUint16(adu[0:2], tx)
	binary.BigEndian.PutUint16(adu[2:4], 0)
	binary.BigEndian.PutUint16(adu[4:6], 6)
	adu[6] = c.UnitID
	copy(adu[7:], pdu)

	if _, err := c.conn.Write(adu); err != nil {
		return c.failLocked(err)
	}

	// The FC05 success response is a 12-byte ADU (MBAP Length=6).
	// An exception is a 9-byte ADU (MBAP Length=3). Read the MBAP header
	// first instead of blocking on a fixed-size success response.
	hdr := make([]byte, 7)
	if _, err := io.ReadFull(c.conn, hdr); err != nil {
		return c.failLocked(err)
	}
	if binary.BigEndian.Uint16(hdr[0:2]) != tx {
		return c.failLocked(errors.New("transaction id mismatch"))
	}
	if binary.BigEndian.Uint16(hdr[2:4]) != 0 {
		return c.failLocked(errors.New("invalid protocol id"))
	}
	if hdr[6] != c.UnitID {
		return c.failLocked(errors.New("unit id mismatch"))
	}
	length := int(binary.BigEndian.Uint16(hdr[4:6]))
	if length != 3 && length != 6 {
		return c.failLocked(fmt.Errorf("invalid FC05 response length %d", length))
	}
	body := make([]byte, length-1)
	if _, err := io.ReadFull(c.conn, body); err != nil {
		return c.failLocked(err)
	}
	switch body[0] {
	case 0x85:
		if length != 3 {
			return c.failLocked(fmt.Errorf("invalid FC05 exception length %d", length))
		}
		// An exception is a complete Modbus response. Preserve the TCP
		// connection, and do not retry an ambiguous write automatically.
		return &ExceptionError{Function: 5, Code: body[1]}
	case 5:
		if length != 6 {
			return c.failLocked(fmt.Errorf("invalid FC05 success length %d", length))
		}
		if binary.BigEndian.Uint16(body[1:3]) != address || binary.BigEndian.Uint16(body[3:5]) != raw {
			return c.failLocked(errors.New("write echo mismatch"))
		}
		return nil
	default:
		return c.failLocked(fmt.Errorf("unexpected function %d", body[0]))
	}
}
