// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

// Package options provides a flexible framework for defining and managing
// configuration options for application components in the Happy SDK or custom libraries.
// It supports type-safe option storage, validation, and sealing, with features like readonly
// options, default values, and prefix-based extraction. Options are backed by
// vars.Map for thread-safe access and integrate with application components via
// named option sets.
package options

import (
	"errors"
	"fmt"
	"strings"

	"github.com/happy-sdk/happy/pkg/vars"
)

type (

	// Options is general collection of options
	// attached to specific application component.
	Options struct {
		name   string
		db     vars.Map
		config map[string]Spec
		sealed bool
	}

	// Spec holds specification for given option.
	Spec struct {
		key       string
		desc      string
		value     any // default
		kind      Kind
		validator ValueValidator
	}

	// Kind is a bitmask for option kind. It defines option behavior.
	Kind uint

	// OptionValueValidator is callback function to validate
	// given value, it recieves copy of value for validation.
	// It MUST return error if validation fails, returned
	// boolean indicates shoulkd that option be marked
	// as radonly if validation succeeds.
	ValueValidator func(key string, val vars.Value) error

	Arg struct {
		key   string
		value any
	}

	Option struct {
		val vars.Variable
	}
)

const (
	KindRuntime Kind = 1 << iota
	KindReadOnly
	KindConfig
)

var (
	ErrOption           = errors.New("option error")
	ErrOptionReadOnly   = fmt.Errorf("%w: readonly option", ErrOption)
	ErrOptionValidation = fmt.Errorf("%w: validation failed", ErrOption)
)

// NewOption returns new option specification with given key, value, description and validator.
func NewOption(key string, dval any, desc string, kind Kind, vfunc ValueValidator) Spec {
	return Spec{
		key:       key,
		value:     dval,
		desc:      desc,
		kind:      kind,
		validator: vfunc,
	}
}

// func (o OptionSpec) apply(opts *Options) error {
// 	return opts.Set(o.key, o.value)
// }

// New returns new named options set.
// *Options is quarantined to be not nil even when errors occur during initialization.
func New(name string, specs []Spec) (*Options, error) {
	opts := &Options{
		name: name,
	}
	for _, spec := range specs {
		if err := opts.Add(spec); err != nil {
			return opts, err
		}
	}
	return opts, nil
}

// Accepts reports whether given option key is accepted by Options.
func (opts *Options) Accepts(key string) bool {
	if opts.config == nil {
		return false
	}
	if _, ok := opts.config["*"]; ok {
		return true
	}
	_, ok := opts.config[key]
	return ok
}

// Name is name for this Option collection.
func (opts *Options) Name() string {
	return opts.name
}

// String is alias for Name and to satisfy fmt.Stringer interface.
func (opts *Options) String() string {
	return opts.name
}

func (opts *Options) Describe(key string) string {
	c, ok := opts.config[key]
	if !ok {
		return ""
	}
	return c.desc
}

var emptyStringVariable, _ = vars.New("empty", "", true)

func (opts *Options) Get(key string) vars.Variable {
	if opts.db.Has(key) {
		return opts.db.Get(key)
	}
	return emptyStringVariable
}

// Load returns the vars.Variable stored in the Options for a key,
// or vars.EmptyVar if no key is present.
// The loaded result indicates whether Option by key was found in the Options.
func (opts *Options) Load(key string) (v vars.Variable, loaded bool) {
	v, loaded = opts.db.Load(key)
	return
}

func (opts *Options) Range(fn func(opt Option) bool) {
	opts.db.Range(func(v vars.Variable) bool {
		return fn(Option{val: v})
	})
}

func (opts *Options) set(key string, value any, override bool) error {
	if key == "*" {
		return nil
	}

	if !opts.Accepts(key) {
		return fmt.Errorf(
			"%w: %s does not accept option %s",
			ErrOption,
			opts.name,
			key,
		)
	}
	// Check is readonly
	if opts.sealed && opts.db.Get(key).ReadOnly() {
		if !override {
			return fmt.Errorf(
				"%w: can not set %s for %s, (opts sealed %t)",
				ErrOptionReadOnly,
				key,
				opts.name,
				opts.sealed,
			)
		}
		// remove old readonly option
	}

	if override {
		opts.db.Delete(key)
	}

	val, err := vars.NewValue(value)
	if err != nil {
		return err
	}

	var cnf *Spec
	if c, ok := opts.config[key]; ok {
		cnf = &c
		if vv, ok := value.(vars.Variable); ok {
			if vv.ReadOnly() {
				cnf.kind |= KindReadOnly
			}
		}
	} else if c, ok := opts.config["*"]; ok {
		cnf = &c
	}
	if cnf.validator != nil {
		// validate
		if err := cnf.validator(key, val); err != nil {
			return err
		}
	}

	return opts.db.StoreReadOnly(key, val, cnf.kind&KindReadOnly != 0)
}

