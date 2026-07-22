// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"errors"
	"slices"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
)

func TestMessage_String(t *testing.T) {
	msg := NewMessage("hello {name}")
	testutils.Equal(t, "hello {name}", msg.String())
}

func TestMessage_Builder(t *testing.T) {
	msg := NewMessage("{users_p} total {n:d}").
		WithFragment("users_p", "users", map[string]string{
			"one":   "1 user",
			"other": "{users:d} users",
		}).
		WithDescription("header count", map[string]MessageArgDescription{
			"n": {Description: "some other count"},
		})

	testutils.Equal(t, "{users_p} total {n:d}", msg.Msg)
	frag, ok := msg.Fragments["users_p"]
	testutils.Assert(t, ok, "expected users_p fragment to be present")
	testutils.Equal(t, FragmentPlural, frag.Kind)
	testutils.Equal(t, "users", frag.Arg)
	testutils.Equal(t, "1 user", frag.Forms["one"])
	testutils.NotNil(t, msg.Description)
	testutils.Equal(t, "header count", msg.Description.Msg)

	testutils.NoError(t, msg.validate())
	testutils.Equal(t, ArgTypeDecimal, msg.Description.Args["n"].Type)
}

func TestMessage_Builder_OrdinalAndSelect(t *testing.T) {
	msg := NewMessage("{place_p}, {pronoun_p} won").
		WithOrdinalFragment("place_p", "rank", map[string]string{
			"one": "{rank:d}st", "two": "{rank:d}nd", "few": "{rank:d}rd", "other": "{rank:d}th",
		}).
		WithSelectFragment("pronoun_p", "gender", map[string]string{
			"male": "he", "female": "she", "other": "they",
		})

	place := msg.Fragments["place_p"]
	testutils.Equal(t, FragmentOrdinal, place.Kind)
	testutils.Equal(t, "rank", place.Arg)

	pronoun := msg.Fragments["pronoun_p"]
	testutils.Equal(t, FragmentSelect, pronoun.Kind)
	testutils.Equal(t, "gender", pronoun.Arg)
	testutils.NoError(t, msg.validate())
}

func TestMessage_Validate_MissingOther(t *testing.T) {
	msg := NewMessage("{users_p}").WithFragment("users_p", "users", map[string]string{
		"one": "1 user",
	})
	err := msg.validate()
	testutils.Error(t, err, "expected validation error for missing \"other\" form")
	testutils.Assert(t, errors.Is(err, ErrInvalidMessage), "expected err to be ErrInvalidMessage")
	testutils.Assert(t, errors.Is(err, Error), "expected err to satisfy errors.Is(err, i18n.Error)")
}

func TestMessage_Validate_MissingArg(t *testing.T) {
	msg := NewMessage("{users_p}")
	msg.Fragments["users_p"] = MessageFragment{Kind: FragmentPlural, Forms: map[string]string{"other": "x"}}
	err := msg.validate()
	testutils.Error(t, err, "expected validation error for missing arg")
	testutils.Assert(t, errors.Is(err, ErrInvalidMessage), "expected err to be ErrInvalidMessage")
}

func TestMessage_Validate_UnknownKind(t *testing.T) {
	msg := NewMessage("{x}")
	msg.Fragments["x"] = MessageFragment{Kind: "bogus", Arg: "n", Forms: map[string]string{"other": "x"}}
	err := msg.validate()
	testutils.Error(t, err, "expected validation error for unknown kind")
	testutils.Assert(t, errors.Is(err, ErrInvalidMessage), "expected err to be ErrInvalidMessage")
}

func TestMessage_Validate_OK(t *testing.T) {
	msg := NewMessage("{users_p}").WithFragment("users_p", "users", map[string]string{"other": "x"})
	testutils.NoError(t, msg.validate())
}

// parseMessageObject

func TestParseMessageObject_NotAMessage(t *testing.T) {
	_, isMessage, err := parseMessageObject(map[string]any{
		"nested": map[string]any{"key": "value"},
	})
	testutils.Assert(t, !isMessage, "expected an object with no \"$message\" field not to be treated as a Message")
	testutils.NoError(t, err)
}

