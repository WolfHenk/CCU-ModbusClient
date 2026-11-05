// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/compare_test.go
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

import "testing"

func TestCompareValuesDeterministic(t *testing.T) {
	a := Struct(map[string]Value{"B": Integer(1), "A": Array(String("old"), Boolean(true))})
	b := Struct(map[string]Value{"A": Array(String("new")), "C": Integer(1)})
	diff := CompareValues(a, b)
	want := []string{"$.A.length", "$.A[0]", "$.B", "$.C"}
	if len(diff) != len(want) {
		t.Fatalf("diff=%+v", diff)
	}
	for i, w := range want {
		if diff[i].Path != w {
			t.Fatalf("diff[%d] = %+v", i, diff)
		}
	}
	if d := CompareValues(a, a); len(d) != 0 {
		t.Fatalf("equal trees differ: %+v", d)
	}
	if d := CompareValues(String("1"), Integer(1)); len(d) != 1 || d[0].Before != "type=string" {
		t.Fatalf("missing type comparison: %+v", d)
	}
}
