// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

import (
	"fmt"

	"github.com/happy-sdk/happy/pkg/vars"
	"golang.org/x/text/language"
)

type Mutability uint8

const (
	// SettingImmutable can not be changed on runtime.
	SettingImmutable Mutability = 254
	// SettingOnce can be set only once on runtime.
	// When changed typically requires a reload of application.
	SettingOnce Mutability = 253
	// SettingMutable can be changed on runtime.
	SettingMutable Mutability = 252
)

func (m Mutability) String() string {
	switch m {
	case SettingImmutable:
		return "immutable"
	case SettingOnce:
		return "once"
	case SettingMutable:
		return "mutable"
	}
	return "unknown"
}

type SettingSpec struct {
	IsSet       bool
	Kind        Kind
	Key         string
	Default     string
	Mutability  Mutability
	Value       string
	Required    bool
	Persistent  bool
	UserDefined bool
	Unmarchaler Unmarshaller
	Marchaler   Marshaller
	Settings    *Blueprint
	validators  []validator
	i18n        map[language.Tag]string
}

func (s SettingSpec) Validate() error {
	if s.Mutability > SettingImmutable || s.Mutability < SettingMutable {
		return fmt.Errorf("%w: invalid mutability setting %d for %s", ErrSpec, s.Mutability, s.Key)
	}
	return nil
}

func (s SettingSpec) ValidateValue(value string) error {
	s.Value = value
	setting, err := s.setting()
	if err != nil {
		return err
	}
	for _, v := range s.validators {
		if err := v.fn(setting); err != nil {
			return fmt.Errorf("%s: %s", v.desc, err.Error())
		}
	}
	return nil
}

func (s SettingSpec) Setting(lang language.Tag) (Setting, error) {
	setting, err := s.setting()
	if err != nil {
		return setting, err
	}
	if s.i18n != nil {

		if desc, ok := s.i18n[lang]; ok {
			setting.desc = desc
		}
	}
	return setting, nil
}

func (s SettingSpec) setting() (Setting, error) {
	setting := Setting{
		key:         s.Key,
		kind:        s.Kind,
		isSet:       s.IsSet,
		mutability:  s.Mutability,
		persistent:  s.Persistent,
		userDefined: s.UserDefined,
	}

	var err error
	setting.vv, err = vars.NewAs(s.Key, s.Value, true, vars.Kind(s.Kind))
	if err != nil {
		return Setting{}, fmt.Errorf("%w: key(%s)  %s", ErrProfile, s.Key, err.Error())
	}
	setting.dvv, err = vars.NewAs(s.Key, s.Default, true, vars.Kind(s.Kind))
	if err != nil {
		return Setting{}, fmt.Errorf("%w: key(%s)  %s", ErrProfile, s.Key, err.Error())
	}
	return setting, nil
}

type Setting struct {
	key         string
	vv          vars.Variable
	dvv         vars.Variable
	kind        Kind
	isSet       bool
	mutability  Mutability
	persistent  bool
	userDefined bool
	desc        string
}

func (s Setting) String() string {
	return s.vv.String()
}

func (s Setting) Key() string {
	return s.key
}

func (s Setting) IsSet() bool {
	return s.isSet
}

func (s Setting) Value() vars.Variable {
	return s.vv
}

func (s Setting) Default() vars.Variable {
	return s.dvv
}

func (s Setting) Kind() Kind {
	return s.kind
}

func (s Setting) Persistent() bool {
	return s.persistent
}

func (s Setting) UserDefined() bool {
	return s.userDefined
}

func (s Setting) Mutability() Mutability {
	return s.mutability
}

func (s Setting) Description() string {
	return s.desc
}
