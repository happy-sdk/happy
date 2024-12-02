// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"golang.org/x/text/language"
)

var (
	ErrBlueprint = errors.New("settings blueprint")
)

type Blueprint struct {
	name string
	mode ExecutionMode
	pkg  string
	// module string
	// lang   language.Tag // default language
	specs      map[string]SettingSpec
	errs       []error
	groups     map[string]*Blueprint
	migrations map[string]string
}

type groupSettings struct{}

func (g groupSettings) Blueprint() (*Blueprint, error) {
	return New(g)
}

func (b *Blueprint) AddSpec(spec SettingSpec) error {
	if b.specs == nil {
		b.specs = make(map[string]SettingSpec)
	}
	if b.groups == nil {
		b.groups = make(map[string]*Blueprint)
	}
	if err := spec.Validate(); err != nil {
		return err
	}
	if strings.Contains(spec.Key, ".") {
		group, skey, _ := strings.Cut(spec.Key, ".")
		g, ok := b.groups[group]
		if !ok {
			if err := b.Extend(group, groupSettings{}); err != nil {
				return fmt.Errorf("%w: extending %s %s", ErrBlueprint, b.pkg, err.Error())
			}
			return b.AddSpec(spec)
		}
		spec.Key = skey
		return g.AddSpec(spec)
	}
	if spec.Settings != nil {
		if _, ok := b.groups[spec.Key]; ok {
			return fmt.Errorf("%w: group %s already exists", ErrBlueprint, spec.Key)
		}
		b.groups[spec.Key] = spec.Settings
	}
	b.specs[spec.Key] = spec
	return nil
}

func (b *Blueprint) Migrate(keyfrom, keyto string) error {
	if b.migrations == nil {
		b.migrations = make(map[string]string)
	}
	if to, ok := b.migrations[keyfrom]; ok {
		return fmt.Errorf("%w: adding migration from %s to %s. from %s to %s already exists", ErrBlueprint, keyfrom, keyto, keyfrom, to)
	}
	b.migrations[keyfrom] = keyto
	return nil
}

func (b *Blueprint) settingSpecFromField(field reflect.StructField, value reflect.Value) (SettingSpec, error) {
	spec := SettingSpec{}

	var persistent string
	spec.Key, persistent, _ = strings.Cut(field.Tag.Get("key"), ",")

	if persistent == "save" {
		spec.Persistent = true
	}
	if persistent == "config" {
		spec.UserDefined = true
	}

	// Use struct field name converted to dot.separated.format if 'key' tag is not present
	if spec.Key == "" {
		spec.Key = toUndersCoreSeparated(field.Name)
	}

	if fieldImplementsSettings(field) {
		spec.Mutability = SettingImmutable
		spec.IsSet = true
		spec.Kind = KindSettings

		// Handle nested settings
		var err error
		if value.Kind() == reflect.Ptr {
			// If the value is a nil pointer, initialize it
			if value.IsNil() {
				value.Set(reflect.New(value.Type().Elem()))
			}
			spec.Settings, err = New(value.Interface().(Settings))
		} else {
			spec.Settings, err = New(value.Addr().Interface().(Settings))
		}

		if err != nil {
			return spec, err
		}
	} else if fieldImplementsSetting(field) {
		spec.Required = field.Tag.Get("required") == "" || field.Tag.Get("required") == "true"

		mutation := field.Tag.Get("mutation")
		switch mutation {
		case "once":
			spec.Mutability = SettingOnce
		case "mutable":
			spec.Mutability = SettingMutable
		default:
			spec.Mutability = SettingImmutable
			spec.IsSet = true
		}

		kindGetterMethod := value.MethodByName("SettingKind")
		if kindGetterMethod.IsValid() {
			results := kindGetterMethod.Call(nil)
			if len(results) != 1 {
				return spec, fmt.Errorf("%w: %q field %q must implement either Setting or Settings interface", ErrBlueprint, b.pkg, spec.Key)
			}
			spec.Kind = results[0].Interface().(Kind)
		} else {
			spec.Kind = KindCustom
		}

		desc := field.Tag.Get("desc")
		if desc != "" {
			if spec.i18n == nil {
				spec.i18n = make(map[language.Tag]string)
			}
			spec.i18n[language.English] = desc
		}
		spec.Default = field.Tag.Get("default")
		if spec.Kind == KindBool && (spec.Default != "" && spec.Default != "false") {
			return spec, fmt.Errorf("%w: %q boolean field %q can have default value only false", ErrBlueprint, b.pkg, spec.Key)
		}

		if isFieldSet(value) {
			// if the field is set by the developer, then set it as the default value
			val := getStringValue(value)
			spec.Value = val
			spec.Default = val
		} else {
			spec.Value = spec.Default
		}

		// Check if value implements Marshaller and Unmarshaller
		if value.CanAddr() {
			ptr := value.Addr().Interface()
			if unmarshaller, ok := ptr.(Unmarshaller); ok {
				spec.Unmarchaler = unmarshaller
			}
			if marshaller, ok := ptr.(Marshaller); ok {
				spec.Marchaler = marshaller
			}
		}
		if spec.Unmarchaler == nil || spec.Marchaler == nil {
			return spec, fmt.Errorf("%w: %q field %q must implement SettingField interface (both UnmarshalSetting and MarshalSetting methods required)", ErrBlueprint, b.pkg, spec.Key)
		}
	} else {
		return spec, fmt.Errorf("%w: %q field %q must implement either Settings or SettingField interface", ErrBlueprint, b.pkg, spec.Key)
	}

	return spec, nil
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
	b.specs[key] = spec
}

