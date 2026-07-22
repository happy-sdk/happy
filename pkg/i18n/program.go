// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/text/currency"
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// This file holds the compiled, render-time representation of a translation
// template. The design goal is: parse/analyze a template's structure exactly
// once, at registration time, into an immutable *program, and never re-parse
// (regex, format-string building, map allocation) on the hot render path. A
// *program is safe for concurrent reads with no locking - it is only ever
// constructed while the registration mutex is held, then published inside an
// immutable snapshot (see catalog.go).

// verbKind classifies a placeholder's formatting verb into the small set that
// gets a bespoke, allocation-free fast path in formatInto. Anything not worth
// a bespoke path (verbFloat, verbQuoted, ...) resolves through fmt.Appendf via
// the precompiled fmt spec, which is guaranteed identical to the old
// fmt.Sprintf-per-placeholder behaviour it replaces.
type verbKind uint8

const (
	verbAny      verbKind = iota // {name} or {name:v}
	verbString                   // {name:s}
	verbDecimal                  // {name:d}
	verbNumber                   // {name:number}   - locale-aware (x/text/number)
	verbCurrency                 // {name:currency} - locale-aware (x/text/currency)
	verbOther                    // every other fmt verb, via verb.spec + fmt.Appendf
)

// verb is a placeholder's precompiled formatting descriptor. spec is the
// equivalent fmt verb string (e.g. "%d", "%.2f", "%q") built once at compile
// time; kind selects a fast path in formatInto.
type verb struct {
	kind verbKind
	spec string
}

// seg is one compiled segment of a template. Exactly one role is active:
//   - a literal run: name == "" && frag == nil, emit lit verbatim;
//   - a fragment: frag != nil, render the plural/ordinal/select fragment;
//   - an interpolation: name != "", format the named arg per verb.
type seg struct {
	lit  string
	name string
	verb verb
	frag *fragProgram
}

// argSpec is a precompiled record of one literal argument a template
// interpolates and the ArgType its verb resolves to. Computed once at compile
// time so Message.ArgTypes()/codegen never need to re-scan the template.
type argSpec struct {
	Name string
	Type ArgType
}

// program is the compiled form of a single translation template for a single
// (language, key). Exactly one render mode is active:
//   - legacy != "": a legacy %-verb positional template, rendered with the raw
//     variadic args via the language's message.Printer (preserving the prior
//     golang.org/x/text/message positional behaviour exactly);
//   - catalogKey != "": a value registered as an x/text catalog.Message, rendered
//     through the language's printer by key (legacy compatibility only);
//   - otherwise: a named/{fragment} template rendered from segs.
type program struct {
	segs       []seg
	args       []argSpec
	src        string // original template text, for reporting/introspection
	legacy     string
	catalogKey string
}

// isLiteral reports whether p is a single literal run with no placeholders -
// the overwhelmingly common case for plain strings - which renders with zero
// buffer allocation and zero copying.
func (p *program) isLiteral() bool {
	return p.legacy == "" && p.catalogKey == "" &&
		len(p.segs) == 1 && p.segs[0].name == "" && p.segs[0].frag == nil
}

// fragProgram is a compiled plural/ordinal/select fragment. rules is resolved
// once at compile time (plural.Cardinal/Ordinal) for plural/ordinal fragments
// and is nil for select fragments; forms holds each category's own compiled
// template.
type fragProgram struct {
	kind  string
	arg   string
	rules *plural.Rules
	forms map[string]*program
}

// namedPlaceholderRe indices helper: the regex has two capture groups -
// group 1 (the name) and the optional group 2 (":verb").

// compileString compiles a plain-string translation template. A string that
// uses this package's {name}/{name:verb} syntax (or no placeholders at all)
// becomes a segment program; a string that uses only legacy %-verbs becomes a
// positional legacy program.
func compileString(s string) *program {
	if !hasNamedPlaceholders(s) && hasFmtVerb(s) {
		return &program{legacy: s, src: s}
	}
	segs, args := compileSegs(s)
	return &program{segs: segs, args: args, src: s}
}

