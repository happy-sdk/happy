// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

// Package options provides a flexible framework for defining and managing
// configuration options for application components in the Happy SDK or
// custom libraries. It supports type-safe option storage (each value backed
// by vars.Value), validation, parsing, and sealing, with features like
// read-only options, default values, wildcard-accepted keys, and merging
// option sets together via Spec.Extend (which namespaces the merged-in
// spec's keys under its own name, e.g. "other.key").
package options

import (
	"errors"
	"fmt"
	"iter"
	"sort"
	"strings"
	"sync"

	"github.com/happy-sdk/happy/pkg/vars"
)

type (
	Spec struct {
		mu     sync.RWMutex
		sealed bool
		opts   *Options
	}

	Value = vars.Value

	ValueValidator func(opt Option) error
	ValueParser    func(opt Option, newval Value) (parsed Value, err error)

	// Flags is a bitmask for option kind. It defines option behavior.
	Flag  uint
	Flags = Flag
)

const (
	Mutable Flag = 1 << iota
	ReadOnly
	Once
)

const UndefinedOptionDescription = "undefined"

var (
	ErrOptions          = errors.New("option error")
	ErrOption           = fmt.Errorf("%w", ErrOptions)
	ErrOptionNotExists  = fmt.Errorf("%w: no such option", ErrOption)
	ErrOptionReadOnly   = fmt.Errorf("%w: readonly option", ErrOption)
	ErrOptionOnce       = fmt.Errorf("%w: already set", ErrOption)
	ErrOptionValidation = fmt.Errorf("%w: validation failed", ErrOption)
)

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// OptionNoopValueValidator provides no validation for option value.
var OptionNoopValueValidator = func(opt Option, newval Value) error {
	return nil
}

// OptionValidatorNotEmpty validates that option value is not empty.
var OptionValidatorNotEmpty = func(opt Option, newval Value) error {
	if opt.Value().Empty() {
		return fmt.Errorf("%w: %s value can not be empty", ErrOption, opt.Key())
	}
	return nil
}

func New(name string, opts ...*OptionSpec) (spec *Spec, err error) {
	name, err = vars.ParseKey(name)
	if err != nil {
		return nil, fmt.Errorf("%w: name %q is invalid, err: %w", ErrOptions, name, err)
	}
	spec = &Spec{
		opts: &Options{
			name:       name,
			db:         make(map[string]*Option),
			parsers:    make(map[string]ValueParser),
			validators: make(map[string]ValueValidator),
		},
	}
	if err := spec.Add(opts...); err != nil {
		return nil, err
	}
	return spec, nil
}

func NewValue(v any) (Value, error) {
	val, err := vars.NewValue(v)
	if err != nil {
		return vars.EmptyValue, fmt.Errorf("%w: %s", ErrOption, err)
	}
	return val, nil
}

func (spec *Spec) Add(opts ...*OptionSpec) error {
	spec.mu.Lock()
	defer spec.mu.Unlock()

	if spec.sealed {
		return fmt.Errorf("%w(%s): sealed, can not add options", ErrOption, spec.opts.name)
	}

	for _, opt := range opts {
		if err := spec.add(opt); err != nil {
			return err
		}
	}
	return nil
}

