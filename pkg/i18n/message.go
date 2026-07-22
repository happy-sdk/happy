// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/happy-sdk/happy/pkg/i18n/schema/v1"
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
)

// Fragment kinds - see MessageFragment. Defined by package schema/v1 - the
// single source of truth for the on-disk message value shape (unchanged
// across every document schema version) - and re-exported here since
// they're part of this package's own public API.
const (
	// FragmentPlural selects Forms by CLDR cardinal plural category
	// (quantity: "1 item" vs "2 items") via golang.org/x/text/feature/plural's
	// Cardinal rules. The default kind when a fragment omits "kind".
	FragmentPlural = v1.FragmentPlural
	// FragmentOrdinal selects Forms by CLDR ordinal plural category
	// (rank: "1st", "2nd", "3rd", "4th") via Ordinal rules - the same
	// mechanism as FragmentPlural, different rule set.
	FragmentOrdinal = v1.FragmentOrdinal
	// FragmentSelect selects Forms by exact match against Arg's value
	// (formatted with %v) rather than any plural rule - e.g. selecting
	// wording by gender, formality, or any other closed set of literal
	// values. "other" is still the required fallback.
	FragmentSelect = v1.FragmentSelect
)

// messageKey mirrors v1.MessageKey for this package's own internal use
// (see v1.MessageKey for the full rationale for the "$"-prefixed sigil).
const messageKey = v1.MessageKey

// Message is a rich, translator-authorable translation entry. The
// {name}/{name:verb} minimal form (a plain string) covers most messages,
// but a message that needs correct grammatical variation by number
// (plural/ordinal) or by an arbitrary category (select) needs both a
// template and, for each varying piece, one wording per resolved category.
//
// A JSON translation value becomes a Message when it's an object holding
// a "$message" key, e.g.:
//
//	{
//	  "$message": {
//	    "msg": "{users_p} total {notifications_p}",
//	    "fragments": {
//	      "users_p": {
//	        "arg": "users",
//	        "one": "1 user has",
//	        "other": "{users:d} users have"
//	      },
//	      "notifications_p": {
//	        "arg": "unread_notifications",
//	        "one": "{unread_notifications:d} unread notification",
//	        "other": "{unread_notifications:d} unread notifications"
//	      }
//	    },
//	    "description": {
//	      "msg": "shown on the notifications page header",
//	      "args": {
//	        "users": {"description": "number of users"},
//	        "unread_notifications": {"description": "total unread notifications"}
//	      }
//	    }
//	  }
//	}
//
// Msg is rendered like any other named template, except a placeholder
// name matching a key in Fragments resolves to that fragment's own
// rendering (see MessageFragment) instead of a literal arg. A fragment
// omitting "kind" defaults to FragmentPlural - the common case.
//
// description.args never carries a "type": MessageArgDescription.Type is
// always derived from the verb the template actually uses for that arg
// (see MessageDescription.deriveArgTypes), not hand-authored - a
// hand-written type can drift out of sync with a template that later
// changes verb, silently documenting the wrong thing.
type Message struct {
	Msg         string
	Fragments   map[string]MessageFragment
	Description *MessageDescription
	// SourceHash is an optional, opaque fingerprint of the source-locale
	// content this translation was authored from (see schema/v1.Message.
	// SourceHash). It has no effect on rendering; it exists for future drift-
	// detection tooling and defaults to "" when a message declares none.
	SourceHash string
}

// MessageFragment is a single named, variable piece of a Message. Kind
// determines how a category is chosen (see FragmentPlural, FragmentOrdinal,
// FragmentSelect); Arg names which T/TL argument's value decides it. Forms
// holds one template per category - the CLDR category names ("zero",
// "one", "two", "few", "many", "other") for FragmentPlural/FragmentOrdinal,
// or arbitrary literal values for FragmentSelect. "other" is the one
// category every kind requires: for plural/ordinal it's the CLDR-mandated
// universal fallback (used whenever the language being rendered doesn't
// distinguish the category actually selected); for select it's the
// fallback for any value with no exact-match form.
type MessageFragment struct {
	Kind  string
	Arg   string
	Forms map[string]string
}

