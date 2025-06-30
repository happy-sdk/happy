// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package options

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
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

// func TestOptions_New(t *testing.T) {
// 	specs := []Spec{
// 		NewOption("key1", "val1", "desc1", KindConfig, NoopValueValidator),
// 		NewOption("key2", 42, "desc2", KindReadOnly, nil),
// 	}
// 	opts, err := New("test", specs...)
// 	if err != nil {
// 		t.Fatalf("New failed: %v", err)
// 	}
// 	if opts.Name() != "test" || opts.String() != "test" {
// 		t.Errorf("Name: got %q, want %q", opts.Name(), "test")
// 	}
// 	if opts.Len() != 2 {
// 		t.Errorf("Len: got %d, want %d", opts.Len(), 2)
// 	}
// 	if !opts.Has("key1") || !opts.Has("key2") {
// 		t.Error("Has: expected key1 and key2 to exist")
// 	}
// }

// func TestNewDuplicatedSpec(t *testing.T) {
// 	specs := []Spec{
// 		NewOption("key1", "val1", "desc1", KindConfig, NoopValueValidator),
// 		NewOption("key1", "val1", "desc1", KindConfig, NoopValueValidator),
// 	}
// 	_, err := New("test", specs...)
// 	if err == nil {
// 		t.Error("New: expected error for duplicated key, got nil")
// 	}
// }

// func TestSetAndGet(t *testing.T) {
// 	opts, _ := New("test")

// 	// Test setting and getting string option
// 	err := opts.Set("key1", "value1")
// 	if err == nil {
// 		t.Error("Set: expected error for unaccepted key, got nil")
// 	}

// 	// Add valid option spec
// 	testutils.NoError(t, opts.Add(NewOption("key1", "default", "desc", KindConfig, NoopValueValidator)))
// 	err = opts.Set("key1", "value1")
// 	if err != nil {
// 		t.Fatalf("Set failed: %v", err)
// 	}
// 	val := opts.Get("key1").String()
// 	if val != "value1" {
// 		t.Errorf("Get: got %q, want %q", val, "value1")
// 	}

// 	// Test readonly option
// 	testutils.NoError(t, opts.Add(NewOption("key2", 42, "desc", KindReadOnly, nil)))
// 	err = opts.Set("key2", 99)
// 	if err != nil {
// 		t.Fatalf("Set failed: %v", err)
// 	}
// 	testutils.NoError(t, opts.Seal())
// 	err = opts.Set("key2", 100)
// 	if !errors.Is(err, ErrOptionReadOnly) {
// 		t.Errorf("Set readonly: expected ErrOptionReadOnly, got %v", err)
// 	}

// 	empty := opts.Get("empty").String()
// 	if empty != "" {
// 		t.Errorf("Get: got %q, want %q", val, "")
// 	}
// }

// func TestValidation(t *testing.T) {
// 	opts, _ := New("test")
// 	validator := func(opt Option) error {
// 		if opt.Value().String() == "invalid" {
// 			return fmt.Errorf("%w: invalid value", ErrOptionValidation)
// 		}
// 		return nil
// 	}
// 	testutils.NoError(t, opts.Add(NewSpec("key1", "default", "desc", KindConfig, validator).Build()))
// 	testutils.Error(t, opts.Add(NewSpec("key1", "default", "desc", KindConfig, validator).Build()))

// 	// Test valid value
// 	err := opts.Set("key1", "valid")
// 	if err != nil {
// 		t.Fatalf("Set valid value failed: %v", err)
// 	}

// 	// Test invalid value
// 	err = opts.Set("key1", "invalid")
// 	if !errors.Is(err, ErrOptionValidation) {
// 		t.Errorf("Set invalid value: expected ErrOptionValidation, got %v", err)
// 	}

// 	// Test not empty validator
// 	testutils.NoError(t, opts.Add(NewSpec("key2", "default", "desc", KindConfig, OptionValidatorNotEmpty)))
// 	err = opts.Set("key2", "")
// 	if !errors.Is(err, ErrOption) {
// 		t.Errorf("Set empty value: expected ErrOption, got %v", err)
// 	}
// }

