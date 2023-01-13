// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mkungla/happy/pkg/address"
	"github.com/mkungla/happy/pkg/hlog"
	"github.com/mkungla/happy/pkg/vars"
	"github.com/mkungla/happy/pkg/version"
	"golang.org/x/mod/semver"
)

type (

	// Options is general collection of settings
	// attached to specific application component.
	Options struct {
		name   string
		db     vars.Map
		config map[string]OptionAttr
	}

	// Option is used to define option and
	// apply given key value to options.
	OptionAttr struct {
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
	optionKindOption OptionKind = iota << 1
	optionKindReadOnly
	// groups
	optionKindSetting
	optionKindConfig
	optionKindAddon
)

var (
	ErrOption           = errors.New("option error")
	ErrOptionReadOnly   = fmt.Errorf("%w: readonly option", ErrOption)
	ErrOptionValidation = fmt.Errorf("%w: option validation error", ErrOption)
	ErrOptions          = errors.New("options error")
)

// Opt creates option for given key value pair
// which can be applied to any Options set.
func Option(key string, value any) OptionAttr {
	return OptionAttr{
		key:   key,
		value: value,
		kind:  optionKindOption,
	}
}

func (o OptionAttr) apply(opts *Options) error {
	return opts.Set(o.key, o.value)
}

// NewOptions returns new named options with optiona validator when provided.
func NewOptions(name string, defaults []OptionAttr) (*Options, error) {
	opts := &Options{
		name: name,
	}
	if defaults != nil && len(defaults) > 0 {
		opts.config = make(map[string]OptionAttr)
		for _, cnf := range defaults {
			key, err := vars.ParseKey(cnf.key)
			if err != nil {
				return nil, errors.Join(fmt.Errorf("%w: %s invalid option key", ErrOptions, name), err)
			}
			if _, ok := opts.config[key]; ok {
				return nil, fmt.Errorf("%w: %s duplicated option key %s", ErrOptions, name, key)
			}
			opts.config[key] = cnf
		}
	}
	return opts, nil
}

// Accepts reports whether given option key is accepted by Options.
func (opts *Options) Accepts(key string) bool {
	if opts.config == nil {
		return true
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

func (opts *Options) Get(key string) vars.Variable {
	return opts.db.Get(key)
}

func (opts *Options) Load(key string) (vars.Variable, bool) {
	return opts.db.Load(key)
}

func (opts *Options) Set(key string, value any) error {
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
		return fmt.Errorf(
			"%w: can not set %s for %s",
			ErrOptionReadOnly,
			key,
			opts.name,
		)
	}
	val, err := vars.NewValue(value)
	if err != nil {
		return err
	}

	// there is no validation required
	if opts.config == nil {
		return opts.db.Store(key, val)
	}

	var cnf *OptionAttr
	if c, ok := opts.config[key]; ok && c.validator != nil {
		cnf = &c
	} else if c, ok := opts.config["*"]; ok && c.validator != nil {
		cnf = &c
	}
	if cnf == nil {
		return fmt.Errorf("%w: no validator for %s", ErrOption, key)
	}
	// fallback validator
	if err := cnf.validator(key, val); err != nil {
		return err
	}
	readonly := cnf.kind&optionKindReadOnly == optionKindReadOnly
	return opts.db.StoreReadOnly(key, val, readonly)
}

// Has reports whether options has given key
func (opts *Options) Has(key string) bool {
	return opts.db.Has(key)
}

var noopOptValidator = func(key string, val vars.Value) error {
	return nil
}

func getDefaultApplicationConfig() []OptionAttr {
	configOpts := []OptionAttr{
		{
			key:   "*",
			value: "",
			kind:  optionKindReadOnly,
			validator: func(key string, val vars.Value) error {
				if strings.HasPrefix(key, "app.") {
					return fmt.Errorf("%w: unknown application option %s", ErrOptionValidation, key)
				}
				if strings.HasPrefix(key, "log.") {
					return fmt.Errorf("%w: unknown application option %s", ErrOptionValidation, key)
				}
				return nil
			},
		},
		{
			key:       "app.name",
			value:     "Happy Application",
			desc:      "Name of the application",
			kind:      optionKindReadOnly | optionKindConfig,
			validator: noopOptValidator,
		},
		{
			key:       "app.description",
			value:     "",
			desc:      "Short description for application",
			kind:      optionKindReadOnly | optionKindConfig,
			validator: noopOptValidator,
		},
		{
			key:       "app.copyright.by",
			value:     "",
			desc:      "Copyright owner",
			kind:      optionKindReadOnly | optionKindConfig,
			validator: noopOptValidator,
		},
		{
			key:       "app.copyright.since",
			value:     0,
			desc:      "Copyright since",
			kind:      optionKindReadOnly | optionKindConfig,
			validator: noopOptValidator,
		},
		{
			key:       "app.license",
			value:     "0",
			desc:      "License",
			kind:      optionKindReadOnly | optionKindConfig,
			validator: noopOptValidator,
		},
		{
			key: "app.host.addr",
			value: func() string {
				addr, err := address.Current()
				if err != nil {
					panic(err)
				}
				return addr.String()
			}(),
			desc: "Application happy host address",
			kind: optionKindReadOnly | optionKindConfig,
			validator: func(key string, val vars.Value) error {
				_, err := address.Parse(val.String())
				if err != nil {
					return fmt.Errorf(
						"%w: invalid host address (%q)",
						ErrOptionValidation, val.String())
				}
				return nil
			},
		},
		{
			key:   "app.version",
			value: version.Current(),
			desc:  "Application version",
			kind:  optionKindReadOnly | optionKindConfig,
			validator: func(key string, val vars.Value) error {
				if !semver.IsValid(val.String()) {
					return fmt.Errorf("%w %q, version must be valid semantic version", ErrInvalidVersion, val)
				}
				return nil
			},
		},
		{
			key:   "app.settings.persistent",
			value: false,
			desc:  "persist settings across restarts",
			kind:  optionKindReadOnly | optionKindConfig,
			validator: func(key string, val vars.Value) error {
				if val.Kind() != vars.KindBool {
					return fmt.Errorf("%w: %s must be boolean got %s(%s)", ErrOptionValidation, key, val.Kind(), val.String())
				}
				return nil
			},
		},
		{
			key:       "log.level",
			value:     hlog.LevelInfo,
			desc:      "Log level for applicaton",
			kind:      optionKindReadOnly | optionKindSetting,
			validator: noopOptValidator,
		},
		{
			key:       "log.source",
			value:     false,
			desc:      "adds source = file:line attribute to the output indicating the source code position of the log statement.",
			kind:      optionKindReadOnly | optionKindSetting,
			validator: noopOptValidator,
		},
		{
			key:       "log.colors",
			value:     true,
			desc:      "enable colored log output",
			kind:      optionKindReadOnly | optionKindSetting,
			validator: noopOptValidator,
		},
		{
			key:       "log.stdlog",
			value:     false,
			desc:      "set configured logger as slog.Default",
			kind:      optionKindReadOnly | optionKindConfig,
			validator: noopOptValidator,
		},
		{
			key:       "log.secrets",
			value:     "",
			desc:      "comma separated list of attr key to mask value with ****",
			kind:      optionKindReadOnly | optionKindConfig,
			validator: noopOptValidator,
		},
	}
	return configOpts
}

func getDefaultCommandOpts() []OptionAttr {
	opts := []OptionAttr{
		{
			key:       "description",
			value:     "",
			desc:      "Long escription for command",
			kind:      optionKindReadOnly,
			validator: noopOptValidator,
		},
		{
			key:       "usage",
			value:     "",
			desc:      "Usage description for command",
			kind:      optionKindReadOnly,
			validator: noopOptValidator,
		},
		{
			key:       "category",
			value:     "",
			desc:      "Command category",
			kind:      optionKindReadOnly,
			validator: noopOptValidator,
		},
	}
	return opts
}
