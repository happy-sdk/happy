// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"sync/atomic"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// This file holds the immutable, lock-free read side of the translation store.
//
// The hot path (T/TL) reads a *snapshot published via an atomic.Pointer and
// never takes a mutex. Registration is copy-on-write and rare/bursty (init
// time): each individual registration call takes the manager's mutex only
// around its own catalog/bookkeeping mutation (see registerTranslation), and
// reload() serializes the actual snapshot rebuild under the same mutex before
// atomically swapping the result in. A render in progress keeps reading the
// old snapshot; the swap is a single
// pointer store.

// bundleMeta records a bundle's authoritative metadata, read straight from a
// schema/v2 Document rather than guessed. source is the bundle's declared
// source locale (schema/v2.KeySource); notes holds per-locale translator notes.
type bundleMeta struct {
	source language.Tag
	notes  map[language.Tag]string
}

// snapshot is one immutable view of every registered translation, compiled and
// ready to render. It is never mutated after publication.
type snapshot struct {
	byLang    map[language.Tag]map[string]*program
	bundles   map[string]bundleMeta
	keyBundle map[string]string // full key -> owning bundle (authoritative for v2)
	langs     []language.Tag
	fallback  language.Tag
	matcher   language.Matcher
	printers  map[language.Tag]*message.Printer
}

// active is the currently published snapshot. nil until Initialize runs.
var active atomic.Pointer[snapshot]

// currentLang is the current language, kept separate from the snapshot so
// SetLanguage is a single atomic store rather than a full snapshot rebuild.
var currentLang atomic.Pointer[language.Tag]

func loadSnapshot() *snapshot { return active.Load() }

func storeSnapshot(s *snapshot) { active.Store(s) }

func loadCurrentLang() language.Tag {
	if p := currentLang.Load(); p != nil {
		return *p
	}
	return language.English
}

func storeCurrentLang(lang language.Tag) {
	l := lang
	currentLang.Store(&l)
}

// printerFor returns the precompiled printer for lang, if the snapshot has one.
func (s *snapshot) printerFor(lang language.Tag) *message.Printer {
	if s == nil {
		return nil
	}
	return s.printers[lang]
}

// lookup resolves key for lang, cascading through a real BCP 47 fallback chain:
//
//	exact lang -> lang.Parent() chain (e.g. zh-Hant -> zh) ->
//	the owning bundle's declared source locale -> the global fallback.
//
// It returns the compiled program and the language it was actually found in
// (its wording's language, which - not the requested language - decides plural
// category), plus an explicit ok. "Not found" is never inferred from string
// equality; only T/TL's outermost layer turns !ok into the raw key.
func (s *snapshot) resolve(lang language.Tag, key string) (*program, language.Tag, bool) {
	if s == nil {
		return nil, language.Und, false
	}

	if p := s.at(lang, key); p != nil {
		return p, lang, true
	}

	// Real BCP 47 parent chain: zh-Hant-TW -> zh-Hant -> zh.
	for l := lang; ; {
		parent := l.Parent()
		if parent == l || parent == language.Und {
			break
		}
		if p := s.at(parent, key); p != nil {
			return p, parent, true
		}
		l = parent
	}

	// The owning bundle's declared source locale.
	if bundle, ok := s.keyBundle[key]; ok {
		if meta, ok := s.bundles[bundle]; ok {
			src := meta.source
			if src != language.Und && src != lang {
				if p := s.at(src, key); p != nil {
					return p, src, true
				}
			}
		}
	}

	// The global fallback language.
	if s.fallback != lang {
		if p := s.at(s.fallback, key); p != nil {
			return p, s.fallback, true
		}
	}

	return nil, language.Und, false
}

func (s *snapshot) at(lang language.Tag, key string) *program {
	if m := s.byLang[lang]; m != nil {
		return m[key]
	}
	return nil
}

// --- rendering entry points (hot path) --------------------------------------

// translate is the shared implementation behind T/TL: resolve key for lang
// through the immutable snapshot and render, with no locking on the read path.
func translate(lang language.Tag, key string, args []any) string {
	s := loadSnapshot()
	if s == nil {
		return key
	}
	opts, rest := splitOptions(args)
	prog, foundLang, ok := s.resolve(lang, key)
	if !ok {
		return key
	}
	prn := s.printerFor(foundLang)
	if prog.legacy != "" || prog.catalogKey != "" {
		return prog.render(foundLang, nil, rest, prn, opts)
	}
	al := buildArgList(rest)
	return prog.render(foundLang, &al, rest, prn, opts)
}

// --- render options ----------------------------------------------------------

// Option is a per-call rendering option passed inline in T/TL's variadic
// arguments (e.g. i18n.TL(lang, key, args..., i18n.WithBidiIsolation())). It is
// a distinct interface type so it can never be mistaken for a translation
// argument value the way a bare Arg or slog pair could.
type Option interface {
	apply(*renderOpts)
}

type optionFunc func(*renderOpts)

func (f optionFunc) apply(o *renderOpts) { f(o) }

// WithBidiIsolation wraps every interpolated (not literal) segment in Unicode
// FSI/PDI isolation marks when rendering for a right-to-left locale, so an
// LTR-looking argument value (a number, a URL, a name) embedded in RTL text
// can't visually reorder the surrounding wording. It is strictly opt-in and
// never applied by default anywhere, including the HTTP middleware.
func WithBidiIsolation() Option {
	return optionFunc(func(o *renderOpts) { o.bidiIsolation = true })
}

// splitOptions separates any Option values out of a variadic arg list. The
// common case (no options) returns args untouched with no allocation.
func splitOptions(args []any) (renderOpts, []any) {
	has := false
	for _, a := range args {
		if _, ok := a.(Option); ok {
			has = true
			break
		}
	}
	if !has {
		return renderOpts{}, args
	}
	var opts renderOpts
	rest := make([]any, 0, len(args))
	for _, a := range args {
		if o, ok := a.(Option); ok {
			o.apply(&opts)
			continue
		}
		rest = append(rest, a)
	}
	return opts, rest
}

// Unicode bidirectional isolation marks.
const (
	fsi = "⁨" // FIRST STRONG ISOLATE
	pdi = "⁩" // POP DIRECTIONAL ISOLATE
)

// rtlBases lists the base languages whose scripts are written right-to-left.
// golang.org/x/text exposes no cheap per-Tag RTL predicate, and none of this
// package's own supported locales are RTL, so this is a small, explicit set
// used only by the opt-in WithBidiIsolation path.
var rtlBases = map[string]bool{
	"ar": true, // Arabic
	"he": true, // Hebrew
	"fa": true, // Persian
	"ur": true, // Urdu
	"ps": true, // Pashto
	"sd": true, // Sindhi
	"yi": true, // Yiddish
	"dv": true, // Divehi
}

func isRTL(lang language.Tag) bool {
	base, _ := lang.Base()
	return rtlBases[base.String()]
}
