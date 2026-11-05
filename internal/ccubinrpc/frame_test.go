// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccubinrpc/frame_test.go
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

package ccubinrpc

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	for _, kind := range []byte{0, 1, 0xff} {
		var b bytes.Buffer
		if e := WriteFrame(&b, kind, []byte{0, 1, 2, 255}); e != nil {
			t.Fatal(e)
		}
		out, e := ReadFrame(&b)
		if e != nil || out.Kind != kind || !bytes.Equal(out.Payload, []byte{0, 1, 2, 255}) {
			t.Fatalf("frame %+v %v", out, e)
		}
	}
}
func TestFrameRejectsInvalidHeaders(t *testing.T) {
	var buf bytes.Buffer
	if e := WriteFrame(&buf, 0, nil); e != nil {
		t.Fatal(e)
	}
	original := append([]byte(nil), buf.Bytes()...)
	for _, index := range []int{0, 3} {
		bad := append([]byte(nil), original...)
		if index == 3 {
			bad[index] = 2
		} else {
			bad[index] = 255
		}
		if _, e := ReadFrame(bytes.NewReader(bad)); e == nil {
			t.Fatalf("accepted bad header byte %d", index)
		}
	}
	bad := append([]byte(nil), original...)
	binary.BigEndian.PutUint32(bad[4:8], MaxPayload+1)
	if _, e := ReadFrame(bytes.NewReader(bad)); e == nil {
		t.Fatal("oversized frame accepted")
	}
	if _, e := ReadFrame(bytes.NewReader(original[:7])); e != io.ErrUnexpectedEOF {
		t.Fatalf("truncated header: %v", e)
	}
	if e := WriteFrame(io.Discard, 2, nil); e == nil {
		t.Fatal("invalid kind accepted")
	}
	if e := WriteFrame(io.Discard, 0, make([]byte, MaxPayload+1)); e == nil {
		t.Fatal("oversized output accepted")
	}
}