func TestParseMessageObject_NoCollisionWithLiteralKeys(t *testing.T) {
	// A translation group that happens to have real keys named "msg" or
	// "description" must not be misdetected as a Message, since those
	// names only mean something inside "$message".
	_, isMessage, err := parseMessageObject(map[string]any{
		"msg":         "just a regular nested key called msg",
		"description": "just a regular nested key called description",
	})
	testutils.Assert(t, !isMessage, "expected no \"$message\" key to mean this is not a Message")
	testutils.NoError(t, err)
}

func TestParseMessageObject_Basic(t *testing.T) {
	raw := map[string]any{
		messageKey: map[string]any{
			"msg": "{users_p} total {notifications_p}",
			"fragments": map[string]any{
				"users_p": map[string]any{
					"arg":   "users",
					"one":   "1 user has",
					"other": "{users:d} users have",
				},
				"notifications_p": map[string]any{
					"arg":   "unread_notifications",
					"one":   "{unread_notifications:d} unread notification",
					"other": "{unread_notifications:d} unread notifications",
				},
			},
			"description": map[string]any{
				"msg": "message for translators",
				"args": map[string]any{
					"users": map[string]any{"description": "number of users"},
				},
			},
		},
	}
	msg, isMessage, err := parseMessageObject(raw)
	testutils.Assert(t, isMessage, "expected object with \"$message\" field to be treated as a Message")
	testutils.NoError(t, err)
	testutils.NotNil(t, msg)
	testutils.Equal(t, "{users_p} total {notifications_p}", msg.Msg)
	testutils.Equal(t, 2, len(msg.Fragments))
	testutils.Equal(t, FragmentPlural, msg.Fragments["users_p"].Kind)
	testutils.NotNil(t, msg.Description)
	testutils.Equal(t, "message for translators", msg.Description.Msg)
	testutils.Equal(t, ArgTypeDecimal, msg.Description.Args["users"].Type)
}

func TestParseMessageObject_SourceHashRoundTrips(t *testing.T) {
	msg, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{
			"msg":         "hello {name:s}",
			"source_hash": "sha256:deadbeef",
		},
	})
	testutils.Assert(t, isMessage, "expected a Message")
	testutils.NoError(t, err)
	testutils.Equal(t, "sha256:deadbeef", msg.SourceHash)
}

func TestParseMessageObject_ExplicitOrdinalAndSelectKinds(t *testing.T) {
	raw := map[string]any{
		messageKey: map[string]any{
			"msg": "{place_p}, {pronoun_p} won",
			"fragments": map[string]any{
				"place_p": map[string]any{
					"kind": "ordinal",
					"arg":  "rank",
					"one":  "{rank:d}st", "two": "{rank:d}nd", "few": "{rank:d}rd", "other": "{rank:d}th",
				},
				"pronoun_p": map[string]any{
					"kind": "select",
					"arg":  "gender",
					"male": "he", "female": "she", "other": "they",
				},
			},
		},
	}
	msg, isMessage, err := parseMessageObject(raw)
	testutils.Assert(t, isMessage, "expected object with \"$message\" field to be treated as a Message")
	testutils.NoError(t, err)
	testutils.Equal(t, FragmentOrdinal, msg.Fragments["place_p"].Kind)
	testutils.Equal(t, FragmentSelect, msg.Fragments["pronoun_p"].Kind)
}