func (b *Blueprint) Describe(key string, lang language.Tag, description string) {
	spec, ok := b.specs[key]
	if !ok {
		b.errs = append(b.errs, fmt.Errorf("%w: %s not found to add description", ErrBlueprint, key))
		return
	}

	if spec.i18n == nil {
		spec.i18n = make(map[language.Tag]string)
	}

	// check does language exist already
	if v, ok := spec.i18n[lang]; ok {
		b.errs = append(b.errs, fmt.Errorf("%w: %s already described in %s: %s", ErrBlueprint, key, lang, v))
		return
	}

	spec.i18n[lang] = description
	b.specs[key] = spec
}

func (b *Blueprint) GetSpec(key string) (SettingSpec, error) {
	var (
		spec SettingSpec
	)
	if strings.Contains(key, ".") {
		group, skey, _ := strings.Cut(key, ".")
		if g, ok := b.groups[group]; ok {
			return g.GetSpec(skey)
		}
		return spec, fmt.Errorf("no settings group %s found in %s (%s) key: %s", group, b.name, b.pkg, key)
	}

	if g, ok := b.groups[key]; ok {
		return g.GetSpec(key)
	}

	var ok bool
	spec, ok = b.specs[key]
	if !ok {
		return spec, fmt.Errorf("no settings group or key found %s in %s (%s)", key, b.name, b.pkg)
	}

	return spec, nil
}

func (b *Blueprint) SetDefault(key string, value string) error {
	if strings.Contains(key, ".") {
		group, skey, _ := strings.Cut(key, ".")
		g, ok := b.groups[group]
		if !ok {
			return fmt.Errorf("%w: SetDefault group %s not found", ErrBlueprint, group)
		}
		return g.SetDefault(skey, value)
	}
	spec, ok := b.specs[key]
	if !ok {
		return fmt.Errorf("%w: SetDefault key %s not found", ErrBlueprint, key)
	}
	spec.Default = value
	b.specs[key] = spec

	return nil
}

func (b *Blueprint) Extend(group string, ext Settings) (err error) {
	if ext == nil {
		return fmt.Errorf("%w: extending %s with nil", ErrBlueprint, group)
	}

	var exptbp *Blueprint
	var berr error

	// Attempt to call Blueprint with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Panic occurred, treat it as Blueprint being not safely callable
				berr = fmt.Errorf("failed to call Blueprint safely: %v", r)
			}
		}()

		exptbp, err = ext.Blueprint()
	}()

	// Check if there was an error or panic during Blueprint call
	if err != nil {
		return fmt.Errorf("%w: extending %s with error: %s", ErrBlueprint, b.pkg, err)
	}

	// Check if there was an error or panic during Blueprint call
	if berr != nil {
		exptbp, err = New(ext)
		if err != nil {
			return fmt.Errorf("%w: extending %s with error: %s", ErrBlueprint, b.pkg, err)
		}
	}
	if exptbp == nil {
		// Handle the case where Blueprint does not return an expected value
		return fmt.Errorf("%w: Blueprint returned a nil value for group %s", ErrBlueprint, group)
	}

	exptbp.name = group
	if b.groups == nil {
		b.groups = make(map[string]*Blueprint)
	}
	if _, ok := b.groups[group]; ok {
		return fmt.Errorf("%w: group %s already exists, cannot extend with %s", ErrBlueprint, group, exptbp.pkg)
	}
	b.groups[group] = exptbp
	return nil
}

func (b *Blueprint) Schema(module, version string) (Schema, error) {
	s := Schema{
		version:    version,
		mode:       b.mode,
		pkg:        b.pkg,
		module:     module,
		settings:   make(map[string]SettingSpec),
		migrations: b.migrations,
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