// MessageDescription documents a Message for translator/tooling use (e.g.
// a future l10n translate/list/report command) - it has no effect on
// rendering.
type MessageDescription struct {
	Msg  string
	Args map[string]MessageArgDescription
}

// MessageArgDescription documents a single named argument a Message
// expects. Type is never authored directly - see
// MessageDescription.deriveArgTypes - it's derived from the verb actually
// used for this arg in the message's own template, so the documented type
// can never silently drift out of sync with reality (e.g. a template
// changed from {x:s} to {x:q} without its description being updated to
// match). Description is free-text, written by whoever authors the
// message.
type MessageArgDescription struct {
	Type        ArgType
	Description string
}

// NewMessage starts building a Message with the given top-level template.
func NewMessage(msg string) *Message {
	return &Message{Msg: msg, Fragments: make(map[string]MessageFragment)}
}

// WithFragment adds (or replaces) a named FragmentPlural fragment: arg
// names the argument whose value selects forms' CLDR cardinal plural
// category. forms must include at least "other".
func (m *Message) WithFragment(name, arg string, forms map[string]string) *Message {
	return m.withFragment(name, FragmentPlural, arg, forms)
}

// WithOrdinalFragment adds (or replaces) a named FragmentOrdinal fragment:
// arg names the argument whose value selects forms' CLDR ordinal plural
// category (1st/2nd/3rd/4th-style). forms must include at least "other".
func (m *Message) WithOrdinalFragment(name, arg string, forms map[string]string) *Message {
	return m.withFragment(name, FragmentOrdinal, arg, forms)
}

// WithSelectFragment adds (or replaces) a named FragmentSelect fragment:
// arg names the argument whose value (formatted with %v) is matched
// exactly against forms' keys - e.g. for gender or formality. forms must
// include at least "other".
func (m *Message) WithSelectFragment(name, arg string, forms map[string]string) *Message {
	return m.withFragment(name, FragmentSelect, arg, forms)
}

func (m *Message) withFragment(name, kind, arg string, forms map[string]string) *Message {
	m.Fragments[name] = MessageFragment{Kind: kind, Arg: arg, Forms: forms}
	return m
}

// WithDescription attaches translator/tooling metadata to the message.
func (m *Message) WithDescription(msg string, args map[string]MessageArgDescription) *Message {
	m.Description = &MessageDescription{Msg: msg, Args: args}
	return m
}

// ArgTypes reports, for every literal argument placeholder in m's own
// template and its fragments' forms, the ArgType its verb resolves to -
// the same derivation deriveArgTypes uses for documented args, but for
// every argument the template actually interpolates, whether or not
// there's a Description for it at all. A translation UI can use this to
// show a translator what kind of value each placeholder holds - and
// ArgType.ExampleValue to show a sample - without needing any
// documentation to have been written.
//
// A fragment's own discriminating Arg (e.g. a FragmentSelect's "gender")
// is always included, even if it's never interpolated literally anywhere -
// see collectArgVerbs for the default type each fragment kind gets in that
// case. If the same name is also used literally, that usage's real verb
// takes precedence over the fragment-kind default.
func (m *Message) ArgTypes() map[string]ArgType {
	verbs := collectArgVerbs(m)
	types := make(map[string]ArgType, len(verbs))
	for name, verb := range verbs {
		types[name] = argTypeForVerb(verb)
	}
	return types
}

// String renders as the raw top-level template - used when a Message ends
// up somewhere expecting a plain string (e.g. a translation report or
// %v/%s formatting), since actually rendering it requires concrete
// argument values a formatter doesn't have.
func (m *Message) String() string {
	return m.Msg
}

// validate reports whether m is well-formed: every fragment has a known
// Kind, names its Arg, and defines at least the required "other" form; and
// every documented arg (if m.Description is set) actually appears
// somewhere in m's own templates.
func (m *Message) validate() error {
	for name, frag := range m.Fragments {
		if err := frag.validate(name); err != nil {
			return err
		}
	}
	if m.Description != nil {
		if err := m.Description.deriveArgTypes(m); err != nil {
			return err
		}
	}
	return nil
}

