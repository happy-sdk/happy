// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"testing"

	"golang.org/x/text/language"
)

// These benchmarks exercise the hot render path (T/TL) that the rewrite made
// lock-free and precompiled. BenchmarkT_Parallel in particular demonstrates the
// effect of removing the read-side mutex: it scales across goroutines because a
// render only does an atomic snapshot load and immutable reads.

func benchInit(b *testing.B) {
	b.Helper()
	Initialize(language.English)
}

func BenchmarkT_PlainString(b *testing.B) {
	benchInit(b)
	_ = RegisterTranslation(language.English, "bench.plain", "a plain untemplated translation string")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = T("bench.plain")
	}
}

func BenchmarkT_Interpolated(b *testing.B) {
	benchInit(b)
	_ = RegisterTranslation(language.English, "bench.interp", "hello {name:s}, you have {count:d} new messages in {folder:s}")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = T("bench.interp", "name", "Ada", "count", 7, "folder", "Inbox")
	}
}

func BenchmarkT_Plural(b *testing.B) {
	benchInit(b)
	_ = RegisterTranslation(language.English, "bench.plural", map[string]any{
		messageKey: map[string]any{
			"msg": "{items_p}",
			"fragments": map[string]any{
				"items_p": map[string]any{
					"arg":   "n",
					"one":   "1 item",
					"other": "{n:d} items",
				},
			},
		},
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = T("bench.plural", "n", 5)
	}
}

// BenchmarkOldEngine_Interpolated isolates the previous render engine (regex
// ReplaceAllStringFunc + per-call map[string]any + fmt.Sprintf per placeholder),
// still reachable via renderNamed, rendering the same template as
// BenchmarkNewEngine_Interpolated for a direct before/after comparison of just
// the rendering work (no snapshot lookup on either side).
func BenchmarkOldEngine_Interpolated(b *testing.B) {
	const tmpl = "hello {name:s}, you have {count:d} new messages in {folder:s}"
	args := []any{"name", "Ada", "count", 7, "folder", "Inbox"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderNamed(tmpl, args)
	}
}

// BenchmarkNewEngine_Interpolated renders the same template through the
// precompiled program, compiled once outside the timed loop.
func BenchmarkNewEngine_Interpolated(b *testing.B) {
	prog := compileString("hello {name:s}, you have {count:d} new messages in {folder:s}")
	args := []any{"name", "Ada", "count", 7, "folder", "Inbox"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		al := buildArgList(args)
		_ = prog.render(language.English, &al, args, nil, renderOpts{})
	}
}

func BenchmarkT_Parallel(b *testing.B) {
	benchInit(b)
	_ = RegisterTranslation(language.English, "bench.parallel", "hello {name:s}, you have {count:d} new messages")
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = T("bench.parallel", "name", "Ada", "count", 7)
		}
	})
}
