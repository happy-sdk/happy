// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

import (
	"fmt"
	"strconv"
	"time"

	"github.com/happy-sdk/happy-go/vars"
)

type Kind uint8

const (
	KindSettings = Kind(vars.KindInterface)
	KindCustom   = Kind(vars.KindByteSlice)

	KindInvalid  = Kind(vars.KindInvalid)
	KindBool     = Kind(vars.KindBool)
	KindInt      = Kind(vars.KindInt)
	KindUint     = Kind(vars.KindUint)
	KindString   = Kind(vars.KindString)
	KindDuration = Kind(vars.KindDuration)
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

// String represents a setting type based on a string.
type String string

// MarshalSetting converts the String setting to a byte slice for storage or transmission.
func (s String) MarshalSetting() ([]byte, error) {
	// Simply cast the String to a byte slice.
	return []byte(s), nil
}

// UnmarshalSetting updates the String setting from a byte slice, typically read from storage or received in a message.
func (s *String) UnmarshalSetting(data []byte) error {
	// Directly convert the byte slice to String.
	*s = String(data)
	return nil
}

func (s String) String() string {
	return string(s)
}

func (s String) SettingKind() Kind {
	return KindString
}

// Bool represents a setting type based on a boolean.
type Bool bool

// MarshalSetting converts the Bool setting to a byte slice, representing "true" or "false".
func (b Bool) MarshalSetting() ([]byte, error) {
	return []byte(b.String()), nil
}

// UnmarshalSetting updates the Bool setting from a byte slice, interpreting it as "true" or "false".
func (b *Bool) UnmarshalSetting(data []byte) error {
	// Parse the byte slice as a boolean and store it in Bool.
	val, err := strconv.ParseBool(string(data))
	if err != nil {
		return err
	}
	*b = Bool(val)
	return nil
}

func (b Bool) String() string {
	return strconv.FormatBool(bool(b))
}

func (b Bool) SettingKind() Kind {
	return KindBool
}

// Int represents a setting type based on an integer.
type Int int

// MarshalSetting converts the Int setting to a byte slice for storage or transmission.
func (i Int) MarshalSetting() ([]byte, error) {
	// Convert Int to a string and then to a byte slice.
	return []byte(i.String()), nil
}

func (i Int) String() string {
	return strconv.Itoa(int(i))
}

func (i Int) SettingKind() Kind {
	return KindInt
}

// UnmarshalSetting updates the Int setting from a byte slice, interpreting it as an integer.
func (i *Int) UnmarshalSetting(data []byte) error {
	// Convert the byte slice to a string, then parse it as an integer.
	val, err := strconv.Atoi(string(data))
	if err != nil {
		return err
	}
	*i = Int(val)
	return nil
}

// Uint represents a setting type based on an unsigned integer.
type Uint uint

// MarshalSetting converts the Uint setting to a byte slice for storage or transmission.
func (u Uint) MarshalSetting() ([]byte, error) {
	// Convert Uint to a string and then to a byte slice.
	return []byte(u.String()), nil
}

func (u Uint) SettingKind() Kind {
	return KindUint
}

// UnmarshalSetting updates the Uint setting from a byte slice, interpreting it as an unsigned integer.
func (u *Uint) UnmarshalSetting(data []byte) error {
	str := string(data)
	if str == "" {
		return nil
	}
	// Convert the byte slice to a string, then parse it as an unsigned integer.
	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	*u = Uint(val)
	return nil
}

func (u Uint) String() string {
	return fmt.Sprint(uint(u))
}

type Duration time.Duration

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d Duration) MarshalSetting() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Duration) UnmarshalSetting(data []byte) error {
	val, err := time.ParseDuration(string(data))
	if err != nil {
		return err
	}
	*d = Duration(val)
	return nil
}

func (d Duration) SettingKind() Kind {
	return KindDuration
}