func (opts *Options) Set(key string, value any) error {
	return opts.set(key, value, !opts.sealed)
}

// Has reports whether options has given key
func (opts *Options) Has(key string) bool {
	return opts.db.Has(key)
}

// Len reports the number of options in Options set
func (opts *Options) Len() int {
	return opts.db.Len()
}

// Add adds a new option to the Options set.
// It returns an error if the option is already set or set is sealed.
func (opts *Options) Add(spec Spec) error {
	if opts.sealed {
		return fmt.Errorf("%w: can not add %s to already sealed %s options", ErrOption, spec.key, opts.name)
	}
	if opts.config == nil {
		opts.config = make(map[string]Spec)
	}
	key, err := vars.ParseKey(spec.key)
	if err != nil {
		return errors.Join(fmt.Errorf("%w(%s): invalid key %s", ErrOption, opts.name, spec.key), err)
	}
	if _, ok := opts.config[key]; ok {
		return fmt.Errorf("%w(%s): duplicated key %s", ErrOption, opts.name, key)
	}
	opts.config[key] = spec
	return opts.Set(spec.key, spec.value)
}

// WithPrefix returns a new vars.Map with all options that have the given prefix.
func (opts *Options) WithPrefix(prefix string) *vars.Map {
	return opts.db.ExtractWithPrefix(prefix)
}

// Seal ensures that all required options are set.
// It will set default values for options that are not set.
// It will return error if options are already sealed.
func (opts *Options) Seal() error {
	if opts.sealed {
		return fmt.Errorf("%w: %s already sealed", ErrOption, opts.name)
	}
	for key, cnf := range opts.config {
		if key == "*" {
			continue
		}
		if !opts.db.Has(key) {
			if err := opts.Set(key, cnf.value); err != nil {
				return err
			}
		}
	}
	opts.sealed = true
	return nil
}

// Noopvalidator provides no validation for option value.
var NoopValueValidator = func(key string, val vars.Value) error {
	return nil
}

// OptionValidatorNotEmpty validates that option value is not empty.
var OptionValidatorNotEmpty = func(key string, val vars.Value) error {
	if val.Len() == 0 {
		return fmt.Errorf("%w: %s value can not be empty", ErrOption, key)
	}
	return nil
}

// Key returns the key of the argument.
func (a Arg) Key() string {
	return a.key
}

// Value returns the value of the argument.
func (a Arg) Value() any {
	return a.value
}

// NewArg returns new Arg with given key and value.
// It is used to pass options to application components which accept options.
func NewArg(key string, value any) Arg {
	return Arg{
		key:   key,
		value: value,
	}
}

// Name returns the name of the option.
func (o Option) Name() string {
	return o.val.Name()
}

// Value returns the value of the option.
func (o Option) Value() vars.Value {
	return o.val.Value()
}

// ReadOnly returns true if the option is read-only.
func (o Option) ReadOnly() bool {
	return o.val.ReadOnly()
}

// Kind returns the vars.Kind of the option Value.
func (o Option) Kind() vars.Kind {
	return o.val.Kind()
}

// MergeOptions merges options from one or multiplce sources to dest.
func MergeOptions(dest *Options, srcs ...*Options) error {
	if dest.sealed {
		var sources []string
		for _, src := range srcs {
			sources = append(sources, src.name)
		}
		return fmt.Errorf("%w: can not add %s options to sealed destination %s", ErrOption, strings.Join(sources, ","), dest.name)
	}

	for _, src := range srcs {
		for _, spec := range src.config {
			spec.key = src.name + "." + spec.key
			if err := dest.Add(spec); err != nil {
				return err
			}
		}
	}

	return nil
}
