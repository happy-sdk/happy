// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package options

import (
	"fmt"
	"sync"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/vars"
)

func TestSpec(t *testing.T) {
	t.Run("New_empty_name", func(t *testing.T) {
		spec, err := New("")
		testutils.Nil(t, spec)
		testutils.ErrorIs(t, err, ErrOptions)
	})
	t.Run("New_valid_name", func(t *testing.T) {
		spec, err := New("name")
		testutils.NotNil(t, spec)
		testutils.NoError(t, err)
	})
}

func TestNewWithOptions(t *testing.T) {
	spec, err := New("test",
		NewOption("key1", "val1"),
		NewOption("key2", 42),
	)
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)
	testutils.Equal(t, "test", opts.Name())
	testutils.Equal(t, 2, opts.Len())
	testutils.Assert(t, opts.Accepts("key1"))
	testutils.Assert(t, opts.Accepts("key2"))
}

func TestNewDuplicatedKey(t *testing.T) {
	_, err := New("test",
		NewOption("key1", "val1"),
		NewOption("key1", "val2"),
	)
	testutils.ErrorIs(t, err, ErrOption)
}

func TestSpecAddToSealed(t *testing.T) {
	spec, err := New("test", NewOption("key1", "val1"))
	testutils.NoError(t, err)
	_, err = spec.Seal()
	testutils.NoError(t, err)

	err = spec.Add(NewOption("key2", "val2"))
	testutils.ErrorIs(t, err, ErrOption)
}

func TestSetAndGet(t *testing.T) {
	spec, err := New("test", NewOption("key1", "default"))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	testutils.NoError(t, opts.Set("key1", "value1"))
	testutils.Equal(t, "value1", opts.Get("key1").Value().String())

	// Unknown key without a wildcard.
	err = opts.Set("unknown", "value")
	testutils.ErrorIs(t, err, ErrOptionNotExists)
}

func TestReadOnlyOption(t *testing.T) {
	spec, err := New("test", NewOption("key1", "default").Flags(ReadOnly))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	err = opts.Set("key1", "value1")
	testutils.ErrorIs(t, err, ErrOptionReadOnly)
}

func TestOnceOption(t *testing.T) {
	spec, err := New("test", NewOption("key1", "default").Flags(Mutable|Once))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	testutils.NoError(t, opts.Set("key1", "value1"))
	err = opts.Set("key1", "value2")
	testutils.ErrorIs(t, err, ErrOptionOnce)
}

func TestValidation(t *testing.T) {
	validator := func(opt Option) error {
		if opt.Value().String() == "invalid" {
			return fmt.Errorf("%w: invalid value", ErrOptionValidation)
		}
		return nil
	}
	spec, err := New("test", NewOption("key1", "default").Validator(validator))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	testutils.NoError(t, opts.Set("key1", "valid"))
	err = opts.Set("key1", "invalid")
	testutils.ErrorIs(t, err, ErrOptionValidation)
}

func TestParser(t *testing.T) {
	parser := func(opt Option, newval Value) (Value, error) {
		return vars.NewValue("parsed:" + newval.String())
	}
	spec, err := New("test", NewOption("key1", "default").Parser(parser))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	testutils.Equal(t, "parsed:default", opts.Get("key1").Value().String())

	testutils.NoError(t, opts.Set("key1", "value1"))
	testutils.Equal(t, "parsed:value1", opts.Get("key1").Value().String())
}

func TestAccepts(t *testing.T) {
	t.Run("no wildcard", func(t *testing.T) {
		spec, err := New("test", NewOption("key1", "val1"))
		testutils.NoError(t, err)
		opts, err := spec.Seal()
		testutils.NoError(t, err)

		testutils.Assert(t, opts.Accepts("key1"))
		testutils.Assert(t, !opts.Accepts("random"))
	})

	t.Run("empty string wildcard accepts anything", func(t *testing.T) {
		spec, err := New("test", NewOption("key1", "val1"))
		testutils.NoError(t, err)
		spec.AllowWildcard()
		opts, err := spec.Seal()
		testutils.NoError(t, err)

		testutils.Assert(t, opts.Accepts("key1"))
		testutils.Assert(t, opts.Accepts("anything"))
	})
}

// TestAcceptsPrefixWildcardRejectsNonMatching is a regression test: Accepts
// unconditionally returned true after the wildcard loop regardless of
// whether any wildcard actually matched, so once any non-empty prefix
// wildcard was registered, every key was accepted -- not just keys matching
// that prefix.
func TestAcceptsPrefixWildcardRejectsNonMatching(t *testing.T) {
	spec, err := New("test", NewOption("key1", "val1"))
	testutils.NoError(t, err)
	spec.opts.wildcards = append(spec.opts.wildcards, "allowed.")
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	testutils.Assert(t, opts.Accepts("key1"), "exact key must be accepted")
	testutils.Assert(t, opts.Accepts("allowed.foo"), "key matching prefix wildcard must be accepted")
	testutils.Assert(t, !opts.Accepts("unrelated.key"), "key not matching any wildcard or exact key must be rejected")
}

func TestExtend(t *testing.T) {
	dest, err := New("dest", NewOption("key1", "val1"))
	testutils.NoError(t, err)

	src, err := New("src", NewOption("key2", "val2"))
	testutils.NoError(t, err)

	testutils.NoError(t, dest.Extend(src))

	opts, err := dest.Seal()
	testutils.NoError(t, err)

	testutils.Assert(t, opts.Accepts("key1"))
	testutils.Assert(t, opts.Accepts("src.key2"))
	testutils.Equal(t, "val2", opts.Get("src.key2").Value().String())
}