func (f MessageFragment) validate(name string) error {
	if f.Arg == "" {
		return ErrInvalidMessage.WithArgs("name", name, "reason", `fragment is missing its "arg" field naming its argument`)
	}
	switch f.Kind {
	case FragmentPlural, FragmentOrdinal, FragmentSelect:
	default:
		return ErrInvalidMessage.WithArgs("name", name, "reason", fmt.Sprintf(
			"unknown fragment kind %q (want %q, %q, or %q)", f.Kind, FragmentPlural, FragmentOrdinal, FragmentSelect))
	}
	if _, ok := f.Forms["other"]; !ok {
		return ErrInvalidMessage.WithArgs("name", name, "reason", `fragment is missing its required "other" form`)
	}
	return nil
}

// ArgType identifies the category of value a Message argument expects -
// the same vocabulary fmt itself uses to describe its verbs, so
// MessageArgDescription.Type reads like Go's own documentation rather
// than an ad hoc phrase. It is always derived from the verb actually used
// for that arg in the message's own template (see
// MessageDescription.deriveArgTypes) - never hand-authored, so it can
// never silently drift out of sync with a template that changes verb.
type ArgType int

const (
	// ArgTypeUnknown means the arg's verb isn't one this package assigns
	// a documentation category to - a detectable, reportable condition
	// (see deriveArgTypes), not a silent fallback to some arbitrary label.
	ArgTypeUnknown  ArgType = iota
	ArgTypeAny              // {name} or {name:v}: default format, like fmt.Sprint
	ArgTypeType             // {name:T}: Go type of the value
	ArgTypeString           // {name:s}: string representation
	ArgTypeDecimal          // {name:d}: base-10 integer
	ArgTypeFloat            // {name:f}: floating-point number
	ArgTypeTruth            // {name:t}: the word true or false
	ArgTypePointer          // {name:p}: memory address
	ArgTypeQuoted           // {name:q}: quoted string/rune
	ArgTypeHex              // {name:x}: hexadecimal
	ArgTypeNumber           // {name:number}: locale-aware formatted number
	ArgTypeCurrency         // {name:currency}: locale-aware formatted currency amount
)

// String implements fmt.Stringer, and is also ArgType's JSON-facing
// spelling (the word a future l10n tool would display).
func (t ArgType) String() string {
	switch t {
	case ArgTypeAny:
		return "any"
	case ArgTypeType:
		return "type"
	case ArgTypeString:
		return "string"
	case ArgTypeDecimal:
		return "decimal"
	case ArgTypeFloat:
		return "float"
	case ArgTypeTruth:
		return "truth"
	case ArgTypePointer:
		return "pointer"
	case ArgTypeQuoted:
		return "quoted"
	case ArgTypeHex:
		return "hex"
	case ArgTypeNumber:
		return "number"
	case ArgTypeCurrency:
		return "currency"
	default:
		return "unknown"
	}
}

// ExampleValue returns a short, human-readable example value for t - for
// translation tooling (e.g. a future l10n translate/list command) to show
// a translator what a {name:verb} placeholder will actually look like,
// without requiring any Go or programming knowledge. Returns "" for
// ArgTypeUnknown - there's nothing meaningful to show.
func (t ArgType) ExampleValue() string {
	switch t {
	case ArgTypeAny:
		return "example"
	case ArgTypeType:
		return "string"
	case ArgTypeString:
		return "example"
	case ArgTypeDecimal:
		return "42"
	case ArgTypeFloat:
		return "3.14"
	case ArgTypeTruth:
		return "true"
	case ArgTypePointer:
		return "0xc0000a4000"
	case ArgTypeQuoted:
		return `"example"`
	case ArgTypeHex:
		return "2a"
	case ArgTypeNumber:
		return "1,234.5"
	case ArgTypeCurrency:
		return "$1,234.00"
	default:
		return ""
	}
}

