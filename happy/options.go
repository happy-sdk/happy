// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mkungla/happy/pkg/address"
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
	defaultOption OptionKind = 1 << iota
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
		kind:  defaultOption,
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
	if defaults != nil && len(defaults) > 0 {
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
			if err := opts.Set(key, cnf.value); err != nil {
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

func getDefaultApplicationConfig() ([]OptionArg, error) {
	addr, err := address.Current()
	if err != nil {
		return nil, err
	}

	configOpts := []OptionArg{
		{
			key:   "*",
			value: "",
			kind:  ReadOnlyOption,
			validator: func(key string, val vars.Value) error {
				if strings.HasPrefix(key, "app.") || strings.HasPrefix(key, "log.") || strings.HasPrefix(key, "happy.") {
					return fmt.Errorf("%w: unknown application option %s", ErrOptionValidation, key)
				}
				return nil
			},
		},
		{
			key:       "app.name",
			value:     "Happy Application",
			desc:      "Name of the application",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "app.slug",
			value:     addr.Instance,
			desc:      "Slug of the application",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "app.description",
			value:     "",
			desc:      "Short description for application",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "app.copyright.by",
			value:     "",
			desc:      "Copyright owner",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "app.copyright.since",
			value:     0,
			desc:      "Copyright since",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "app.cron.on.service.start",
			value:     false,
			desc:      "Execute Cronjobs first time when service starts",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "app.fs.enabled",
			value:     false,
			desc:      "enable and load filesystem paths for application",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "app.license",
			value:     "",
			desc:      "License",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:   "app.throttle.ticks",
			value: time.Duration(time.Millisecond * 100),
			desc:  "Interfal target for system and service ticks",
			kind:  ReadOnlyOption | SettingsOption,
			validator: func(key string, val vars.Value) error {
				v, err := val.Int64()
				if err != nil {
					return err
				}
				if v < 1 {
					return fmt.Errorf(
						"%w: invalid throttle value %s(%d - %v), must be greater that 1",
						ErrOptionValidation, val.Kind(), v, val.Any())
				}
				return nil
			},
		},
		{
			key:   "app.version",
			value: version.Current(),
			desc:  "Application version",
			kind:  ReadOnlyOption | ConfigOption,
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
			kind:  ReadOnlyOption | ConfigOption,
			validator: func(key string, val vars.Value) error {
				if val.Kind() != vars.KindBool {
					return fmt.Errorf("%w: %s must be boolean got %s(%s)", ErrOptionValidation, key, val.Kind(), val.String())
				}
				return nil
			},
		},
		{
			key:       "log.level",
			value:     LogLevelTask,
			desc:      "Log level for applicaton",
			kind:      ReadOnlyOption | SettingsOption,
			validator: noopvalidator,
		},
		{
			key:       "log.source",
			value:     false,
			desc:      "adds source = file:line attribute to the output indicating the source code position of the log statement.",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "log.colors",
			value:     true,
			desc:      "enable colored log output",
			kind:      ReadOnlyOption | SettingsOption,
			validator: noopvalidator,
		},
		{
			key:       "log.stdlog",
			value:     false,
			desc:      "set configured logger as slog.Default",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "log.secrets",
			value:     "",
			desc:      "comma separated list of attr key to mask value with ****",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:   "app.host.addr",
			value: addr.String(),
			desc:  "Application happy host address",
			kind:  ReadOnlyOption | ConfigOption,
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
	}
	return configOpts, nil
}

func getDefaultCommandOpts() []OptionArg {
	opts := []OptionArg{
		{
			key:       "description",
			value:     "",
			desc:      "Long escription for command",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "usage",
			value:     "",
			desc:      "Usage description for command",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "category",
			value:     "",
			desc:      "Command category",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "allow.on.fresh.install",
			value:     true,
			desc:      "Is command allowed to be used when application is used first time",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
		{
			key:       "skip.addons",
			value:     false,
			desc:      "Skip registering addons for this command, Addons and their provided services will not be loaded when this command is used.",
			kind:      ReadOnlyOption | ConfigOption,
			validator: noopvalidator,
		},
	}
	return opts
}
