package main

import (
	"bytes"
	"encoding/json"
	"sort"
)

// orderedJSONMap wraps a map[string]any and marshals keys in a specified order.
// Keys in the order slice are output first (in that order), followed by
// any remaining keys in alphabetical order.
type orderedJSONMap struct {
	order  []string
	values map[string]any
}

func (m orderedJSONMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	written := make(map[string]bool, len(m.order))
	first := true

	for _, key := range m.order {
		val, ok := m.values[key]
		if !ok {
			continue
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false
		keyBytes, _ := json.Marshal(key)
		buf.Write(keyBytes)
		buf.WriteByte(':')
		valBytes, err := json.Marshal(val)
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
		written[key] = true
	}

	// Write remaining keys in alphabetical order
	var remaining []string
	for key := range m.values {
		if !written[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	for _, key := range remaining {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		keyBytes, _ := json.Marshal(key)
		buf.Write(keyBytes)
		buf.WriteByte(':')
		valBytes, err := json.Marshal(m.values[key])
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}
