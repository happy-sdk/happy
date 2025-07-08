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

	value, err := vars.NewValue(opt.value)
	if err != nil {
		return fmt.Errorf("%w(%s): failed to create value for key %s: %s", ErrOption, spec.opts.name, key, err.Error())
	}

	variable, err := vars.New(key, value, opt.flags&ReadOnly != 0)
	if err != nil {
		return fmt.Errorf("%w(%s): failed to create default variable for key %s: %s", ErrOption, spec.opts.name, key, err.Error())
	}

	option := newOption(variable, variable, false, opt.flags, opt.desc)

	if opt.parser != nil {
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

	if opt.validator != nil {
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
		if other.sealed {
			return fmt.Errorf("%w: %s already sealed", ErrOptions, other.opts.name)
		}

		for key, opt := range other.opts.db {
			skey := strings.Join([]string{other.opts.name, key}, ".")

			if _, ok := spec.opts.db[skey]; ok {
				return fmt.Errorf("%w(%s): key %s already exists", ErrOption, spec.opts.name, skey)
			}

			spec.opts.db[skey] = opt.withKey(skey)
		}

		for key, parser := range other.opts.parsers {
			skey := strings.Join([]string{other.opts.name, key}, ".")
			if _, ok := spec.opts.parsers[skey]; ok {
				return fmt.Errorf("%w(%s): parser for key %s already exists", ErrOption, spec.opts.name, skey)
			}
			spec.opts.parsers[skey] = parser
		}

		for key, validator := range other.opts.validators {
			skey := strings.Join([]string{other.opts.name, key}, ".")
			if _, ok := spec.opts.validators[skey]; ok {
				return fmt.Errorf("%w(%s): validator for key %s already exists", ErrOption, spec.opts.name, skey)
			}
			spec.opts.validators[skey] = validator
		}

		for _, wildcard := range other.opts.wildcards {
			swildcard := strings.Join([]string{other.opts.name, wildcard}, ".")
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
	if len(opts.wildcards) == 0 {
		return false
	}
	for _, wildcard := range opts.wildcards {
		if wildcard == "" || strings.HasPrefix(key, wildcard) {
			return true
		}
	}
	return true
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

///////////////////////////////////////////////////
///////////////////////////////////////////////////
///////////////////////////////////////////////////
///////////////////////////////////////////////////

// func (opts *Options) set(key string, value any, override bool) error {
// 	if key == "*" {
// 		return nil
// 	}

// 	if !opts.Accepts(key) {
// 		return fmt.Errorf(
// 			"%w: %s does not accept option %s",
// 			ErrOption,
// 			opts.name,
// 			key,
// 		)
// 	}
// 	// Check is readonly
// 	if opts.sealed && opts.db.Get(key).ReadOnly() {
// 		if !override {
// 			return fmt.Errorf(
// 				"%w: can not set %s for %s, (opts sealed %t)",
// 				ErrOptionReadOnly,
// 				key,
// 				opts.name,
// 				opts.sealed,
// 			)
// 		}
// 		// remove old readonly option
// 	}

// 	if override {
// 		opts.db.Delete(key)
// 	}

// 	val, err := vars.NewValue(value)
// 	if err != nil {
// 		return err
// 	}

// 	var cnf *OptionSpec
// 	if c, ok := opts.config[key]; ok {
// 		cnf = &c
// 		if vv, ok := value.(vars.Variable); ok {
// 			if vv.ReadOnly() {
// 				cnf.kind |= ReadOnly
// 			}
// 		}
// 	} else if c, ok := opts.config["*"]; ok {
// 		cnf = &c
// 	}

// 	ro := cnf.kind&ReadOnly != 0
// 	opt := Option{key: key, val: val, ro: ro}
// 	if opts.sealed {
// 		if cnf.parser != nil {
// 			if newval, err := cnf.parser(opt); err != nil {
// 				return err
// 			} else {
// 				opt.val = newval
// 			}
// 		}
// 		if cnf.validator != nil {
// 			if err := cnf.validator(opt); err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	return opts.db.StoreReadOnly(key, val, cnf.kind&ReadOnly != 0)
// }

// func (opts *Options) Set(key string, value any) error {
// 	return opts.set(key, value, !opts.sealed)
// }

// // WithPrefix returns a new vars.Map with all options that have the given prefix.
// func (opts *Options) WithPrefix(prefix string) *vars.Map {
// 	return opts.db.ExtractWithPrefix(prefix)
// }

// // MergeOptions merges options from one or multiplce sources to dest.
// func MergeOptions(dest *Options, srcs ...*Options) error {
// 	if dest.sealed {
// 		var sources []string
// 		for _, src := range srcs {
// 			sources = append(sources, src.name)
// 		}
// 		return fmt.Errorf("%w: can not add %s options to sealed destination %s", ErrOption, strings.Join(sources, ","), dest.name)
// 	}

// 	for _, src := range srcs {
// 		for _, spec := range src.config {
// 			spec.key = src.name + "." + spec.key
// 			if err := dest.Add(spec); err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	return nil
// }
