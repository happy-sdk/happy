// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import "fmt"

type Error struct {
	code int
	key  string
}

func NewError(key string) *Error {
	return &Error{
		key: key,
	}
}

func (e *Error) WithCode(code int) *Error {
	e.code = code
	return e
}

func (e *Error) Error() string {
	if e.code == 0 {
		return T(e.key)
	}
	return fmt.Sprintf("%d: %s", e.code, T(e.key))
}
