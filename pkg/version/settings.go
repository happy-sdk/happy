// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package version

type Setting Version

// MarshalSetting converts the String setting to a byte slice for storage or transmission.
func (s *Setting) MarshalSetting() ([]byte, error) {
	// Simply cast the String to a byte slice.
	return []byte(*s), nil
}

// UnmarshalSetting updates the String setting from a byte slice, typically read from storage or received in a message.
func (s *Setting) UnmarshalSetting(data []byte) error {
	// Directly convert the byte slice to String.
	*s = Setting(data)
	return nil
}

// Stringer implementation for String type
func (s Setting) String() string {
	return string(s)
}

// // SettingKind returns the kind of setting
// func (s Setting) SettingKind() Kind {
// 	return KindString
// }