// func TestOptions_Seal(t *testing.T) {
// 	t.Run("basic seal", func(t *testing.T) {
// 		specs := []Spec{
// 			NewOption("key1", "default1", "desc1", KindConfig, NoopValueValidator),
// 			NewOption("key2", 42, "desc2", KindConfig, nil),
// 		}
// 		opts, _ := New("test", specs...)
// 		testutils.NoError(t, opts.Set("key1", "value1"))

// 		err := opts.Seal()
// 		if err != nil {
// 			t.Fatalf("Seal failed: %v", err)
// 		}
// 		if !opts.sealed {
// 			t.Error("Seal: expected sealed to be true")
// 		}
// 		if opts.Get("key2").Int() != 42 {
// 			t.Errorf("Seal: got %d, want %d", opts.Get("key2").Int(), 42)
// 		}

// 		// Test adding to sealed options
// 		err = opts.Add(NewOption("key3", "val3", "desc3", KindConfig, nil))
// 		if !errors.Is(err, ErrOption) {
// 			t.Errorf("Add to sealed: expected ErrOption, got %v", err)
// 		}

// 		// Test sealing already sealed options
// 		err = opts.Seal()
// 		if !errors.Is(err, ErrOption) {
// 			t.Errorf("Seal already sealed: expected ErrOption, got %v", err)
// 		}
// 	})

// 	t.Run("wildcard spec", func(t *testing.T) {
// 		// Hit: continue for key == "*"
// 		specs := []Spec{
// 			NewOption("key1", "default1", "desc1", KindConfig, NoopValueValidator),
// 			NewOption("*", nil, "any", KindConfig, NoopValueValidator),
// 		}
// 		opts, _ := New("test", specs...)
// 		err := opts.Seal()
// 		if err != nil {
// 			t.Fatalf("Seal with wildcard failed: %v", err)
// 		}
// 		if !opts.sealed {
// 			t.Error("Seal: expected sealed to be true")
// 		}
// 		if opts.Get("key1").String() != "default1" {
// 			t.Errorf("Seal: got %q, want %q", opts.Get("key1").String(), "default1")
// 		}
// 	})

// 	t.Run("set default error", func(t *testing.T) {
// 		// Hit: error in opts.Set(key, cnf.value)
// 		specs := []Spec{
// 			NewOption("key1", "invalid", "desc1", KindConfig, func(arg Arg) error {
// 				return fmt.Errorf("%w: invalid default value", ErrOptionValidation)
// 			}),
// 		}
// 		opts, err := New("test", specs...)
// 		testutils.Error(t, err)

// 		err = opts.Seal()
// 		if err == nil || !errors.Is(err, ErrOptionValidation) {
// 			t.Errorf("Seal: expected ErrOptionValidation, got %v", err)
// 		}
// 		if opts.sealed {
// 			t.Error("Seal: expected sealed to be false on error")
// 		}
// 	})
// }

// func TestConcurrentAccess(t *testing.T) {
// 	opts, _ := New("test")
// 	testutils.NoError(t, opts.Add(NewOption("key1", "default", "desc", KindConfig, NoopValueValidator)))
// 	var wg sync.WaitGroup

// 	// Test concurrent writes
// 	for i := range 100 {
// 		wg.Add(1)
// 		go func(i int) {
// 			defer wg.Done()
// 			err := opts.Set("key1", fmt.Sprintf("value%d", i))
// 			if err != nil {
// 				t.Errorf("Concurrent Set failed: %v", err)
// 			}
// 		}(i)
// 	}
// 	wg.Wait()

// 	// Test concurrent reads
// 	for range 100 {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			_ = opts.Get("key1").String()
// 		}()
// 	}
// 	wg.Wait()
// }

// func TestOptions_MergeOptions(t *testing.T) {
// 	t.Run("valid merge", func(t *testing.T) {
// 		opts1, _ := New("dest", NewOption("key1", "val1", "desc1", KindConfig, NoopValueValidator))
// 		opts2, _ := New("src", NewOption("key2", "val2", "desc2", KindConfig, NoopValueValidator))