func TestParseMessageObject_NotAnObject(t *testing.T) {
	_, isMessage, err := parseMessageObject(map[string]any{
		messageKey: "not an object",
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message regardless of validity")
	testutils.Error(t, err, "expected error when $message isn't an object")
	testutils.Assert(t, errors.Is(err, ErrInvalidMessage), "expected err to be ErrInvalidMessage")
}

func TestParseMessageObject_MissingMsg(t *testing.T) {
	_, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{"fragments": map[string]any{}},
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message")
	testutils.Error(t, err, "expected error for missing \"msg\" field")
}

func TestParseMessageObject_FragmentNotObject(t *testing.T) {
	_, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{
			"msg":       "{x}",
			"fragments": map[string]any{"x": "not an object"},
		},
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message")
	testutils.Error(t, err, "expected error for non-object fragment")
	testutils.Assert(t, errors.Is(err, ErrInvalidMessage), "expected err to be ErrInvalidMessage")
}

func TestParseMessageObject_FragmentMissingArg(t *testing.T) {
	_, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{
			"msg":       "{x}",
			"fragments": map[string]any{"x": map[string]any{"other": "y"}},
		},
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message")
	testutils.Error(t, err, "expected error for fragment missing arg")
}

func TestParseMessageObject_FragmentMissingOther(t *testing.T) {
	_, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{
			"msg":       "{x}",
			"fragments": map[string]any{"x": map[string]any{"arg": "n", "one": "y"}},
		},
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message")
	testutils.Error(t, err, "expected error for fragment missing \"other\"")
}

func TestParseMessageObject_FragmentInvalidCategoryName(t *testing.T) {
	// A plural/ordinal fragment with a form key that isn't a real CLDR
	// category (e.g. a typo like "singular") must fail loudly instead of
	// silently becoming dead, unreachable data.
	_, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{
			"msg": "{x}",
			"fragments": map[string]any{
				"x": map[string]any{"arg": "n", "singular": "y", "other": "z"},
			},
		},
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message")
	testutils.Error(t, err, "expected error for invalid CLDR category name")
	testutils.Assert(t, errors.Is(err, ErrInvalidMessage), "expected err to be ErrInvalidMessage")
}

func TestParseMessageObject_SelectAllowsArbitraryFormNames(t *testing.T) {
	msg, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{
			"msg": "{x}",
			"fragments": map[string]any{
				"x": map[string]any{"kind": "select", "arg": "mood", "happy": "yay", "sad": "aww", "other": "ok"},
			},
		},
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message")
	testutils.NoError(t, err)
	testutils.Equal(t, "yay", msg.Fragments["x"].Forms["happy"])
}

func TestMessage_ArgTypes(t *testing.T) {
	msg := NewMessage("{users_p} total {notifications_p}, seen by {name:s}").
		WithFragment("users_p", "users", map[string]string{
			"one": "1 user has", "other": "{users:d} users have",
		}).
		WithFragment("notifications_p", "unread", map[string]string{
			"other": "{unread:d} notifications",
		})

	types := msg.ArgTypes()
	testutils.Equal(t, ArgTypeString, types["name"])
	testutils.Equal(t, ArgTypeDecimal, types["users"])
	testutils.Equal(t, ArgTypeDecimal, types["unread"])
	// Fragment names themselves are not args.
	_, hasFragmentAsArg := types["users_p"]
	testutils.Assert(t, !hasFragmentAsArg, "expected a fragment name not to be reported as an arg")
}

func TestMessage_ArgTypes_NoDescriptionNeeded(t *testing.T) {
	// ArgTypes works even when the message has no Description at all -
	// it's meant to help a translation UI introspect any message.
	msg := NewMessage("{count:d} items")
	types := msg.ArgTypes()
	testutils.Equal(t, ArgTypeDecimal, types["count"])
}

// TestMessage_ArgTypes_FragmentDiscriminatorNeverDisplayed is a regression
// test for a bug where a fragment's own discriminating Arg was silently
// absent from ArgTypes() whenever it was only ever used to pick a category,
// never interpolated literally (e.g. a "gender" select fragment, or a plural
// count that's never shown as text). That meant addons/l10n's `l10n generate`
// silently omitted the discriminator from a generated accessor's parameters,
// producing a typed function that could never be given the value that
// selects its own wording - the fragment would always render "other".
func TestMessage_ArgTypes_FragmentDiscriminatorNeverDisplayed(t *testing.T) {
	msg := NewMessage("{greeting}").
		WithSelectFragment("greeting", "gender", map[string]string{
			"male": "he", "female": "she", "other": "they",
		})
	types := msg.ArgTypes()
	typ, ok := types["gender"]
	testutils.Assert(t, ok, "expected the select fragment's discriminator \"gender\" to be reported even though it's never interpolated literally")
	testutils.Equal(t, ArgTypeAny, typ)

	plural := NewMessage("{count_p}").
		WithFragment("count_p", "n", map[string]string{
			"one": "single", "other": "multiple",
		})
	pluralTypes := plural.ArgTypes()
	nTyp, ok := pluralTypes["n"]
	testutils.Assert(t, ok, "expected the plural fragment's discriminator \"n\" to be reported even though it's never interpolated literally")
	testutils.Equal(t, ArgTypeDecimal, nTyp)
}

// TestMessage_ArgTypes_LiteralUsageWinsOverFragmentDefault confirms that when
// a fragment's discriminator IS also interpolated literally elsewhere, that
// usage's real verb takes precedence over the fragment-kind default.
func TestMessage_ArgTypes_LiteralUsageWinsOverFragmentDefault(t *testing.T) {
	msg := NewMessage("{items_p}: {count:number} total").
		WithFragment("items_p", "count", map[string]string{
			"one": "1 item", "other": "many items",
		})
	types := msg.ArgTypes()
	testutils.Equal(t, ArgTypeNumber, types["count"])
}

func TestArgType_ExampleValue(t *testing.T) {
	cases := []struct {
		typ  ArgType
		want string
	}{
		{ArgTypeAny, "example"},
		{ArgTypeType, "string"},
		{ArgTypeString, "example"},
		{ArgTypeDecimal, "42"},
		{ArgTypeFloat, "3.14"},
		{ArgTypeTruth, "true"},
		{ArgTypePointer, "0xc0000a4000"},
		{ArgTypeQuoted, `"example"`},
		{ArgTypeHex, "2a"},
		{ArgTypeUnknown, ""},
	}
	for _, tt := range cases {
		testutils.Equal(t, tt.want, tt.typ.ExampleValue())
	}
}

func TestArgType_String(t *testing.T) {
	cases := []struct {
		typ  ArgType
		want string
	}{
		{ArgTypeAny, "any"},
		{ArgTypeType, "type"},
		{ArgTypeString, "string"},
		{ArgTypeDecimal, "decimal"},
		{ArgTypeFloat, "float"},
		{ArgTypeTruth, "truth"},
		{ArgTypePointer, "pointer"},
		{ArgTypeQuoted, "quoted"},
		{ArgTypeHex, "hex"},
		{ArgTypeUnknown, "unknown"},
	}
	for _, tt := range cases {
		testutils.Equal(t, tt.want, tt.typ.String())
	}
}

func TestArgTypeForVerb(t *testing.T) {
	cases := []struct {
		verb string
		want ArgType
	}{
		{"", ArgTypeAny},
		{"v", ArgTypeAny},
		{"T", ArgTypeType},
		{"s", ArgTypeString},
		{"d", ArgTypeDecimal},
		{"f", ArgTypeFloat},
		{"f.2", ArgTypeFloat}, // a precision suffix doesn't change the category
		{"t", ArgTypeTruth},
		{"p", ArgTypePointer},
		{"q", ArgTypeQuoted},
		{"x", ArgTypeHex},
		{"o", ArgTypeUnknown}, // octal isn't catalogued
	}
	for _, tt := range cases {
		testutils.Equal(t, tt.want, argTypeForVerb(tt.verb))
	}
}

func TestParseMessageObject_DescriptionArgNotInTemplate(t *testing.T) {
	_, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{
			"msg": "{x:s}",
			"description": map[string]any{
				"args": map[string]any{
					"typo_name": map[string]any{"description": "documents an arg that doesn't exist"},
				},
			},
		},
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message")
	testutils.Error(t, err, "expected error for a documented arg the template never uses")
	testutils.Assert(t, errors.Is(err, ErrInvalidMessage), "expected err to be ErrInvalidMessage")
}

func TestParseMessageObject_DescriptionArgUnknownVerb(t *testing.T) {
	_, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{
			"msg": "{x:o}", // octal isn't a catalogued ArgType
			"description": map[string]any{
				"args": map[string]any{
					"x": map[string]any{"description": "an octal value"},
				},
			},
		},
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message")
	testutils.Error(t, err, "expected error for an arg whose verb has no known ArgType")
	testutils.Assert(t, errors.Is(err, ErrInvalidMessage), "expected err to be ErrInvalidMessage")
}

func TestParseMessageObject_DescriptionArgBareVerbIsAny(t *testing.T) {
	msg, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{
			"msg": "{x}",
			"description": map[string]any{
				"args": map[string]any{
					"x": map[string]any{"description": "default-formatted value"},
				},
			},
		},
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message")
	testutils.NoError(t, err)
	testutils.Equal(t, ArgTypeAny, msg.Description.Args["x"].Type)
}

func TestParseMessageObject_DescriptionNotObject(t *testing.T) {
	_, isMessage, err := parseMessageObject(map[string]any{
		messageKey: map[string]any{
			"msg":         "{x:d}",
			"description": "not an object",
		},
	})
	testutils.Assert(t, isMessage, "expected \"$message\" key to be treated as a Message")
	testutils.Error(t, err, "expected error for non-object description")
}

// Rendering: category selection and end-to-end via T/TL.

func TestCldrCategory_EnglishCardinal(t *testing.T) {
	testutils.Equal(t, "one", cldrCategory(plural.Cardinal, language.English, 1))
	testutils.Equal(t, "other", cldrCategory(plural.Cardinal, language.English, 0))
	testutils.Equal(t, "other", cldrCategory(plural.Cardinal, language.English, 2))
	testutils.Equal(t, "other", cldrCategory(plural.Cardinal, language.English, 5))
}

func TestCldrCategory_RussianCardinal(t *testing.T) {
	testutils.Equal(t, "one", cldrCategory(plural.Cardinal, language.Russian, 1))
	testutils.Equal(t, "one", cldrCategory(plural.Cardinal, language.Russian, 21))
	testutils.Equal(t, "few", cldrCategory(plural.Cardinal, language.Russian, 2))
	testutils.Equal(t, "few", cldrCategory(plural.Cardinal, language.Russian, 3))
	testutils.Equal(t, "few", cldrCategory(plural.Cardinal, language.Russian, 22))
	testutils.Equal(t, "many", cldrCategory(plural.Cardinal, language.Russian, 0))
	testutils.Equal(t, "many", cldrCategory(plural.Cardinal, language.Russian, 5))
	testutils.Equal(t, "many", cldrCategory(plural.Cardinal, language.Russian, 11))
}

func TestCldrCategory_EnglishOrdinal(t *testing.T) {
	testutils.Equal(t, "one", cldrCategory(plural.Ordinal, language.English, 1))
	testutils.Equal(t, "two", cldrCategory(plural.Ordinal, language.English, 2))
	testutils.Equal(t, "few", cldrCategory(plural.Ordinal, language.English, 3))
	testutils.Equal(t, "other", cldrCategory(plural.Ordinal, language.English, 4))
	testutils.Equal(t, "other", cldrCategory(plural.Ordinal, language.English, 11))
	testutils.Equal(t, "one", cldrCategory(plural.Ordinal, language.English, 21))
}

func TestCldrCategory_NonNumeric(t *testing.T) {
	testutils.Equal(t, "other", cldrCategory(plural.Cardinal, language.English, "not-a-number"))
}

func TestRenderMessage_PluralEndToEnd(t *testing.T) {
	Initialize(language.English)
	key := "test.message.notifications"
	_ = RegisterTranslation(language.English, key, map[string]any{
		messageKey: map[string]any{
			"msg": "{users_p} total {notifications_p}",
			"fragments": map[string]any{
				"users_p": map[string]any{
					"arg": "users", "one": "1 user has", "other": "{users:d} users have",
				},
				"notifications_p": map[string]any{
					"arg": "unread_notifications", "one": "{unread_notifications:d} unread notification", "other": "{unread_notifications:d} unread notifications",
				},
			},
		},
	})

	testutils.Equal(t, "1 user has total 1 unread notification", T(key, "users", 1, "unread_notifications", 1))
	testutils.Equal(t, "10 users have total 100 unread notifications", T(key, "users", 10, "unread_notifications", 100))
}

func TestRenderMessage_RussianPlurals_EndToEnd(t *testing.T) {
	Initialize(language.English)
	key := "test.message.ru_notifications"
	_ = RegisterTranslation(language.Russian, key, map[string]any{
		messageKey: map[string]any{
			"msg": "{notifications_p}",
			"fragments": map[string]any{
				"notifications_p": map[string]any{
					"arg": "n", "one": "{n:d} уведомление", "few": "{n:d} уведомления", "many": "{n:d} уведомлений", "other": "{n:d} уведомления",
				},
			},
		},
	})

	testutils.Equal(t, "1 уведомление", TL(language.Russian, key, "n", 1))
	testutils.Equal(t, "2 уведомления", TL(language.Russian, key, "n", 2))
	testutils.Equal(t, "5 уведомлений", TL(language.Russian, key, "n", 5))
	testutils.Equal(t, "21 уведомление", TL(language.Russian, key, "n", 21))
}

func TestRenderMessage_OrdinalEndToEnd(t *testing.T) {
	Initialize(language.English)
	key := "test.message.rank"
	_ = RegisterTranslation(language.English, key, map[string]any{
		messageKey: map[string]any{
			"msg": "{place_p} place",
			"fragments": map[string]any{
				"place_p": map[string]any{
					"kind": "ordinal", "arg": "rank",
					"one": "{rank:d}st", "two": "{rank:d}nd", "few": "{rank:d}rd", "other": "{rank:d}th",
				},
			},
		},
	})

	testutils.Equal(t, "1st place", T(key, "rank", 1))
	testutils.Equal(t, "2nd place", T(key, "rank", 2))
	testutils.Equal(t, "3rd place", T(key, "rank", 3))
	testutils.Equal(t, "4th place", T(key, "rank", 4))
	testutils.Equal(t, "11th place", T(key, "rank", 11))
	testutils.Equal(t, "21st place", T(key, "rank", 21))
}

func TestRenderMessage_SelectEndToEnd(t *testing.T) {
	Initialize(language.English)
	key := "test.message.pronoun"
	_ = RegisterTranslation(language.English, key, map[string]any{
		messageKey: map[string]any{
			"msg": "{pronoun_p} went home",
			"fragments": map[string]any{
				"pronoun_p": map[string]any{
					"kind": "select", "arg": "gender",
					"male": "He", "female": "She", "other": "They",
				},
			},
		},
	})

	testutils.Equal(t, "He went home", T(key, "gender", "male"))
	testutils.Equal(t, "She went home", T(key, "gender", "female"))
	testutils.Equal(t, "They went home", T(key, "gender", "nonbinary"))
}

func TestRenderMessage_MissingArgValue(t *testing.T) {
	Initialize(language.English)
	key := "test.message.missing_arg"
	_ = RegisterTranslation(language.English, key, map[string]any{
		messageKey: map[string]any{
			"msg": "{users_p}",
			"fragments": map[string]any{
				"users_p": map[string]any{"arg": "users", "other": "{users:d} users"},
			},
		},
	})

	out := T(key)
	testutils.Assert(t, out == "%!users(MISSING)", "expected a MISSING marker, got %q", out)
}

func TestRenderMessage_LanguageWithFewerCategories(t *testing.T) {
	// English only distinguishes one/other; a fragment lacking "one"
	// entirely must still fall back to "other" cleanly.
	Initialize(language.English)
	key := "test.message.other_only"
	_ = RegisterTranslation(language.English, key, map[string]any{
		messageKey: map[string]any{
			"msg": "{items_p}",
			"fragments": map[string]any{
				"items_p": map[string]any{"arg": "n", "other": "{n:d} items"},
			},
		},
	})

	testutils.Equal(t, "1 items", T(key, "n", 1))
	testutils.Equal(t, "3 items", T(key, "n", 3))
}

func TestRenderFragment_MissingOtherProducesMarker(t *testing.T) {
	// validate() normally prevents a fragment from lacking "other" at
	// registration time; renderFragment is defensive about it anyway,
	// exercised directly here since that path can't be reached through
	// the public API once validate() is in place.
	frag := MessageFragment{Kind: FragmentPlural, Arg: "n", Forms: map[string]string{"one": "x"}}
	out := renderFragment(language.Japanese, frag, map[string]any{"n": 5})
	testutils.Equal(t, "%!FORM(n)", out)
}

func TestRegisterTranslation_InvalidMessageObject_ReturnsError(t *testing.T) {
	Initialize(language.English)
	err := RegisterTranslation(language.English, "test.message.invalid", map[string]any{
		messageKey: map[string]any{
			"msg":       "{x}",
			"fragments": map[string]any{"x": map[string]any{"arg": "n"}}, // missing "other"
		},
	})
	testutils.Error(t, err, "expected error for invalid message registration")
	testutils.Assert(t, errors.Is(err, ErrInvalidMessage), "expected err to be ErrInvalidMessage")
}

func TestRegisterTranslation_DirectMessageValue(t *testing.T) {
	Initialize(language.English)
	msg := NewMessage("{n:d} items")
	err := RegisterTranslation(language.English, "test.message.direct", msg)
	testutils.NoError(t, err)
	testutils.Equal(t, "5 items", T("test.message.direct", "n", 5))
}

func TestRegisterTranslation_DirectInvalidMessageValue(t *testing.T) {
	Initialize(language.English)
	msg := NewMessage("{x}")
	msg.Fragments["x"] = MessageFragment{Kind: FragmentPlural, Arg: "n", Forms: map[string]string{}} // missing "other"
	err := RegisterTranslation(language.English, "test.message.direct_invalid", msg)
	testutils.Error(t, err, "expected error registering an invalid *Message directly")
	testutils.Assert(t, errors.Is(err, ErrInvalidMessage), "expected err to be ErrInvalidMessage")
}

func TestRegisterTranslations_NestedObjectStillRecursesAsKeys(t *testing.T) {
	// Regression guard: an object with no "$message" field must keep
	// behaving as a plain nested group of translation keys, unaffected by
	// Message support existing alongside it.
	Initialize(language.English)
	err := RegisterTranslations(language.English, map[string]any{
		"test.nested.group": map[string]any{
			"a": "Value A",
			"b": "Value B",
		},
	})
	testutils.NoError(t, err)
	testutils.Equal(t, "Value A", T("test.nested.group.a"))
	testutils.Equal(t, "Value B", T("test.nested.group.b"))
}

func TestToPluralOperand_AllNumericTypes(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  int
	}{
		{"int", int(3), 3},
		{"int negative", int(-3), 3},
		{"int8", int8(3), 3},
		{"int16", int16(3), 3},
		{"int32", int32(3), 3},
		{"int64", int64(3), 3},
		{"uint", uint(3), 3},
		{"uint8", uint8(3), 3},
		{"uint16", uint16(3), 3},
		{"uint32", uint32(3), 3},
		{"uint64", uint64(3), 3},
		{"float32", float32(3), 3},
		{"float64", float64(3), 3},
	}
	for _, tt := range cases {
		n, ok := toPluralOperand(tt.value)
		testutils.Assert(t, ok, "%s: expected ok", tt.name)
		testutils.Equal(t, tt.want, n)
	}

	_, ok := toPluralOperand("not a number")
	testutils.Assert(t, !ok, "expected ok=false for a non-numeric value")
}

func TestGetLanguages_ReflectsMessageOnlyRegistration(t *testing.T) {
	// Regression guard: a *Message value never touches globalCatalog (it's
	// rendered by this package's own engine, not x/text's), so a language
	// whose ENTIRE bundle is Message-based must still be reported by
	// GetLanguages - it must not rely on globalCatalog.Languages().
	Initialize(language.English)
	_ = RegisterTranslation(language.Japanese, "test.message.lang_tracking", map[string]any{
		messageKey: map[string]any{"msg": "{n:d} items"},
	})
	testutils.Assert(t, SupportsLanguage(language.Japanese), "expected Japanese to be supported after a Message-only registration")
	testutils.Assert(t, slices.Contains(GetLanguages(), language.Japanese), "expected GetLanguages to include Japanese after a Message-only registration")
}

func TestAbsInt(t *testing.T) {
	testutils.Equal(t, 5, absInt(5))
	testutils.Equal(t, 5, absInt(-5))
	testutils.Equal(t, 0, absInt(0))
}
