// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package settings

import (
	"fmt"

	"github.com/happy-sdk/vars"
)

type Mutability uint8

const (
	// Mutability
	SettingImmutable Mutability = 254
	SettingOnce      Mutability = 253
	SettingMutable   Mutability = 252
)

type Kind uint8

const (
	KindSettings = Kind(vars.KindInterface)
	KindCustom   = Kind(vars.KindByteSlice)

	KindInvalid = Kind(vars.KindInvalid)
	KindBool    = Kind(vars.KindBool)
	KindInt     = Kind(vars.KindInt)
	KindUint    = Kind(vars.KindUint)
	KindString  = Kind(vars.KindString)
)

func (k Kind) String() string {
	switch k {
	case KindCustom:
		return "custom"
	case KindSettings:
		return "settings"
	}
	return vars.Kind(k).String()
}

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
