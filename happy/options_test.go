// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"fmt"
	"testing"

	"github.com/mkungla/happy/pkg/vars"
	"github.com/mkungla/happy/sdk/testutils"
)

func TestAppDefaultOptions(t *testing.T) {
	app := New()

	var tests = []struct {
		Key      string
		Expected string
	}{
		{"app.name", "Happy Application"},
		{"app.settings.persistent", "false"},
		{"log.level", "info"},
		{"log.source", "false"},
		{"log.colors", "true"},
		{"log.stdlog", "false"},
		{"log.secrets", ""},
	}
	for _, test := range tests {
		opt := app.session.Get(test.Key)
		testutils.Equal(t, test.Expected, opt.String())
		testutils.True(t, opt.ReadOnly(), "expected %s to be readonly", test.Key)
	}

	testutils.Error(t, app.session.Set("app.name", "Test Application"), "app.name should be readonly")
	testutils.NoError(t, app.session.Set("some.name", "Test Application"), "some.name should be allowed")
	testutils.Error(t, app.session.Set("app.mycustom", ""), "app.mycustom should not be allowed")

	testutils.Equal(t, "config", app.session.opts.Name())
}

func TestNewOptions(t *testing.T) {
	// Test case for nil defaults
	opts, err := NewOptions("test", nil)
	testutils.NoError(t, err, "Unexpected error for nil defaults: %v", err)
	if opts.config != nil {
		t.Errorf("Expected config map to be nil for nil defaults, got non-nil %v", opts.config)
	}

	// Test case for invalid key in option config
	_, err = NewOptions("test", []OptionArg{
		Option("$$$", nil),
	})
	testutils.Error(t, err, "Expected error for invalid key in option config, got nil")

	// Test case for duplicate key in option configs
	_, err = NewOptions("test", []OptionArg{
		Option("key1", nil),
		Option("key1", nil),
	})
	testutils.Error(t, err, "Expected error for duplicate key in option configs, got nil")
}

func TestOptionSet(t *testing.T) {
	// Test case for key not accepted by Options struct
	opts := Options{
		name: "test",
		config: map[string]OptionArg{
			"key1": {},
			"key2": {},
		},
	}
	err := opts.Set("key3", "value")
	testutils.Error(t, err, "Expected error for key not accepted by Options struct, got nil")

	// Test case for read-only key
	opts = Options{
		name: "test",
	}
	testutils.NoError(t, opts.db.StoreReadOnly("key1", "value", true))

	err = opts.Set("key1", "new value")
	testutils.Error(t, err, "Expected error for read-only key, got nil")

	// Test case for invalid value
	opts = Options{
		name: "test",
	}
	err = opts.Set("key1", make(chan int))
	testutils.Error(t, err, "Expected error for invalid value, got nil")

	// Test case for no validation required
	opts = Options{
		name:   "test",
		config: nil,
	}
	err = opts.Set("key1", "value")
	testutils.NoError(t, err, "Unexpected error for no validation required: %v", err)

	// Test case for specific validator
	opts = Options{
		name: "test",
		config: map[string]OptionArg{
			"key1": {
				validator: func(key string, value vars.Value) error {
					if value.String() == "invalid" {
						return fmt.Errorf("invalid value for key %s", key)
					}
					return nil
				},
			},
		},
	}
	err = opts.Set("key1", "invalid")
	testutils.Error(t, err, "Expected error for invalid value with specific validator, got nil")

	// Test case for fallback validator
	opts = Options{
		name: "test",
		config: map[string]OptionArg{
			"*": {
				validator: func(key string, value vars.Value) error {
					if value.String() == "invalid" {
						return fmt.Errorf("invalid value for key %s", key)
					}
					return nil
				},
			},
		},
	}
	err = opts.Set("key1", "invalid")
	testutils.Error(t, err, "Expected error for invalid value with fallback validator, got nil")
}