// compileMessage compiles a rich *Message: its Msg is compiled like any named
// template, except a placeholder whose name matches one of the message's own
// fragments becomes a fragment segment resolved recursively.
func compileMessage(m *Message) *program {
	segs, args := compileMessageSegs(m.Msg, m)
	return &program{segs: segs, args: args, src: m.Msg}
}

// compileSegs splits s into literal and interpolation segments using the
// shared named-placeholder regex, exactly once.
func compileSegs(s string) ([]seg, []argSpec) {
	locs := namedPlaceholderRe.FindAllStringSubmatchIndex(s, -1)
	if len(locs) == 0 {
		return []seg{{lit: s}}, nil
	}
	segs := make([]seg, 0, len(locs)*2+1)
	var args []argSpec
	last := 0
	for _, loc := range locs {
		start, end := loc[0], loc[1]
		if start > last {
			segs = append(segs, seg{lit: s[last:start]})
		}
		name := s[loc[2]:loc[3]]
		verbStr := ""
		if loc[4] >= 0 {
			verbStr = s[loc[4]+1 : loc[5]] // skip the leading ':'
		}
		v := compileVerb(verbStr)
		segs = append(segs, seg{name: name, verb: v})
		args = append(args, argSpec{Name: name, Type: argTypeForVerb(verbStr)})
		last = end
	}
	if last < len(s) {
		segs = append(segs, seg{lit: s[last:]})
	}
	return segs, args
}

// compileMessageSegs is compileSegs, but a placeholder naming one of msg's
// fragments compiles to a fragment segment rather than a literal arg.
func compileMessageSegs(s string, msg *Message) ([]seg, []argSpec) {
	locs := namedPlaceholderRe.FindAllStringSubmatchIndex(s, -1)
	if len(locs) == 0 {
		return []seg{{lit: s}}, nil
	}
	segs := make([]seg, 0, len(locs)*2+1)
	var args []argSpec
	last := 0
	for _, loc := range locs {
		start, end := loc[0], loc[1]
		if start > last {
			segs = append(segs, seg{lit: s[last:start]})
		}
		name := s[loc[2]:loc[3]]
		if frag, ok := msg.Fragments[name]; ok {
			segs = append(segs, seg{name: name, frag: compileFragment(frag)})
			last = end
			continue
		}
		verbStr := ""
		if loc[4] >= 0 {
			verbStr = s[loc[4]+1 : loc[5]]
		}
		segs = append(segs, seg{name: name, verb: compileVerb(verbStr)})
		args = append(args, argSpec{Name: name, Type: argTypeForVerb(verbStr)})
		last = end
	}
	if last < len(s) {
		segs = append(segs, seg{lit: s[last:]})
	}
	return segs, args
}

// compileFragment resolves a fragment's plural rule set once and compiles each
// of its category forms into its own program.
func compileFragment(f MessageFragment) *fragProgram {
	fp := &fragProgram{kind: f.Kind, arg: f.Arg, forms: make(map[string]*program, len(f.Forms))}
	switch f.Kind {
	case FragmentOrdinal:
		fp.rules = plural.Ordinal
	case FragmentSelect:
		fp.rules = nil
	default: // FragmentPlural
		fp.rules = plural.Cardinal
	}
	for cat, tmpl := range f.Forms {
		segs, args := compileSegs(tmpl)
		fp.forms[cat] = &program{segs: segs, args: args, src: tmpl}
	}
	return fp
}

// compileVerb turns a {name:verb} verb string (e.g. "s", "d", "f.2", "number")
// into a precompiled verb descriptor: a fast-path kind plus the equivalent fmt
// spec used for everything without a bespoke path.
func compileVerb(spec string) verb {
	letter, prec, hasPrec := strings.Cut(spec, ".")
	switch letter {
	case "number":
		return verb{kind: verbNumber, spec: "%v"}
	case "currency":
		return verb{kind: verbCurrency, spec: "%v"}
	}

	kind := verbOther
	fmtLetter := letter
	switch letter {
	case "", "v":
		kind = verbAny
		fmtLetter = "v"
	case "s":
		kind = verbString
	case "d":
		kind = verbDecimal
	}

	var sb strings.Builder
	sb.Grow(len(spec) + 2)
	sb.WriteByte('%')
	if hasPrec {
		sb.WriteByte('.')
		sb.WriteString(prec)
	}
	sb.WriteString(fmtLetter)
	return verb{kind: kind, spec: sb.String()}
}