// Add adds a new option to the Options set.
// It returns an error if the option is already set or set is sealed.
func (spec *Spec) add(opt *OptionSpec) error {

	key, err := vars.ParseKey(opt.key)
	if err != nil {
		return errors.Join(fmt.Errorf("%w(%s): invalid key %s", ErrOption, spec.opts.name, opt.key), err)
	}

	if _, set := spec.opts.db[key]; set {
		return fmt.Errorf("%w(%s): duplicated key %s", ErrOption, spec.opts.name, key)
	}

	hasDefault := opt.value != nil
	if !hasDefault {
		opt.value = ""
	}
	value, err := vars.NewValue(opt.value)
	if err != nil {
		return fmt.Errorf("%w(%s): failed to create value for key %s: %s", ErrOption, spec.opts.name, key, err.Error())
	}

	variable, err := vars.New(key, value, opt.flags&ReadOnly != 0)
	if err != nil {
		return fmt.Errorf("%w(%s): failed to create default variable for key %s: %s", ErrOption, spec.opts.name, key, err.Error())
	}

	option := newOption(variable, variable, false, opt.flags, opt.desc)

	if hasDefault && opt.parser != nil {
		parser := opt.parser
		spec.opts.parsers[key] = parser
		value, err = parser(*option, value)
		if err != nil {
			return err
		}
		variable, err = vars.New(key, value, opt.flags&ReadOnly != 0)
		if err != nil {
			return fmt.Errorf("%w(%s): failed to create default parsed variable for key %s: %s", ErrOption, spec.opts.name, key, err.Error())
		}
	}

	if hasDefault && opt.validator != nil {
		validator := opt.validator
		spec.opts.validators[key] = validator
		if err := validator(*option); err != nil {
			return fmt.Errorf("%w(%s): failed to validate default value for key %s: %w", ErrOption, spec.opts.name, key, err)
		}
	}
	option.def = variable
	option.v = variable
	spec.opts.db[key] = option

	return nil
}

func (spec *Spec) Extend(others ...*Spec) error {
	spec.mu.Lock()
	defer spec.mu.Unlock()
	if spec.sealed {
		return fmt.Errorf("%w: %s already sealed", ErrOptions, spec.opts.name)
	}

	for _, other := range others {
		if other == nil {
			return fmt.Errorf("%w: attempt to extend %s with nil spec", ErrOptions, spec.opts.name)
		}

		// Snapshot everything needed from other while holding its own
		// locks, rather than reading its fields directly: other is a
		// different *Spec than the receiver, guarded by its own mu (and
		// other.opts by its own, separate mu), so reading them without
		// acquiring those locks raced with any concurrent call that
		// mutates other (e.g. other.Set).
		other.mu.RLock()
		otherSealed := other.sealed
		other.mu.RUnlock()
		if otherSealed {
			return fmt.Errorf("%w: %s already sealed", ErrOptions, other.opts.name)
		}

		other.opts.mu.RLock()
		otherName := other.opts.name
		dbCopy := make(map[string]*Option, len(other.opts.db))
		for k, v := range other.opts.db {
			dbCopy[k] = v
		}
		parsersCopy := make(map[string]ValueParser, len(other.opts.parsers))
		for k, v := range other.opts.parsers {
			parsersCopy[k] = v
		}
		validatorsCopy := make(map[string]ValueValidator, len(other.opts.validators))
		for k, v := range other.opts.validators {
			validatorsCopy[k] = v
		}
		wildcardsCopy := append([]string(nil), other.opts.wildcards...)
		other.opts.mu.RUnlock()

		for key, opt := range dbCopy {
			skey := strings.Join([]string{otherName, key}, ".")

			if _, ok := spec.opts.db[skey]; ok {
				return fmt.Errorf("%w(%s): key %s already exists", ErrOption, spec.opts.name, skey)
			}

			spec.opts.db[skey] = opt.withKey(skey)
		}

		for key, parser := range parsersCopy {
			skey := strings.Join([]string{otherName, key}, ".")
			if _, ok := spec.opts.parsers[skey]; ok {
				return fmt.Errorf("%w(%s): parser for key %s already exists", ErrOption, spec.opts.name, skey)
			}
			spec.opts.parsers[skey] = parser
		}

		for key, validator := range validatorsCopy {
			skey := strings.Join([]string{otherName, key}, ".")
			if _, ok := spec.opts.validators[skey]; ok {
				return fmt.Errorf("%w(%s): validator for key %s already exists", ErrOption, spec.opts.name, skey)
			}
			spec.opts.validators[skey] = validator
		}

		for _, wildcard := range wildcardsCopy {
			swildcard := strings.Join([]string{otherName, wildcard}, ".")
			spec.opts.wildcards = append(spec.opts.wildcards, swildcard)
		}
	}

	return nil
}
func (spec *Spec) Set(key string, val any) error {
	spec.mu.Lock()
	defer spec.mu.Unlock()
	if spec.sealed {
		return fmt.Errorf("%w(%s): %s already sealed", ErrOptions, spec.opts.name, key)
	}
	return spec.opts.Set(key, val)
}