// 		err := MergeOptions(opts1, opts2)
// 		if err != nil {
// 			t.Fatalf("MergeOptions failed: %v", err)
// 		}
// 		if !opts1.Has("src.key2") {
// 			t.Error("MergeOptions: expected src.key2 to exist")
// 		}
// 		if opts1.Get("src.key2").String() != "val2" {
// 			t.Errorf("MergeOptions: got %q, want %q", opts1.Get("src.key2").String(), "val2")
// 		}

// 		// Test merging into sealed options
// 		testutils.NoError(t, opts1.Seal())
// 		err = MergeOptions(opts1, opts2)
// 		if !errors.Is(err, ErrOption) {
// 			t.Errorf("Merge into sealed: expected ErrOption, got %v", err)
// 		}
// 	})
// 	t.Run("invalid merge", func(t *testing.T) {
// 		opts1, _ := New("dest", NewOption("key1", "val1", "desc1", KindConfig, NoopValueValidator))
// 		opts2, _ := New("src", NewOption("key2", vars.NilValue, "desc", KindConfig, NoopValueValidator))

// 		testutils.Error(t, MergeOptions(opts1, opts2))

// 		if !opts1.Has("key1") {
// 			t.Error("MergeOptions: expected key1 to exist")
// 		}
// 	})
// }

// func TestOptions_Range(t *testing.T) {
// 	specs := []Spec{
// 		NewOption("key1", "val1", "desc1", KindConfig, NoopValueValidator),
// 		NewOption("key2", "val2", "desc2", KindConfig, NoopValueValidator),
// 	}
// 	opts, _ := New("test", specs...)

// 	keys := []string{}
// 	opts.Range(func(opt Option) bool {
// 		keys = append(keys, opt.Name())
// 		return true
// 	})
// 	slices.Sort(keys)
// 	if !slices.Equal(keys, []string{"key1", "key2"}) {
// 		t.Errorf("Range: got %v, want %v", keys, []string{"key1", "key2"})
// 	}
// }

// func TestOptions_Accepts(t *testing.T) {
// 	opts, _ := New("test",
// 		NewOption("key1", "val1", "desc1", KindConfig, NoopValueValidator),
// 		NewOption("*", nil, "any", KindConfig, NoopValueValidator),
// 	)

// 	if !opts.Accepts("key1") {
// 		t.Error("Accepts: expected key1 to be accepted")
// 	}
// 	if !opts.Accepts("random") {
// 		t.Error("Accepts: expected random key to be accepted with wildcard")
// 	}

// 	opts, _ = New("test", NewOption("key1", "val1", "desc1", KindConfig, NoopValueValidator))
// 	if opts.Accepts("random") {
// 		t.Error("Accepts: expected random key to be rejected without wildcard")
// 	}
// }

// func TestOptions_WithPrefix(t *testing.T) {
// 	opts, _ := New("test",
// 		NewOption("prefix.key1", "val1", "desc1", KindConfig, NoopValueValidator),
// 		NewOption("prefix.key2", "val2", "desc2", KindConfig, NoopValueValidator),
// 		NewOption("other.key3", "val3", "desc3", KindConfig, NoopValueValidator),
// 	)

// 	m := opts.WithPrefix("prefix.")
// 	if m.Len() != 2 {
// 		t.Errorf("WithPrefix: got %d keys, want %d", m.Len(), 2)
// 	}
// 	if !m.Has("key1") || !m.Has("key2") {
// 		t.Error("WithPrefix: expected key1 and key2 to exist")
// 	}
// }

// func TestOptions_Describe(t *testing.T) {
// 	// Setup options with a single spec
// 	specs := []Spec{
// 		NewOption("key1", "default1", "description for key1", KindConfig, NoopValueValidator),
// 	}
// 	opts, err := New("test", specs...)
// 	if err != nil {
// 		t.Fatalf("New failed: %v", err)
// 	}