func TestExtendNilSpec(t *testing.T) {
	dest, err := New("dest")
	testutils.NoError(t, err)
	err = dest.Extend(nil)
	testutils.ErrorIs(t, err, ErrOptions)
}

func TestExtendSealedOther(t *testing.T) {
	dest, err := New("dest")
	testutils.NoError(t, err)

	src, err := New("src", NewOption("key1", "val1"))
	testutils.NoError(t, err)
	_, err = src.Seal()
	testutils.NoError(t, err)

	err = dest.Extend(src)
	testutils.ErrorIs(t, err, ErrOptions)
}

func TestExtendDuplicateKey(t *testing.T) {
	dest, err := New("dest", NewOption("src.key1", "x"))
	testutils.NoError(t, err)

	src, err := New("src", NewOption("key1", "val1"))
	testutils.NoError(t, err)

	err = dest.Extend(src)
	testutils.ErrorIs(t, err, ErrOption)
}

// TestExtendConcurrentWithSet is a regression test: Extend read other's
// fields (other.sealed, other.opts.db/parsers/validators/wildcards)
// directly without acquiring other's locks, racing with any concurrent
// call that mutates other (e.g. other.Set).
func TestExtendConcurrentWithSet(t *testing.T) {
	other, err := New("other", NewOption("key2", "val2"))
	testutils.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		// Mutate the same Spec that's concurrently being extended, many
		// times, to maximize the chance of overlapping with Extend's
		// (previously unguarded) read window under -race.
		for i := range 1000 {
			_ = other.Set("key2", fmt.Sprintf("v%d", i))
		}
	}()
	go func() {
		defer wg.Done()
		dest, err := New("dest")
		testutils.NoError(t, err)
		_ = dest.Extend(other)
	}()
	wg.Wait()
}

func TestSeal(t *testing.T) {
	spec, err := New("test", NewOption("key1", "default1"))
	testutils.NoError(t, err)

	opts, err := spec.Seal()
	testutils.NoError(t, err)
	testutils.Equal(t, "default1", opts.Get("key1").Value().String())

	_, err = spec.Seal()
	testutils.ErrorIs(t, err, ErrOptions)
}

func TestRange(t *testing.T) {
	spec, err := New("test", NewOption("key1", "val1"), NewOption("key2", "val2"))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	var keys []string
	opts.Range(func(opt Option) bool {
		keys = append(keys, opt.Key())
		return true
	})
	testutils.Equal(t, 2, len(keys))
}

func TestAll(t *testing.T) {
	spec, err := New("test", NewOption("key1", "val1"), NewOption("key2", "val2"))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	count := 0
	for range opts.All() {
		count++
	}
	testutils.Equal(t, 2, count)
}

func TestDescribe(t *testing.T) {
	spec, err := New("test", NewOption("key1", "default1").Description("description for key1"))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	testutils.Equal(t, "description for key1", opts.Describe("key1"))
	testutils.Equal(t, UndefinedOptionDescription, opts.Describe("nonexistent"))
}

func TestLoad(t *testing.T) {
	spec, err := New("test", NewOption("key1", "value1"))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	opt, loaded := opts.Load("key1")
	testutils.Assert(t, loaded)
	testutils.Equal(t, "value1", opt.Value().String())

	_, loaded = opts.Load("nonexistent")
	testutils.Assert(t, !loaded)
}

func TestIsSet(t *testing.T) {
	spec, err := New("test", NewOption("key1", "default"))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	testutils.Assert(t, !opts.IsSet("key1"))
	testutils.NoError(t, opts.Set("key1", "value1"))
	testutils.Assert(t, opts.IsSet("key1"))
}

func TestOptionMethods(t *testing.T) {
	spec, err := New("test",
		NewOption("key1", "value1").Description("desc1"),
		NewOption("key2", 42).Flags(ReadOnly),
	)
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	opt1 := opts.Get("key1")
	testutils.Equal(t, "key1", opt1.Key())
	testutils.Equal(t, "value1", opt1.Value().String())
	testutils.Equal(t, "value1", opt1.Default().String())
	testutils.Equal(t, "desc1", opt1.Description())
	testutils.Assert(t, !opt1.ReadOnly())
	testutils.Assert(t, !opt1.IsSet())
	testutils.Assert(t, opt1.HasFlag(Mutable))
	testutils.Equal(t, "value1", opt1.String())

	opt2 := opts.Get("key2")
	testutils.Assert(t, opt2.ReadOnly())
	v, err := opt2.Value().Int()
	testutils.NoError(t, err)
	testutils.Equal(t, 42, v)
}

func TestArgMethods(t *testing.T) {
	arg := NewArg("key1", "value1")
	testutils.Equal(t, "key1", arg.Key())
	testutils.Equal(t, "value1", arg.Value())
}

func TestConcurrentSetAndGet(t *testing.T) {
	spec, err := New("test", NewOption("key1", "default"))
	testutils.NoError(t, err)
	opts, err := spec.Seal()
	testutils.NoError(t, err)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			_ = opts.Set("key1", fmt.Sprintf("value%d", i))
		}(i)
		go func() {
			defer wg.Done()
			_ = opts.Get("key1").Value().String()
		}()
	}
	wg.Wait()
}

func TestNewOptionInvalidKey(t *testing.T) {
	_, err := New("test", NewOption("", "value"))
	testutils.ErrorIs(t, err, ErrOption)
}