func (spec *Spec) Accepts(key string) bool {
	spec.mu.RLock()
	defer spec.mu.RUnlock()
	return spec.opts.Accepts(key)
}

func (spec *Spec) AllowWildcard() {
	spec.mu.Lock()
	defer spec.mu.Unlock()
	spec.opts.wildcards = append(spec.opts.wildcards, "")
}

func (spec *Spec) Get(key string) Option {
	spec.mu.RLock()
	defer spec.mu.RUnlock()
	return spec.opts.Get(key)
}

func (spec *Spec) Seal() (*Options, error) {
	spec.mu.Lock()
	defer spec.mu.Unlock()

	if spec.sealed {
		return nil, fmt.Errorf("%w: %s already sealed", ErrOptions, spec.opts.name)
	}

	spec.sealed = true

	for _, opt := range spec.opts.db {
		if opt.set {
			continue
		}
		opt.v = opt.def
	}

	return spec.opts, nil
}

type Options struct {
	mu         sync.RWMutex
	name       string
	db         map[string]*Option
	parsers    map[string]ValueParser
	validators map[string]ValueValidator
	wildcards  []string
}

// Name returns name for this Option collection.
// It is alias for String().
func (opts *Options) Name() string {
	return opts.String()
}

// String returns name and satisfies fmt.Stringer interface.
func (opts *Options) String() string {
	opts.mu.RLock()
	defer opts.mu.RUnlock()
	return opts.name
}

func (opts *Options) IsSet(key string) bool {
	opts.mu.RLock()
	defer opts.mu.RUnlock()
	opt, ok := opts.db[key]
	return ok && opt.IsSet()
}

// Accepts reports whether given option key is accepted by Options.
func (opts *Options) Accepts(key string) bool {
	opts.mu.RLock()
	defer opts.mu.RUnlock()
	if _, ok := opts.db[key]; ok {
		return true
	}
	for _, wildcard := range opts.wildcards {
		if wildcard == "" || strings.HasPrefix(key, wildcard) {
			return true
		}
	}
	return false
}

func (opts *Options) Describe(key string) string {
	opts.mu.RLock()
	defer opts.mu.RUnlock()
	if opt, ok := opts.db[key]; ok {
		return opt.Description()
	}
	return UndefinedOptionDescription
}

// Get returns the Option stored in the Options for a key,
// It returns empty Option if key is not found.
// Use Load to check if Option by key was found in the Options.
func (opts *Options) Get(key string) Option {
	opt, _ := opts.Load(key)
	return opt
}

// Load returns the Option stored in the Options for a key,
// The loaded result indicates whether Option by key was found in the Options.
func (opts *Options) Load(key string) (opt Option, loaded bool) {
	opts.mu.RLock()
	defer opts.mu.RUnlock()
	opt, loaded = opts.load(key)
	return
}

func (opts *Options) load(key string) (opt Option, loaded bool) {
	o, ok := opts.db[key]
	if !ok {
		opt.v = vars.EmptyVariable
		opt.desc = UndefinedOptionDescription
		return
	}

	opt = *o
	return opt, true
}