// 	// Test existing key
// 	desc := opts.Describe("key1")
// 	if desc != "description for key1" {
// 		t.Errorf("Describe: got %q, want %q", desc, "description for key1")
// 	}

// 	// Test non-existent key
// 	desc = opts.Describe("nonexistent")
// 	if desc != "" {
// 		t.Errorf("Describe: got %q, want %q", desc, "")
// 	}
// }

// func TestOptions_Load(t *testing.T) {
// 	// Setup options with a single spec
// 	specs := []Spec{
// 		NewOption("key1", "value1", "desc1", KindConfig, NoopValueValidator),
// 	}
// 	opts, err := New("test", specs...)
// 	if err != nil {
// 		t.Fatalf("New failed: %v", err)
// 	}

// 	// Test loading existing key
// 	v, loaded := opts.Load("key1")
// 	if !loaded {
// 		t.Error("Load: expected key1 to exist")
// 	}
// 	if v.String() != "value1" {
// 		t.Errorf("Load: got %q, want %q", v.String(), "value1")
// 	}

// 	// Test loading non-existent key
// 	v, loaded = opts.Load("nonexistent")
// 	if loaded {
// 		t.Error("Load: expected nonexistent key to return false")
// 	}
// 	if v != (vars.Variable{}) {
// 		t.Errorf("Load: got %v, want zero value for non-existent key", v)
// 	}
// }

// func TestOptions_set(t *testing.T) {
// 	t.Run("invalid value", func(t *testing.T) {
// 		opts, _ := New("test",
// 			NewOption("invalid_opt", "default", "desc", KindConfig, NoopValueValidator),
// 		)
// 		err := opts.Set("invalid_opt", vars.NilValue)
// 		if err == nil || !errors.Is(err, vars.ErrValueInvalid) {
// 			t.Errorf("set: expected vars.ErrValueInvalid for invalid value, got %v", err)
// 		}
// 	})

// 	t.Run("readonly variable", func(t *testing.T) {
// 		opts, _ := New("test",
// 			NewOption("key1", "default", "desc", KindConfig, NoopValueValidator),
// 		)

// 		// Hit: cnf.kind |= KindReadOnly
// 		readonlyVar, _ := vars.New("key1", "readonly", true)
// 		err := opts.Set("key1", readonlyVar)
// 		if err != nil {
// 			t.Errorf("set: expected no error for readonly variable, got %v", err)
// 		}
// 		if !opts.Get("key1").ReadOnly() {
// 			t.Error("set: expected key1 to be readonly")
// 		}
// 	})

// 	t.Run("wildcard config", func(t *testing.T) {
// 		opts, _ := New("test",
// 			NewOption("*", nil, "any", KindConfig, NoopValueValidator),
// 		)

// 		// Hit: else if c, ok := opts.config["*"]
// 		err := opts.Set("random", "value1")
// 		if err != nil {
// 			t.Errorf("set: expected no error with wildcard config, got %v", err)
// 		}
// 		if val := opts.Get("random").String(); val != "value1" {
// 			t.Errorf("set: got %q, want %q", val, "value1")
// 		}
// 	})
// }

// func TestOptions_Add_InvalidKey(t *testing.T) {
// 	opts, err := New("test")
// 	if err != nil {
// 		t.Fatalf("New failed: %v", err)
// 	}

// 	// Test invalid key to hit vars.ParseKey error
// 	invalidKey := "" // Empty key, assuming vars.ParseKey rejects it
// 	spec := NewOption(invalidKey, "value", "desc", KindConfig, NoopValueValidator)
// 	err = opts.Add(spec)
// 	if err == nil {
// 		t.Fatal("Add: expected error for invalid key, got nil")
// 	}
// 	if !errors.Is(err, ErrOption) {
// 		t.Errorf("Add: expected ErrOption, got %v", err)
// 	}
// 	expectedMsg := fmt.Sprintf(
// 		"%s \nkey error: key was empty string",
// 		fmt.Errorf("%w(test): invalid key", ErrOption).Error(),
// 	)
// 	if err.Error() != expectedMsg {
// 		t.Errorf("Add: got error %q, want %q", err.Error(), expectedMsg)
// 	}
// }