// hasFmtVerb reports whether s contains at least one classic fmt %-verb (as
// opposed to a bare or doubled '%'). Only strings that use this are treated as
// legacy positional templates.
func hasFmtVerb(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		j := i + 1
		if j >= len(s) {
			return false
		}
		if s[j] == '%' { // escaped percent
			i = j
			continue
		}
		// skip flags/width/precision until a letter (the verb)
		for j < len(s) {
			c := s[j]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				return true
			}
			if (c >= '0' && c <= '9') || c == '.' || c == '+' || c == '-' || c == ' ' || c == '#' {
				j++
				continue
			}
			break
		}
	}
	return false
}

// --- rendering ---------------------------------------------------------------

// bufPool recycles the []byte scratch buffers interpolated renders build into.
var bufPool = sync.Pool{New: func() any { b := make([]byte, 0, 128); return &b }}

// renderOpts carries per-call rendering options (see Option), kept off the
// argument list so it never collides with a translation argument's value.
type renderOpts struct {
	bidiIsolation bool
}

// render produces p's output for lang. positional is the raw variadic arg list
// (used only for legacy/catalog modes); named args are read from a. prn is the
// language's precompiled printer (from the immutable snapshot) used for
// legacy/catalog/number/currency formatting.
func (p *program) render(lang language.Tag, a *argList, positional []any, prn *message.Printer, opts renderOpts) string {
	if p.legacy != "" {
		if prn != nil {
			return prn.Sprintf(p.legacy, positional...)
		}
		return fmt.Sprintf(p.legacy, positional...)
	}
	if p.catalogKey != "" {
		if prn != nil {
			return prn.Sprintf(p.catalogKey)
		}
		return p.catalogKey
	}
	if p.isLiteral() {
		return p.segs[0].lit
	}
	buf := getBuf()
	p.appendTo(buf, lang, a, prn, opts)
	out := string(*buf)
	*buf = (*buf)[:0]
	bufPool.Put(buf)
	return out
}

func (p *program) appendTo(dst *[]byte, lang language.Tag, a *argList, prn *message.Printer, opts renderOpts) {
	rtl := opts.bidiIsolation && isRTL(lang)
	for i := range p.segs {
		s := &p.segs[i]
		switch {
		case s.frag != nil:
			s.frag.appendTo(dst, lang, a, prn, opts)
		case s.name == "":
			*dst = append(*dst, s.lit...)
		default:
			val, ok := a.lookup(s.name)
			if !ok {
				*dst = append(*dst, "%!"...)
				*dst = append(*dst, s.name...)
				*dst = append(*dst, "(MISSING)"...)
				continue
			}
			if rtl {
				*dst = append(*dst, fsi...)
				formatInto(dst, lang, s.verb, val, prn)
				*dst = append(*dst, pdi...)
			} else {
				formatInto(dst, lang, s.verb, val, prn)
			}
		}
	}
}

func (f *fragProgram) appendTo(dst *[]byte, lang language.Tag, a *argList, prn *message.Printer, opts renderOpts) {
	argVal, ok := a.lookup(f.arg)
	if !ok {
		*dst = append(*dst, "%!"...)
		*dst = append(*dst, f.arg...)
		*dst = append(*dst, "(MISSING)"...)
		return
	}

	var category string
	if f.kind == FragmentSelect {
		category = fmt.Sprintf("%v", argVal)
	} else {
		category = cldrCategoryRules(f.rules, lang, argVal)
	}

	prog, ok := f.forms[category]
	if !ok {
		prog, ok = f.forms["other"]
		if !ok {
			*dst = append(*dst, "%!FORM("...)
			*dst = append(*dst, f.arg...)
			*dst = append(*dst, ')')
			return
		}
	}
	prog.appendTo(dst, lang, a, prn, opts)
}