func (opts *Options) Range(cb func(opt Option) bool) {
	opts.mu.RLock()
	defer opts.mu.RUnlock()

	keys := make([]string, 0, len(opts.db))
	for k := range opts.db {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	for _, k := range keys {
		opt := opts.Get(k)
		if !cb(opt) {
			return
		}
	}
}

func (opts *Options) All() iter.Seq[Option] {
	opts.mu.RLock()
	defer opts.mu.RUnlock()
	keys := make([]string, 0, len(opts.db))
	for k := range opts.db {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return func(yield func(Option) bool) {
		for _, k := range keys {
			if !yield(opts.Get(k)) {
				return
			}
		}
	}
}

func (opts *Options) Set(key string, val any) error {
	opts.mu.Lock()
	defer opts.mu.Unlock()

	mut, loaded := opts.db[key]
	if !loaded {
		matched := false
		for _, wildcard := range opts.wildcards {
			if wildcard == "" || strings.HasPrefix(key, wildcard) {
				mut = newOption(vars.EmptyVariable, vars.EmptyVariable, false, 0, "")
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%w: %s", ErrOptionNotExists, key)
		}
	}

	if mut.ReadOnly() {
		return fmt.Errorf("%w: %s", ErrOptionReadOnly, key)
	}

	if mut.set && mut.flags&Once != 0 {
		return fmt.Errorf("%w: for %s can not mutate", ErrOptionOnce, key)
	}

	value, err := NewValue(val)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}

	opt := *mut

	variable, err := vars.New(key, value, opt.flags&ReadOnly != 0)
	if err != nil {
		return fmt.Errorf("%w(%s): failed to create variable for key %s: %s", ErrOption, opts.name, key, err.Error())
	}

	if parser, exists := opts.parsers[key]; exists {
		value, err = parser(opt, value)
		if err != nil {
			return fmt.Errorf("%w: %s", err, key)
		}
		variable, err = vars.New(key, value, mut.flags&ReadOnly != 0)
		if err != nil {
			return fmt.Errorf("%w(%s): failed to create parsed variable for key %s: %s", ErrOption, opts.name, key, err.Error())
		}
	}

	opt.v = variable

	if validator, exists := opts.validators[key]; exists {
		if err := validator(opt); err != nil {
			return fmt.Errorf("%w: %s", err, key)
		}
	}
	opt.set = true
	opts.db[key] = &opt

	return nil
}

func (opts *Options) Len() int {
	opts.mu.RLock()
	defer opts.mu.RUnlock()
	return len(opts.db)
}

func newOption(v vars.Variable, def vars.Variable, set bool, flags Flags, desc string) *Option {
	if flags == 0 {
		flags = Mutable
	}
	return &Option{
		v:     v,
		def:   def,
		set:   set,
		flags: flags,
		desc:  desc,
	}
}

type Option struct {
	v     vars.Variable
	def   vars.Variable
	set   bool
	desc  string
	flags Flags
}

// Key returns the key of the option.
func (o Option) Key() string {
	return o.v.Name()
}

// Value returns the value of the option.
func (o Option) Value() Value {
	return o.v.Value()
}

func (o Option) Default() Value {
	return o.def.Value()
}

// ReadOnly returns true if the option is read-only.
func (o Option) ReadOnly() bool {
	return o.flags&ReadOnly != 0
}

// Description returns the description of the option.
func (o Option) Description() string {
	return o.desc
}

func (o Option) Variable() vars.Variable {
	if o.set {
		return o.v
	}
	return o.def
}

func (o Option) IsSet() bool {
	return o.set
}

func (o Option) HasFlag(flag Flag) bool {
	return o.flags&flag != 0
}

func (o Option) String() string {
	return o.v.String()
}

func (o *Option) withKey(key string) *Option {
	var err error
	opt := *o
	opt.v, err = opt.v.WithName(key)
	if err != nil {
		return o
	}
	opt.def, err = opt.def.WithName(key)
	if err != nil {
		return o
	}
	return &opt
}

type OptionSpec struct {
	_         noCopy
	key       string
	desc      string
	value     any
	flags     Flags
	validator ValueValidator
	parser    ValueParser
}

func NewOption(key string, defval any) *OptionSpec {
	return &OptionSpec{
		key:   key,
		value: defval,
	}
}

func (o *OptionSpec) Description(desc string) *OptionSpec {
	o.desc = desc
	return o
}

func (o *OptionSpec) Parser(parser ValueParser) *OptionSpec {
	o.parser = parser
	return o
}

func (o *OptionSpec) Validator(validator ValueValidator) *OptionSpec {
	o.validator = validator
	return o
}

func (o *OptionSpec) Flags(flags Flags) *OptionSpec {
	o.flags = flags
	return o
}

type Arg struct {
	key   string
	value any
}

func NewArg(key string, value any) *Arg {
	return &Arg{
		key:   key,
		value: value,
	}
}

func (arg *Arg) Key() string {
	return arg.key
}

func (arg *Arg) Value() any {
	return arg.value
}
