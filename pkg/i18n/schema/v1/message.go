// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package v1

import "fmt"

// Fragment kinds recognized by a "$message" fragment's "kind" field - see
// Fragment.
const (
	// FragmentPlural selects a fragment's rendered form by CLDR cardinal
	// plural category (quantity: "1 item" vs "2 items"). The default kind
	// when a fragment omits "kind".
	FragmentPlural = "plural"
	// FragmentOrdinal selects a fragment's rendered form by CLDR ordinal
	// plural category (rank: "1st", "2nd", "3rd", "4th").
	FragmentOrdinal = "ordinal"
	// FragmentSelect selects a fragment's rendered form by exact match
	// against its arg's value, rather than any plural rule - e.g.
	// selecting wording by gender or formality. "other" is still the
	// required fallback.
	FragmentSelect = "select"
)

// MessageKey is the reserved key that marks a JSON object translation
// value as a structured Message (see package i18n's own Message) rather
// than an ordinary nested group of translation keys. It's a "$"-prefixed
// sigil - the same convention JSON Schema ("$schema", "$ref") and this
// package's sibling schema packages use - specifically so it can never
// collide with a real translation key: nothing a translator would ever
// legitimately name a message segment starts with "$".
const MessageKey = "$message"

// CLDRCategories are the only form keys FragmentPlural/FragmentOrdinal
// accept - catching a translator typo (e.g. "singular" instead of "one")
// as a validation error instead of a silently-dead form that only "other"
// ever renders. FragmentSelect forms are matched against arbitrary literal
// values instead, so they aren't restricted to this set.
var CLDRCategories = map[string]bool{
	"zero": true, "one": true, "two": true, "few": true, "many": true, "other": true,
}

// Message is the structurally-validated shape of a "$message" object's
// value.
type Message struct {
	Msg         string
	Fragments   map[string]Fragment
	Description *Description
	// SourceHash is an optional, opaque fingerprint of the source-locale
	// content this translation was made from, carried verbatim if present.
	// It lets tooling detect when a translation has gone stale relative to
	// its source text (drift detection) without changing rendering at all -
	// old readers ignore it, and it defaults to "" when absent. Nothing in
	// this package consumes it yet; it exists so a later tooling pass can.
	SourceHash string
}

// Fragment is a single named, variable piece of a Message - see package
// i18n's MessageFragment, which this mirrors field for field.
type Fragment struct {
	Kind  string
	Arg   string
	Forms map[string]string
}

// Description documents a Message for translator/tooling use.
type Description struct {
	Msg  string
	Args map[string]ArgDescription
}

// ArgDescription documents a single named argument a Message expects.
// Unlike package i18n's MessageArgDescription, this carries no derived
// Type - deriving a documented arg's type from the verb its template
// actually uses is a rendering-engine concern, layered on top of this
// package's file-shape validation by the caller (see i18n.Message.
// validate), not part of the file shape itself.
type ArgDescription struct {
	Description string
}

// ValidationError reports a structural problem found while parsing/
// validating a translation file against this schema - e.g. a fragment
// missing its required "other" form. Name identifies the fragment/field at
// fault; Reason is a human-readable explanation.
type ValidationError struct {
	Name   string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid translation message %s: %s", e.Name, e.Reason)
}

// ParseMessage parses and structurally validates spec - the object under a
// "$message" key - per this schema: "msg" is required; every fragment
// must name its "arg" and define at least an "other" form; plural/ordinal
// fragments restrict form keys to CLDRCategories (select fragments accept
// any literal key). Reused unchanged by schema version 2 - see that
// package's doc comment.
func ParseMessage(spec map[string]any) (*Message, error) {
	msgText, ok := spec["msg"].(string)
	if !ok {
		return nil, &ValidationError{Name: MessageKey, Reason: `missing its "msg" field`}
	}

	m := &Message{Msg: msgText, Fragments: make(map[string]Fragment)}

	// source_hash is optional and purely additive: carry it through if present
	// (a string), ignore it otherwise. It never affects validation or render.
	if h, ok := spec["source_hash"].(string); ok {
		m.SourceHash = h
	}

	if fragsRaw, ok := spec["fragments"].(map[string]any); ok {
		for name, fragRaw := range fragsRaw {
			frag, err := parseFragment(name, fragRaw)
			if err != nil {
				return nil, err
			}
			m.Fragments[name] = frag
		}
	}

	if descRaw, ok := spec["description"]; ok {
		desc, err := parseDescription(descRaw)
		if err != nil {
			return nil, err
		}
		m.Description = desc
	}

	if err := m.validate(); err != nil {
		return nil, err
	}
	return m, nil
}

func parseFragment(name string, raw any) (Fragment, error) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return Fragment{}, &ValidationError{Name: name, Reason: "fragment must be an object"}
	}

	kind, _ := obj["kind"].(string)
	if kind == "" {
		kind = FragmentPlural
	}

	argName, ok := obj["arg"].(string)
	if !ok {
		return Fragment{}, &ValidationError{Name: name, Reason: `fragment is missing its "arg" field naming its argument`}
	}

	forms := make(map[string]string)
	for key, val := range obj {
		if key == "kind" || key == "arg" {
			continue
		}
		s, ok := val.(string)
		if !ok {
			continue
		}
		if kind != FragmentSelect && !CLDRCategories[key] {
			return Fragment{}, &ValidationError{Name: name, Reason: fmt.Sprintf(
				"%q is not a valid plural category (want zero, one, two, few, many, or other)", key)}
		}
		forms[key] = s
	}

	frag := Fragment{Kind: kind, Arg: argName, Forms: forms}
	if err := frag.validate(name); err != nil {
		return Fragment{}, err
	}
	return frag, nil
}

func (f Fragment) validate(name string) error {
	if f.Arg == "" {
		return &ValidationError{Name: name, Reason: `fragment is missing its "arg" field naming its argument`}
	}
	switch f.Kind {
	case FragmentPlural, FragmentOrdinal, FragmentSelect:
	default:
		return &ValidationError{Name: name, Reason: fmt.Sprintf(
			"unknown fragment kind %q (want %q, %q, or %q)", f.Kind, FragmentPlural, FragmentOrdinal, FragmentSelect)}
	}
	if _, ok := f.Forms["other"]; !ok {
		return &ValidationError{Name: name, Reason: `fragment is missing its required "other" form`}
	}
	return nil
}

func (m *Message) validate() error {
	for name, frag := range m.Fragments {
		if err := frag.validate(name); err != nil {
			return err
		}
	}
	return nil
}

func parseDescription(raw any) (*Description, error) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, &ValidationError{Name: "description", Reason: "description must be an object"}
	}

	desc := &Description{}
	if s, ok := obj["msg"].(string); ok {
		desc.Msg = s
	}
	if argsRaw, ok := obj["args"].(map[string]any); ok {
		desc.Args = make(map[string]ArgDescription, len(argsRaw))
		for name, v := range argsRaw {
			argObj, ok := v.(map[string]any)
			if !ok {
				continue
			}
			var ad ArgDescription
			if s, ok := argObj["description"].(string); ok {
				ad.Description = s
			}
			desc.Args[name] = ad
		}
	}
	return desc, nil
}