// verbArgTypes maps a formatNamed verb letter to its ArgType. "" (no verb
// at all, as in a bare {name}) means the same default formatting as %v.
var verbArgTypes = map[string]ArgType{
	"":         ArgTypeAny,
	"v":        ArgTypeAny,
	"T":        ArgTypeType,
	"s":        ArgTypeString,
	"d":        ArgTypeDecimal,
	"f":        ArgTypeFloat,
	"t":        ArgTypeTruth,
	"p":        ArgTypePointer,
	"q":        ArgTypeQuoted,
	"x":        ArgTypeHex,
	"number":   ArgTypeNumber,
	"currency": ArgTypeCurrency,
}

// argTypeForVerb returns verb's ArgType (see verbArgTypes), taking only
// the verb letter itself - a precision suffix like the ".2" in "f.2"
// doesn't change what kind of value is expected. An unrecognized letter
// (e.g. one of fmt's less common verbs this package hasn't catalogued,
// like %o or %c) reports ArgTypeUnknown rather than guessing.
func argTypeForVerb(verb string) ArgType {
	letter, _, _ := strings.Cut(verb, ".")
	if t, ok := verbArgTypes[letter]; ok {
		return t
	}
	return ArgTypeUnknown
}

// collectArgVerbs scans msg's own template and every fragment's forms for
// {name:verb} placeholders, returning the first verb seen for each literal
// argument name (a name that refers to one of msg's own Fragments is
// skipped - it's not itself an argument, whatever it resolves to is).
//
// It also assigns every fragment's own discriminating Arg a default verb -
// "d" (decimal) for FragmentPlural/FragmentOrdinal (the common case: an
// integer count), "" (any, matched via %v) for FragmentSelect - unless that
// name is already claimed by a literal placeholder elsewhere, in which case
// the literal usage's real verb wins. Without this, a fragment whose
// discriminator is only ever used to pick a category and never displayed
// (e.g. a "gender" select fragment, or a plural count that's never shown as
// text) was silently absent from ArgTypes() entirely: a translator couldn't
// even document what the discriminator meant (deriveArgTypes rejected it as
// "never used"), and addons/l10n's `l10n generate` silently omitted it from
// the generated accessor's parameters - producing a typed function that can
// never be given the value that selects its own wording, so the fragment
// could only ever render "other". Both are fixed by always accounting for
// every fragment's Arg here.
func collectArgVerbs(msg *Message) map[string]string {
	verbs := make(map[string]string)
	collect := func(s string) {
		for _, sub := range namedPlaceholderRe.FindAllStringSubmatch(s, -1) {
			name, verb := sub[1], strings.TrimPrefix(sub[2], ":")
			if _, isFragment := msg.Fragments[name]; isFragment {
				continue
			}
			if _, seen := verbs[name]; !seen {
				verbs[name] = verb
			}
		}
	}
	collect(msg.Msg)
	for _, frag := range msg.Fragments {
		for _, tmpl := range frag.Forms {
			collect(tmpl)
		}
	}
	for _, frag := range msg.Fragments {
		if frag.Arg == "" {
			continue
		}
		if _, seen := verbs[frag.Arg]; seen {
			continue
		}
		switch frag.Kind {
		case FragmentPlural, FragmentOrdinal:
			verbs[frag.Arg] = "d"
		default: // FragmentSelect
			verbs[frag.Arg] = ""
		}
	}
	return verbs
}

// deriveArgTypes fills each documented arg's Type from the verb actually
// used for it in msg's own template (see collectArgVerbs), overwriting
// whatever Type may already be set - Type is never meant to be authored by
// hand. Fails validation in either of two cases: an arg documented here
// that msg's template never actually references (stale or mistaken
// documentation), or one whose verb doesn't map to a known ArgType (see
// argTypeForVerb) - a value we don't know how to categorize is a
// misconfiguration to catch here, not something to silently paper over.
func (d *MessageDescription) deriveArgTypes(msg *Message) error {
	if d.Args == nil {
		return nil
	}
	verbs := collectArgVerbs(msg)
	for name, ad := range d.Args {
		verb, ok := verbs[name]
		if !ok {
			return ErrInvalidMessage.WithArgs("name", name, "reason", "description documents an arg the message template never uses")
		}
		argType := argTypeForVerb(verb)
		if argType == ArgTypeUnknown {
			return ErrInvalidMessage.WithArgs("name", name, "reason", fmt.Sprintf("verb %q has no known documentation type", verb))
		}
		ad.Type = argType
		d.Args[name] = ad
	}
	return nil
}