// func TestArg_Methods(t *testing.T) {
// 	t.Run("string value", func(t *testing.T) {
// 		// Test NewArg, Key, and Value with a string
// 		arg := NewArg("key1", "value1")
// 		if key := arg.Key(); key != "key1" {
// 			t.Errorf("Key: got %q, want %q", key, "key1")
// 		}
// 		if val := arg.Value(); val.String() != "value1" {
// 			t.Errorf("Value: got %v, want %v", val, "value1")
// 		}
// 	})

// 	t.Run("non-string value", func(t *testing.T) {
// 		// Test NewArg, Key, and Value with a non-string (int)
// 		arg := NewArg("key2", 42)
// 		if key := arg.Key(); key != "key2" {
// 			t.Errorf("Key: got %q, want %q", key, "key2")
// 		}
// 		if val := arg.Value(); val.String() != "42" {
// 			t.Errorf("Value: got %v, want %v", val, 42)
// 		}
// 	})

// 	t.Run("empty key", func(t *testing.T) {
// 		// Test NewArg with empty key
// 		arg := NewArg("", nil)
// 		if key := arg.Key(); key != "" {
// 			t.Errorf("Key: got %q, want %q", key, "")
// 		}
// 		if val := arg.Value(); val.String() != "<nil>" {
// 			t.Errorf("Value: got %v, want %v", val, nil)
// 		}
// 	})
// }

// func TestOption_Methods(t *testing.T) {
// 	// Setup options with different kinds and readonly settings
// 	specs := []Spec{
// 		NewOption("key1", "value1", "desc1", KindConfig, NoopValueValidator),
// 		NewOption("key2", 42, "desc2", KindReadOnly, NoopValueValidator),
// 		NewOption("key3", true, "desc3", KindConfig, NoopValueValidator),
// 	}
// 	opts, err := New("test", specs...)
// 	if err != nil {
// 		t.Fatalf("New failed: %v", err)
// 	}

// 	// Collect options via Range
// 	options := make(map[string]Option)
// 	opts.Range(func(opt Option) bool {
// 		options[opt.Name()] = opt
// 		return true
// 	})

// 	// Test Value
// 	t.Run("Value", func(t *testing.T) {
// 		if val := options["key1"].Value().String(); val != "value1" {
// 			t.Errorf("Value: got %q, want %q", val, "value1")
// 		}
// 		val2, err := options["key2"].Value().Int()
// 		testutils.NoError(t, err)
// 		if val2 != 42 {
// 			t.Errorf("Value: got %d, want %d", val2, 42)
// 		}
// 		val3, err := options["key3"].Value().Bool()
// 		testutils.NoError(t, err)
// 		if val3 != true {
// 			t.Errorf("Value: got %v, want %v", val3, true)
// 		}
// 	})

// 	// Test ReadOnly
// 	t.Run("ReadOnly", func(t *testing.T) {
// 		if options["key1"].ReadOnly() {
// 			t.Error("ReadOnly: expected key1 to be non-readonly")
// 		}
// 		if !options["key2"].ReadOnly() {
// 			t.Error("ReadOnly: expected key2 to be readonly")
// 		}
// 		if options["key3"].ReadOnly() {
// 			t.Error("ReadOnly: expected key3 to be non-readonly")
// 		}
// 	})

// 	// Test Kind
// 	t.Run("Kind", func(t *testing.T) {
// 		if kind := options["key1"].Kind(); kind != vars.KindString {
// 			t.Errorf("Kind: got %v, want %v", kind, vars.KindString)
// 		}
// 		if kind := options["key2"].Kind(); kind != vars.KindInt {
// 			t.Errorf("Kind: got %v, want %v", kind, vars.KindInt)
// 		}
// 		if kind := options["key3"].Kind(); kind != vars.KindBool {
// 			t.Errorf("Kind: got %v, want %v", kind, vars.KindBool)
// 		}
// 	})
// }
