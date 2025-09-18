// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package adapters

import (
	"encoding/json"
	"log/slog"
	"testing"
)

func BenchmarkAttrMapPool(b *testing.B) {
	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
		slog.Bool("key3", true),
		slog.String("key4", "value4"),
	}

	b.Run("Pooled", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			m := NewAttrMap()
			for _, attr := range attrs {
				m.Set(attr.Key, attr.Value.Any())
			}
			_, err := m.MarshalJSON()
			if err != nil {
				b.Fatal(err)
			}
			m.Free()
		}
	})

	b.Run("NewMap", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			m := make(map[string]any, 10)
			for _, attr := range attrs {
				m[attr.Key] = attr.Value.Any()
			}
			_, err := json.Marshal(m)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Benchmark for performance
func BenchmarkAttrGroupToMap(b *testing.B) {
	input := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Group("group1",
			slog.Int("count", 42),
			slog.Group("group2",
				slog.Bool("enabled", true),
			),
		),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := AttrGroupToMap(input)
		m.Free()
	}
}
