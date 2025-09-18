// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package adapters

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestChars(t *testing.T) {
	testutils.Assert(t, rune(NL) == rune('\n'), "NewLine should be equal to '\\n'")
	testutils.Assert(t, rune(SP) == rune(' '), "Space should be equal to ' '")
	testutils.Assert(t, rune(TAB) == rune('\t'), "Tab should be equal to '\\t'")
	testutils.Assert(t, rune(CR) == rune('\r'), "Tab should be equal to '\\r'")
}

func TestAttrMap(t *testing.T) {
	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int64("key2", 42),
		slog.Bool("key3", true),
		slog.String("key4", "value4"),
		slog.Group("key5", slog.String("gkey1", "gvalue1"), slog.Int64("gkey2", 43)),
	}

	m := NewAttrMap()

	for _, attr := range attrs {
		m.Set(attr.Key, attr.Value)
	}
	data, err := m.MarshalJSON()
	testutils.NoError(t, err, "AttrMap.MarshalJSON should not return an error")
	testutils.Assert(t, len(data) > 0, "AttrMap.MarshalJSON should return a non-empty byte slice")

	testutils.Assert(t,
		string(data) == `{"key1":"value1","key2":42,"key3":true,"key4":"value4","key5":{"gkey1":"gvalue1","gkey2":43}}`,
		"AttrMap.MarshalJSON should return a valid JSON string got: %q",
		string(data),
	)

	testutils.Equal(t, "value1", m.Get("key1").(string))
	testutils.Equal(t, 42, m.Get("key2").(int64))
	testutils.Equal(t, true, m.Get("key3").(bool))
	testutils.Equal(t, "value4", m.Get("key4").(string))

	testutils.Assert(t, m.Len() == len(attrs), "AttrMap.Len should return the number of attributes")
	m.Reset()
	testutils.Assert(t, m.Len() == 0, "AttrMap.Len should be 0 after reset")
}

func TestAttrGroupToMap(t *testing.T) {
	tests := []struct {
		name     string
		input    []slog.Attr
		expected map[string]any // Expected values, with nested groups as *AttrMap
	}{
		{
			name:     "empty attributes",
			input:    []slog.Attr{},
			expected: map[string]any{},
		},
		{
			name: "flat attributes",
			input: []slog.Attr{
				slog.String("name", "test"),
				slog.Int("count", 42),
				slog.Bool("enabled", true),
				slog.Float64("pi", 3.14),
			},
			expected: map[string]any{
				"name":    "test",
				"count":   int64(42), // slog.Int uses int64
				"enabled": true,
				"pi":      3.14,
			},
		},
		{
			name: "nested group",
			input: []slog.Attr{
				slog.Group("user",
					slog.String("name", "alice"),
					slog.Int("age", 30),
				),
			},
			expected: map[string]any{
				"user": &AttrMap{
					"name": "alice",
					"age":  int64(30),
				},
			},
		},
		{
			name: "mixed attributes and groups",
			input: []slog.Attr{
				slog.String("env", "prod"),
				slog.Group("config",
					slog.Bool("debug", false),
					slog.Group("db",
						slog.String("host", "localhost"),
						slog.Int("port", 5432),
					),
				),
			},
			expected: map[string]any{
				"env": "prod",
				"config": &AttrMap{
					"debug": false,
					"db": &AttrMap{
						"host": "localhost",
						"port": int64(5432),
					},
				},
			},
		},
		{
			name: "empty key",
			input: []slog.Attr{
				slog.String("", "empty-key"),
			},
			expected: map[string]any{
				"": "empty-key",
			},
		},
		{
			name: "nil and complex values",
			input: []slog.Attr{
				slog.Any("nil", nil),
				slog.Any("slice", []int{1, 2, 3}),
			},
			expected: map[string]any{
				"nil":   nil,
				"slice": []int{1, 2, 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AttrGroupToMap(tt.input)
			defer result.Free() // Ensure map and nested maps are freed

			got := result.Len()
			// Verify length
			testutils.Equal(t, len(tt.expected), got, "AttrGroupToMap(%v).Len() = %d; want %d", tt.input, got, len(tt.expected))

			// Verify each key-value pair
			for k, v := range tt.expected {
				got := result.Get(k)
				testutils.EqualAny(t, v, got, "AttrGroupToMap(%v).Get(%q) = %v; want %v", tt.input, k, got, v)
			}

			// Ensure no unexpected keys
			for k := range *result {
				if _, ok := tt.expected[k]; !ok {
					t.Errorf("AttrGroupToMap(%v) contains unexpected key %q", tt.input, k)
				}
			}
		})
	}
}

// Test deep nesting to ensure recursive handling and freeing
func TestAttrGroupToMap_DeepNesting(t *testing.T) {
	input := []slog.Attr{
		slog.Group("level1",
			slog.Group("level2",
				slog.Group("level3",
					slog.String("deep", "value"),
				),
			),
		),
	}
	expected := map[string]any{
		"level1": &AttrMap{
			"level2": &AttrMap{
				"level3": &AttrMap{
					"deep": "value",
				},
			},
		},
	}

	result := AttrGroupToMap(input)
	defer result.Free()

	testutils.Equal(t, len(expected), result.Len())

	for k, v := range expected {
		got := result.Get(k)
		testutils.EqualAny(t, v, got, "AttrGroupToMap(%v).Get(%q) = %v; want %v", input, k, got, v)
	}
}

// Test pool reuse and recursive Free behavior
func TestAttrGroupToMap_PoolReuse(t *testing.T) {
	// Test single-level map
	m1 := AttrGroupToMap([]slog.Attr{slog.String("key", "value")})
	if m1.Len() != 1 || m1.Get("key") != "value" {
		t.Errorf("AttrGroupToMap failed to set key-value")
	}

	m1.Free()
	testutils.Assert(t, m1.Len() == 0, "AttrMap.Free() did not reset map; Len() = %d", m1.Len())

	// Test nested map freeing
	m2 := AttrGroupToMap([]slog.Attr{
		slog.Group("nested",
			slog.String("inner", "value"),
		),
	})
	nested, ok := m2.Get("nested").(*AttrMap)
	if !ok || nested.Len() != 1 || nested.Get("inner") != "value" {
		t.Errorf("AttrGroupToMap failed to create nested AttrMap")
	}

	m2.Free()
	testutils.Assert(t, m2.Len() == 0, "AttrMap.Free() did not reset outer map; Len() = %d", m2.Len())
	testutils.Assert(t, nested.Len() == 0, "AttrMap.Free() did not reset nested map; Len() = %d", nested.Len())

	// Test oversized map (len > 32)
	m3 := AttrGroupToMap([]slog.Attr{})
	for i := range 33 { // Modern range syntax
		m3.Set(string(rune('a'+i)), i)
	}

	testutils.Assert(t, m3.Len() == 33, "AttrGroupToMap oversized setup failed; Len() = %d", m3.Len())
	m3.Free()
	testutils.Assert(t, m3.Len() == 0, "AttrMap.Free() did not reset oversized map; Len() = %d", m3.Len())
}

type failingWriter struct {
	shouldFail bool
}

func (f *failingWriter) Write(p []byte) (n int, err error) {
	if f.shouldFail {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}