// pluralFormNames maps plural.Form (as returned by Cardinal/Ordinal's
// MatchPlural) back to its CLDR category name.
var pluralFormNames = map[plural.Form]string{
	plural.Zero:  "zero",
	plural.One:   "one",
	plural.Two:   "two",
	plural.Few:   "few",
	plural.Many:  "many",
	plural.Other: "other",
}

// parseMessageObject reports, for a JSON object v, whether it's a Message
// (isMessage - it holds a "$message" object) and, if so, parses it. An
// object with no "$message" key (isMessage == false, err == nil) is left
// for the caller to treat as an ordinary nested group of translation keys
// - the existing behavior for any map[string]any translation value.
//
// The actual structural parsing/validation of the "$message" object's
// contents lives in package schema/v1 - the single source of truth for the
// message value shape, unchanged across every document schema version -
// see v1.ParseMessage. This package only converts the result into its own
// runtime Message and layers on deriveArgTypes, which needs this package's
// own {name:verb} template parsing and isn't itself part of the value's
// shape.
func parseMessageObject(v map[string]any) (msg *Message, isMessage bool, err error) {
	raw, ok := v[v1.MessageKey]
	if !ok {
		return nil, false, nil
	}
	spec, ok := raw.(map[string]any)
	if !ok {
		return nil, true, ErrInvalidMessage.WithArgs("name", v1.MessageKey, "reason", "must be an object")
	}

	parsed, perr := v1.ParseMessage(spec)
	if perr != nil {
		return nil, true, messageValidationError(perr)
	}

	m := NewMessage(parsed.Msg)
	m.SourceHash = parsed.SourceHash
	for name, frag := range parsed.Fragments {
		m.Fragments[name] = MessageFragment{Kind: frag.Kind, Arg: frag.Arg, Forms: frag.Forms}
	}
	if parsed.Description != nil {
		desc := &MessageDescription{Msg: parsed.Description.Msg}
		if parsed.Description.Args != nil {
			desc.Args = make(map[string]MessageArgDescription, len(parsed.Description.Args))
			for name, ad := range parsed.Description.Args {
				desc.Args[name] = MessageArgDescription{Description: ad.Description}
			}
		}
		m.Description = desc
	}
	return m, true, m.validate()
}

// messageValidationError converts a schema/v1 structural error into this
// package's own (translatable) ErrInvalidMessage, preserving the offending
// name and reason so callers see the same detail either way.
func messageValidationError(err error) error {
	var verr *v1.ValidationError
	if errors.As(err, &verr) {
		return ErrInvalidMessage.WithArgs("name", verr.Name, "reason", verr.Reason)
	}
	return ErrInvalidMessage.WithArgs("name", v1.MessageKey, "reason", err.Error())
}

// renderFragment selects frag's category for lang - per frag.Kind, based
// on the value of the arg frag.Arg names - and renders that category's
// template against the same values every other placeholder in the
// message resolves against.
func renderFragment(lang language.Tag, frag MessageFragment, values map[string]any) string {
	argVal, ok := values[frag.Arg]
	if !ok {
		return fmt.Sprintf("%%!%s(MISSING)", frag.Arg)
	}

	var category string
	switch frag.Kind {
	case FragmentOrdinal:
		category = cldrCategory(plural.Ordinal, lang, argVal)
	case FragmentSelect:
		category = fmt.Sprintf("%v", argVal)
	default: // FragmentPlural
		category = cldrCategory(plural.Cardinal, lang, argVal)
	}

	tmpl, ok := frag.Forms[category]
	if !ok {
		// Not every language distinguishes every CLDR category, and a
		// select's value may simply have no dedicated wording; "other" is
		// the one category every kind and every fragment defines.
		tmpl, ok = frag.Forms["other"]
		if !ok {
			return fmt.Sprintf("%%!FORM(%s)", frag.Arg)
		}
	}
	return renderTemplate(tmpl, values, nil)
}

