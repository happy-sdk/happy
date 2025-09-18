// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

// Package adapters provides adapters for various logging systems.
// and utilities for Adapter implementations.
package adapters

import (
	"encoding/json"
	"log/slog"
	"sync"
)

const (
	// NL represents the newline character.
	NL byte = '\n'
	// SP represents the space character.
	SP byte = ' '
	// TAB represents the tab character.
	TAB byte = '\t'
	// CR represents the return character.
	CR byte = '\r'
)

// AttrMap is a pooled map for adapters to efficiently convert
// slog.Attr to map[string]any for example JSON encoding.
// It uses a sync.Pool to reuse maps, reducing allocation overhead.
// Use this instead of creating new map[string]any instances in adapters
// that frequently build structures from log attributes. Always call Free()
// when done to return the map to the pool, ensuring efficient reuse.
// Oversized maps (capacity > 32) are discarded to prevent excessive memory retention.
//
// Example:
//
//	attrs := []slog.Attr{slog.String("key", "value")}
//	m := logging.AttrGroupToMap(attrs)
//	data, _ := m.MarshalJSON()
//	m.Free()
type AttrMap map[string]any

// attrMapPool manages a pool of AttrMaps with an initial capacity of 6.
var attrMapPool = sync.Pool{
	New: func() any {
		m := make(map[string]any, 10)
		return (*AttrMap)(&m)
	},
}

// NewAttrMap retrieves an AttrMap from the pool with an initial capacity of 10.
// Always call Free() when done to return the map to the pool.
func NewAttrMap() *AttrMap {
	return attrMapPool.Get().(*AttrMap)
}

// Free returns the AttrMap to the pool if its capacity is ≤ 32, resetting its
// contents. Larger maps are discarded to prevent excessive memory use.
func (m *AttrMap) Free() {
	const maxMapSize = 32
	oldlen := len(*m)
	for k, v := range *m {
		if sm, ok := v.(*AttrMap); ok {
			sm.Free()
		}
		delete(*m, k)
	}
	if oldlen <= maxMapSize {
		attrMapPool.Put(m)
	}
}

// Reset clears the AttrMap's contents without returning it to the pool.
func (m *AttrMap) Reset() {
	for k, v := range *m {
		if sm, ok := v.(*AttrMap); ok {
			sm.Reset()
		}
		delete(*m, k)
	}
}

// Set adds or updates a key-value pair in the AttrMap, returning itself for
// chaining.
func (m *AttrMap) Set(key string, value any) *AttrMap {
	if v, ok := value.(slog.Value); ok {
		if v.Kind() == slog.KindGroup {
			(*m)[key] = AttrGroupToMap(v.Group())
		} else {
			(*m)[key] = v.Any()
		}
		return m
	}
	(*m)[key] = value
	return m
}

// Get retrieves the value for the given key, returning nil if the key does not
// exist.
func (m *AttrMap) Get(key string) any {
	return (*m)[key]
}

// Len returns the number of key-value pairs in the AttrMap.
func (m *AttrMap) Len() int {
	return len(*m)
}

// MarshalJSON implements json.Marshaler, encoding the AttrMap to JSON.
func (m *AttrMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(*m)
}

// AttrGroupToMap converts a slice of slog.Attr to an AttrMap, handling nested
// groups recursively. The resulting map is suitable for e.g. JSON encoding.
func AttrGroupToMap(gv []slog.Attr) *AttrMap {
	obj := NewAttrMap()
	for _, v := range gv {
		if v.Value.Kind() == slog.KindGroup {
			obj.Set(v.Key, AttrGroupToMap(v.Value.Group()))
		} else {
			obj.Set(v.Key, v.Value.Any())
		}
	}
	return obj
}
