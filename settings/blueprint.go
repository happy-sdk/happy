// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/text/language"
)

var (
	ErrBlueprint = errors.New("settings blueprint")
)

type Blueprint struct {
	mode ExecutionMode
	pkg  string
	// module string
	// lang   language.Tag // default language
	specs  map[string]SettingSpec
	i18n   map[language.Tag]map[string]string
	errs   []error
	groups map[string]*Blueprint
}

func (b *Blueprint) AddSpec(spec SettingSpec) error {
	if b.specs == nil {
		b.specs = make(map[string]SettingSpec)
	}
	if err := spec.Validate(); err != nil {
		return err
	}
	b.specs[spec.Key] = spec
	return nil
}

type validator struct {
	desc string
	fn   func(s Setting) error
}

func (b *Blueprint) AddValidator(key, desc string, fn func(s Setting) error) {
	spec, ok := b.specs[key]
	if !ok {
		b.errs = append(b.errs, fmt.Errorf("%w: %s not found to add validator", ErrBlueprint, key))
		return
	}
	if len(desc) == 0 {
		desc = fmt.Sprintf("%s validator", key)
	}
	spec.validators = append(b.specs[key].validators, validator{
		desc: desc,
		fn:   fn,
	})
}

func (b *Blueprint) Describe(key string, lang language.Tag, description string) {
	// check are languages initialized
	if b.i18n == nil {
		b.i18n = make(map[language.Tag]map[string]string)
	}
	// check does language exist already
	i18n, ok := b.i18n[lang]
	if !ok {
		i18n = make(map[string]string)
	}
	// check is setting already described in language
	if v, described := i18n[key]; described {
		b.errs = append(b.errs, fmt.Errorf("%w: %s already described in %s: %s", ErrBlueprint, key, lang, v))
		return
	}

	i18n[key] = description
}

func (b *Blueprint) GetSpec(key string) (SettingSpec, error) {
	spec, ok := b.specs[key]
	if !ok {
		return spec, fmt.Errorf("%s: not found", key)
	}
	return spec, nil
}

func (b *Blueprint) Extend(group string, ext Settings) {
	exptbp, err := ext.Blueprint()
	if err != nil {
		b.errs = append(b.errs, fmt.Errorf("%w: extending %s %s", ErrBlueprint, b.pkg, err.Error()))
		return
	}
	if b.groups == nil {
		b.groups = make(map[string]*Blueprint)
	}
	if _, ok := b.groups[group]; ok {
		b.errs = append(b.errs, fmt.Errorf("%w: group %s already exists, can not extend with %s", ErrBlueprint, group, exptbp.pkg))
		return
	}
	b.groups[group] = exptbp
}

func (b *Blueprint) Schema(module, version string) (Schema, error) {
	s := Schema{
		version:  version,
		mode:     b.mode,
		pkg:      b.pkg,
		module:   module,
		settings: make(map[string]SettingSpec),
	}
	s.setID()

	for k, v := range b.specs {
		if v.Settings != nil {
			sschema, err := v.Settings.Schema(module, version)
			if err != nil {
				return s, err
			}
			for sk, sv := range sschema.settings {
				key := fmt.Sprintf("%s.%s", k, sk)
				sv.Key = key
				if err := s.set(key, sv); err != nil {
					return s, err
				}
			}

		} else {
			if err := s.set(k, v); err != nil {
				return s, err
			}
		}
	}

	if b.groups != nil {
		for gname, group := range b.groups {
			gshema, err := group.Schema(module, version)
			if err != nil {
				return s, err
			}
			for k, v := range gshema.settings {
				key := fmt.Sprintf("%s.%s", gname, k)
				v.Key = key
				if err := s.set(key, v); err != nil {
					return s, err
				}
			}
		}
	}
	return s, nil
}

func (b *Blueprint) setPKG() string {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	fnName := fn.Name()
	var pkgPath string

	lastDotIndex := strings.LastIndex(fnName, ".")
	if lastDotIndex == -1 {
		pkgPath = fnName
	} else {
		pkgPath = fnName[:lastDotIndex]
	}

	// In test mode, the path may include "_test" suffix, which we should strip.
	if b.mode == ModeTesting {
		pkgPath, _, _ = strings.Cut(pkgPath, "_test")
	}

	return pkgPath
}
