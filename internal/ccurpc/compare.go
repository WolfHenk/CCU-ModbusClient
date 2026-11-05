// -----------------------------------------------------------------------------
// Projekt:      CCU-ModbusClient
// Datei:        internal/ccurpc/compare.go
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
	"fmt"
	"sort"
)

// Difference describes one reproducible mismatch between two RPC trees.
type Difference struct{ Path, Before, After string }

// CompareValues checks RPC field presence, wire types and values recursively.
// A missing field differs from a field with an empty value.
func CompareValues(old, next Value) []Difference {
	var result []Difference
	var visit func(string, Value, Value)
	visit = func(path string, a, b Value) {
		if a.Kind != b.Kind {
			result = append(result, Difference{path, "type=" + a.Kind, "type=" + b.Kind})
			return
		}
		switch a.Kind {
		case "struct":
			keys := map[string]bool{}
			for k := range a.Members {
				keys[k] = true
			}
			for k := range b.Members {
				keys[k] = true
			}
			sorted := make([]string, 0, len(keys))
			for k := range keys {
				sorted = append(sorted, k)
			}
			sort.Strings(sorted)
			for _, k := range sorted {
				va, hasA := a.Members[k]
				vb, hasB := b.Members[k]
				p := path + "." + k
				if !hasA {
					result = append(result, Difference{p, "<missing>", "present"})
				} else if !hasB {
					result = append(result, Difference{p, "present", "<missing>"})
				} else {
					visit(p, va, vb)
				}
			}
		case "array":
			if len(a.Array) != len(b.Array) {
				result = append(result, Difference{path + ".length", fmt.Sprint(len(a.Array)), fmt.Sprint(len(b.Array))})
			}
			for i := 0; i < len(a.Array) && i < len(b.Array); i++ {
				visit(fmt.Sprintf("%s[%d]", path, i), a.Array[i], b.Array[i])
			}
		default:
			if a.Text != b.Text {
				result = append(result, Difference{path, a.Text, b.Text})
			}
		}
	}
	visit("$", old, next)
	return result
}
