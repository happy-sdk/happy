// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package v1

import (
	"errors"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/language"
)

func TestParseMessage_MinimalMsg(t *testing.T) {
	m, err := ParseMessage(map[string]any{"msg": "hello {name}"})
	testutils.NoError(t, err)
	testutils.Equal(t, "hello {name}", m.Msg)
	testutils.Equal(t, 0, len(m.Fragments))
	if m.Description != nil {
		t.Fatal("expected no description")
	}
}

func TestParseMessage_MissingMsg(t *testing.T) {
	_, err := ParseMessage(map[string]any{})
	if err == nil {
		t.Fatal("expected an error for a missing msg field")
	}
	var verr *ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	testutils.Equal(t, MessageKey, verr.Name)
}

func TestParseMessage_PluralFragmentDefaultsKind(t *testing.T) {
	m, err := ParseMessage(map[string]any{
		"msg": "{users_p}",
		"fragments": map[string]any{
			"users_p": map[string]any{
				"arg":   "users",
				"one":   "1 user",
				"other": "{users:d} users",
			},
		},
	})
	testutils.NoError(t, err)
	frag := m.Fragments["users_p"]
	testutils.Equal(t, FragmentPlural, frag.Kind)
	testutils.Equal(t, "users", frag.Arg)
	testutils.Equal(t, "1 user", frag.Forms["one"])
}

func TestParseMessage_FragmentMissingOtherIsRejected(t *testing.T) {
	_, err := ParseMessage(map[string]any{
		"msg": "{users_p}",
		"fragments": map[string]any{
			"users_p": map[string]any{
				"arg": "users",
				"one": "1 user",
			},
		},
	})
	if err == nil {
		t.Fatal("expected an error for a fragment missing its required \"other\" form")
	}
}

func TestParseMessage_FragmentMissingArgIsRejected(t *testing.T) {
	_, err := ParseMessage(map[string]any{
		"msg": "{users_p}",
		"fragments": map[string]any{
			"users_p": map[string]any{
				"other": "x users",
			},
		},
	})
	if err == nil {
		t.Fatal("expected an error for a fragment missing its required \"arg\" field")
	}
}

func TestParseMessage_InvalidCLDRCategoryIsRejected(t *testing.T) {
	_, err := ParseMessage(map[string]any{
		"msg": "{users_p}",
		"fragments": map[string]any{
			"users_p": map[string]any{
				"arg":      "users",
				"singular": "1 user",
				"other":    "{users:d} users",
			},
		},
	})
	if err == nil {
		t.Fatal("expected an error for an invalid plural category name")
	}
}

func TestParseMessage_SelectFragmentAcceptsArbitraryFormKeys(t *testing.T) {
	m, err := ParseMessage(map[string]any{
		"msg": "{pronoun}",
		"fragments": map[string]any{
			"pronoun": map[string]any{
				"kind":   FragmentSelect,
				"arg":    "gender",
				"female": "she",
				"male":   "he",
				"other":  "they",
			},
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "she", m.Fragments["pronoun"].Forms["female"])
}

func TestParseMessage_UnknownFragmentKindIsRejected(t *testing.T) {
	_, err := ParseMessage(map[string]any{
		"msg": "{x}",
		"fragments": map[string]any{
			"x": map[string]any{
				"kind":  "singular-vs-plural",
				"arg":   "n",
				"other": "x",
			},
		},
	})
	if err == nil {
		t.Fatal("expected an error for an unknown fragment kind")
	}
}

func TestParseMessage_WithDescription(t *testing.T) {
	m, err := ParseMessage(map[string]any{
		"msg": "hello {name:s}",
		"description": map[string]any{
			"msg": "greeting",
			"args": map[string]any{
				"name": map[string]any{"description": "the user's name"},
			},
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "greeting", m.Description.Msg)
	testutils.Equal(t, "the user's name", m.Description.Args["name"].Description)
}

func TestParseMessage_SourceHashRoundTrips(t *testing.T) {
	m, err := ParseMessage(map[string]any{
		"msg":         "hello {name:s}",
		"source_hash": "sha256:abc123",
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "sha256:abc123", m.SourceHash)
}

func TestParseMessage_SourceHashOptional(t *testing.T) {
	m, err := ParseMessage(map[string]any{"msg": "hello"})
	testutils.NoError(t, err)
	testutils.Equal(t, "", m.SourceHash)
}

func TestParse_Document(t *testing.T) {
	raw := map[string]any{"app.name": "hello"}
	doc, err := Parse(language.Und, raw)
	testutils.NoError(t, err)
	testutils.Equal(t, "hello", doc.Body["app.name"])
}