// cldrCategory returns count's CLDR category name for lang under rules
// (plural.Cardinal or plural.Ordinal). A count that isn't a recognizable
// number falls back to "other" - the universal category every language
// and fragment defines - rather than failing to render.
func cldrCategory(rules *plural.Rules, lang language.Tag, count any) string {
	return cldrCategoryRules(rules, lang, count)
}

// cldrCategoryRules is the shared implementation behind both the legacy
// renderFragment path and the compiled fragProgram path, so the two can never
// diverge on plural-category selection. It extracts the full CLDR operand set
// (i, v, w, f, t) - not just the integer part - so a fractional count selects
// the correct category in locales whose rules depend on visible fraction
// digits (e.g. English "1.5 items" is "other", not "one").
func cldrCategoryRules(rules *plural.Rules, lang language.Tag, count any) string {
	i, v, w, f, t, ok := pluralOperands(count)
	if !ok {
		return "other"
	}
	form := rules.MatchPlural(lang, i, v, w, f, t)
	if name, ok := pluralFormNames[form]; ok {
		return name
	}
	return "other"
}

// toPluralOperand extracts the CLDR "i" operand (absolute integer value) from a
// count argument of any of the ordinary Go numeric types. It is retained for
// callers that only need the integer part; the full operand set (used for
// correct plural-category selection) is computed by pluralOperands.
func toPluralOperand(value any) (int, bool) {
	i, _, _, _, _, ok := pluralOperands(value)
	return i, ok
}

// pluralOperands computes the CLDR plural operands MatchPlural needs from a
// count of any ordinary Go numeric type:
//
//	i  integer part (absolute value)
//	v  number of visible fraction digits
//	w  number of visible fraction digits without trailing zeros
//	f  visible fraction digits as an integer
//	t  visible fraction digits without trailing zeros, as an integer
//
// Integer types have v = w = f = t = 0. Floats derive their fraction operands
// from the value's shortest exact decimal representation, so e.g. 1.0 (which is
// indistinguishable from the int 1 at runtime) yields no fraction digits while
// 1.50 yields v = 1, f = 5.
func pluralOperands(value any) (i, v, w, f, t int, ok bool) {
	switch n := value.(type) {
	case int:
		return absInt(n), 0, 0, 0, 0, true
	case int8:
		return absInt(int(n)), 0, 0, 0, 0, true
	case int16:
		return absInt(int(n)), 0, 0, 0, 0, true
	case int32:
		return absInt(int(n)), 0, 0, 0, 0, true
	case int64:
		return absInt(int(n)), 0, 0, 0, 0, true
	case uint:
		return int(n), 0, 0, 0, 0, true
	case uint8:
		return int(n), 0, 0, 0, 0, true
	case uint16:
		return int(n), 0, 0, 0, 0, true
	case uint32:
		return int(n), 0, 0, 0, 0, true
	case uint64:
		return int(n), 0, 0, 0, 0, true
	case float32:
		return floatOperands(float64(n))
	case float64:
		return floatOperands(n)
	default:
		return 0, 0, 0, 0, 0, false
	}
}

// floatOperands derives the CLDR operands from a floating-point count using its
// shortest exact decimal form (strconv 'f', -1), so no spurious precision is
// invented.
func floatOperands(x float64) (i, v, w, f, t int, ok bool) {
	if x < 0 {
		x = -x
	}
	s := strconv.FormatFloat(x, 'f', -1, 64)
	intPart, fracPart, hasFrac := strings.Cut(s, ".")
	iv, _ := strconv.Atoi(intPart)
	if !hasFrac || fracPart == "" {
		return iv, 0, 0, 0, 0, true
	}
	trimmed := strings.TrimRight(fracPart, "0")
	fv, _ := strconv.Atoi(fracPart)
	tv := 0
	if trimmed != "" {
		tv, _ = strconv.Atoi(trimmed)
	}
	return iv, len(fracPart), len(trimmed), fv, tv, true
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