// formatInto appends val formatted per v to dst. The common verbs get direct,
// allocation-free strconv/append paths; everything else falls through to
// fmt.Appendf using the precompiled spec, which is byte-for-byte identical to
// the fmt.Sprintf the old per-placeholder renderer used.
func formatInto(dst *[]byte, lang language.Tag, v verb, val any, prn *message.Printer) {
	switch v.kind {
	case verbAny, verbString:
		if s, ok := val.(string); ok {
			*dst = append(*dst, s...)
			return
		}
	case verbDecimal:
		switch n := val.(type) {
		case int:
			*dst = strconv.AppendInt(*dst, int64(n), 10)
			return
		case int64:
			*dst = strconv.AppendInt(*dst, n, 10)
			return
		case int32:
			*dst = strconv.AppendInt(*dst, int64(n), 10)
			return
		case int16:
			*dst = strconv.AppendInt(*dst, int64(n), 10)
			return
		case int8:
			*dst = strconv.AppendInt(*dst, int64(n), 10)
			return
		case uint:
			*dst = strconv.AppendUint(*dst, uint64(n), 10)
			return
		case uint64:
			*dst = strconv.AppendUint(*dst, n, 10)
			return
		case uint32:
			*dst = strconv.AppendUint(*dst, uint64(n), 10)
			return
		case uint16:
			*dst = strconv.AppendUint(*dst, uint64(n), 10)
			return
		case uint8:
			*dst = strconv.AppendUint(*dst, uint64(n), 10)
			return
		}
	case verbNumber:
		if prn != nil {
			*dst = append(*dst, prn.Sprintf("%v", number.Decimal(val))...)
			return
		}
	case verbCurrency:
		if prn != nil {
			if amt, ok := val.(currency.Amount); ok {
				*dst = append(*dst, prn.Sprintf("%v", currency.Symbol(amt))...)
				return
			}
		}
		// Unlike every other verb here, falling through to the final
		// fmt.Appendf(dst, "%v", val) below would silently render a bare
		// number/string with no currency symbol or grouping at all - no
		// "%!" marker, nothing indicating the format failed. That's worse
		// than wrong output: it looks like a plausible price. Emit fmt's
		// own bad-argument convention (%!verb(type=value), e.g.
		// %!d(string=x)) instead, consistent with how every other verb
		// mismatch in this package is already surfaced.
		*dst = fmt.Appendf(*dst, "%%!currency(%T=%v)", val, val)
		return
	}
	*dst = fmt.Appendf(*dst, v.spec, val)
}

func getBuf() *[]byte { return bufPool.Get().(*[]byte) }

// --- argument list -----------------------------------------------------------

// argList is a small, stack-friendly replacement for the per-call
// map[string]any the old renderer built. Real templates interpolate a handful
// of args, so a fixed inline array plus linear scan is both faster and
// allocation-free versus a map; an overflow slice covers the rare large case.
type argList struct {
	arr [8]Arg
	n   int
	ext []Arg
}

// buildArgList normalizes a slog-style variadic argument list - bare
// "key", value pairs interleaved with typed Arg values - into an argList,
// preserving argsToMap's exact malformed-input markers (%!(NOVALUE) for a
// trailing key, !BADARGn for a value where a key was expected) and its
// last-value-wins behaviour for a repeated key.
func buildArgList(args []any) argList {
	var a argList
	for i := 0; i < len(args); i++ {
		switch v := args[i].(type) {
		case Arg:
			a.set(v.Key, v.Value)
		case string:
			if i+1 < len(args) {
				a.set(v, args[i+1])
				i++
			} else {
				a.set(v, "%!(NOVALUE)")
			}
		default:
			a.set(fmt.Sprintf("!BADARG%d", i), v)
		}
	}
	return a
}

func (a *argList) set(key string, val any) {
	for i := 0; i < a.n; i++ {
		if a.arr[i].Key == key {
			a.arr[i].Value = val
			return
		}
	}
	for i := range a.ext {
		if a.ext[i].Key == key {
			a.ext[i].Value = val
			return
		}
	}
	if a.n < len(a.arr) {
		a.arr[a.n] = Arg{Key: key, Value: val}
		a.n++
		return
	}
	a.ext = append(a.ext, Arg{Key: key, Value: val})
}

func (a *argList) lookup(key string) (any, bool) {
	for i := 0; i < a.n; i++ {
		if a.arr[i].Key == key {
			return a.arr[i].Value, true
		}
	}
	for i := range a.ext {
		if a.ext[i].Key == key {
			return a.ext[i].Value, true
		}
	}
	return nil, false
}
