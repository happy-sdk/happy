// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package happy

import (
	"errors"
	"fmt"

	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/networking/address"
)

type (

	// Options is general collection of settings
	// attached to specific application component.
	Options struct {
		name   string
		db     vars.Map
		config map[string]OptionArg
	}

	// Option is used to define option and
	// apply given key value to options.
	OptionArg struct {
		key       string
		desc      string
		value     any // default
		kind      OptionKind
		validator OptionValueValidator
	}

	OptionKind uint

	// OptionValueValidator is callback function to validate
	// given value, it recieves copy of value for validation.
	// It MUST return error if validation fails, returned
	// boolean indicates shoulkd that option be marked
	// as radonly if validation succeeds.
	OptionValueValidator func(key string, val vars.Value) error
)

const (
	RuntimeOption OptionKind = 1 << iota
	ReadOnlyOption
	SettingsOption
	ConfigOption
)

var (
	ErrOption           = errors.New("option error")
	ErrOptionReadOnly   = fmt.Errorf("%w: readonly option", ErrOption)
	ErrOptionValidation = fmt.Errorf("%w: validation failed", ErrOption)
)

// Opt creates option for given key value pair
// which can be applied to any Options set.
func Option(key string, value any) OptionArg {
	return OptionArg{
		key:   key,
		value: value,
		kind:  RuntimeOption,
	}
}

func OptionReadonly(key string, value any) OptionArg {
	return OptionArg{
		key:   key,
		value: value,
		kind:  RuntimeOption | ReadOnlyOption,
	}
}

func (o OptionArg) apply(opts *Options) error {
	return opts.Set(o.key, o.value)
}

// NewOptions returns new named options with optiona validator when provided.
func NewOptions(name string, defaults []OptionArg) (*Options, error) {
	opts := &Options{
		name: name,
	}
	if len(defaults) > 0 {
		opts.config = make(map[string]OptionArg)
		for _, cnf := range defaults {
			key, err := vars.ParseKey(cnf.key)
			if err != nil {
				return nil, errors.Join(fmt.Errorf("%w: %s invalid key", ErrOption, name), err)
			}
			if _, ok := opts.config[key]; ok {
				return nil, fmt.Errorf("%w: %s duplicated key %s", ErrOption, name, key)
			}
			opts.config[key] = cnf
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

func (opts *Options) Load(key string) (vars.Variable, bool) {
	return opts.db.Load(key)
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
	if opts.db.Get(key).ReadOnly() {
		if !override {
			return fmt.Errorf(
				"%w: can not set %s for %s",
				ErrOptionReadOnly,
				key,
				opts.name,
			)
		}
		// remove old readonly option
		opts.db.Delete(key)
	}

	val, err := vars.NewValue(value)
	if err != nil {
		return err
	}

	// there is no validation required
	if opts.config == nil {
		return opts.db.Store(key, val)
	}

	var cnf *OptionArg
	if c, ok := opts.config[key]; ok {
		cnf = &c
		if vv, ok := value.(vars.Variable); ok {
			if vv.ReadOnly() {
				cnf.kind |= ReadOnlyOption
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

	return opts.db.StoreReadOnly(key, val, cnf.kind&ReadOnlyOption != 0)
}

func (opts *Options) Set(key string, value any) error {
	return opts.set(key, value, false)
}

// Has reports whether options has given key
func (opts *Options) Has(key string) bool {
	return opts.db.Has(key)
}

func (opts *Options) setDefaults() error {
	for key, cnf := range opts.config {
		if key == "*" {
			continue
		}
		if !opts.db.Has(key) {
			if err := opts.set(key, cnf.value, true); err != nil {
				return err
			}
		}
	}
	return nil
}

// noopvalidator is needed so that valid options would not falltrough to
// "*" case validator and fail since runtime options for "app." and "log."
// are not allowed.
var noopvalidator = func(key string, val vars.Value) error {
	return nil
}

var OptionValidatorNotEmpty = func(key string, val vars.Value) error {
	if val.Len() == 0 {
		return fmt.Errorf("%w: %s value can not be empty", ErrOption, key)
	}
	return nil
}

func getRuntimeConfig() []OptionArg {
	ver := version.Current()
	addr, _ := address.Current()

	opts := []OptionArg{
		// {
		// 	key:   "*",
		// 	value: "",
		// 	kind:  ReadOnlyOption | RuntimeOption,
		// 	validator: func(key string, val vars.Value) error {
		// 		if strings.HasPrefix(key, "app.") || strings.HasPrefix(key, "log.") || strings.HasPrefix(key, "happy.") || strings.HasPrefix(key, "fs.") {
		// 			return fmt.Errorf("%w: unknown application option %s", ErrOptionValidation, key)
		// 		}
		// 		return nil
		// 	},
		// },
		{
			key:       "app.devel",
			value:     version.IsDev(ver.String()),
			desc:      "Is application in development mode",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.version",
			value:     ver.String(),
			desc:      "Application version",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.fs.path.pwd",
			value:     "",
			desc:      "Current working directory",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.fs.path.home",
			value:     "",
			desc:      "Current user home directory",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.fs.path.tmp",
			value:     "",
			desc:      "Runtime tmp directory",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.fs.path.cache",
			value:     "",
			desc:      "Application cache directory",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.fs.path.config",
			value:     "",
			desc:      "Application configuration directory",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.main.exec.x",
			value:     "",
			desc:      "-x flag is set to print all commands as executed",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.profile.name",
			value:     "",
			desc:      "name of current settings profile",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.profile.file",
			value:     "",
			desc:      "file path of current settings profile file",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.firstuse",
			value:     false,
			desc:      "application first use",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.module",
			value:     addr.Module(),
			desc:      "application module",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "app.address",
			value:     "",
			desc:      "application address",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
	}
	return opts
}

func getDefaultCommandOpts() []OptionArg {
	opts := []OptionArg{
		{
			key:       "description",
			value:     "",
			desc:      "Long escription for command",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "usage",
			value:     "",
			desc:      "Usage description for command",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "category",
			value:     "",
			desc:      "Command category",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "init.allowed",
			value:     true,
			desc:      "Is command allowed to be used when application is used first time",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "addons.disabled",
			value:     false,
			desc:      "Skip registering addons for this command, Addons and their provided services will not be loaded when this command is used.",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "argn.min",
			value:     0,
			desc:      "Minimum argument count for command",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
		{
			key:       "argn.max",
			value:     0,
			desc:      "Maximum argument count for command",
			kind:      ConfigOption | ReadOnlyOption,
			validator: noopvalidator,
		},
	}
	return opts
}
