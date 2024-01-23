// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

import (
	"fmt"

	"github.com/happy-sdk/happy/pkg/vars"
)

type Mutability uint8

const (
	// Mutability
	SettingImmutable Mutability = 254
	SettingOnce      Mutability = 253
	SettingMutable   Mutability = 252
)

type SettingSpec struct {
	IsSet       bool
	Kind        Kind
	Key         string
	Default     string
	Mutability  Mutability
	Value       string
	Required    bool
	Persistent  bool
	Unmarchaler Unmarshaller
	Marchaler   Marshaller
	Settings    *Blueprint
	validators  []validator
}

func (s SettingSpec) Validate() error {
	if s.Mutability > SettingImmutable || s.Mutability < SettingMutable {
		return fmt.Errorf("%w: invalid mutability setting %d for %s", ErrSpec, s.Mutability, s.Key)
	}
	return nil
}

func (s SettingSpec) ValidateValue(value string) error {
	s.Value = value
	setting, err := s.Setting()
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

func (s SettingSpec) Setting() (Setting, error) {
	setting := Setting{
		key:        s.Key,
		kind:       s.Kind,
		isSet:      s.IsSet,
		mutability: s.Mutability,
		persistent: s.Persistent,
	}
	var err error
	setting.vv, err = vars.NewAs(s.Key, s.Value, true, vars.Kind(s.Kind))
	if err != nil {
		return Setting{}, fmt.Errorf("%w: key(%s)  %s", ErrProfile, s.Key, err.Error())
	}
	return setting, nil
}

type Setting struct {
	key        string
	vv         vars.Variable
	kind       Kind
	isSet      bool
	mutability Mutability
	persistent bool
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

func (s Setting) Kind() Kind {
	return s.kind
}

func (s Setting) Persistent() bool {
	return s.persistent
}
