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

type Setting struct {
	key        string
	vv         vars.Variable
	kind       Kind
	isSet      bool
	mutability Mutability
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
