// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package happyx

import (
	"errors"
	"fmt"
	"github.com/mkungla/happy"
)

var (
	ErrNotImplemented = NewError("missing implementation")
	ErrConfiguration  = NewError("configuration error")
	ErrOption         = NewError("option error")
)

type Error struct {
	err error
	// ts   int64
	code int
}

func Errorf(format string, args ...any) happy.Error {
	return &Error{
		err: fmt.Errorf(format, args...),
	}
}

func NotImplementedError(text string) happy.Error {
	return &Error{
		err: fmt.Errorf("%w: %s", ErrNotImplemented, text),
	}
}

func InvalidOptionError(scope, key string) happy.Error {
	if len(scope) > 0 {
		scope = " " + scope
	}

	return &Error{
		err: fmt.Errorf("%w: invalid option %q for%s", ErrOption, key, scope),
	}
}

func NewError(text string) happy.Error {
	return &Error{
		err: errors.New(text),
	}
}

func (e *Error) Is(target error) bool {
	return errors.Is(e.err, target)
}

func (e *Error) Code() int {
	return e.code
}

func (e *Error) Error() string {
	return e.err.Error()
}

func (e *Error) Wrap(err error) happy.Error {
	c := *e
	msg := e.err.Error()
	c.err = fmt.Errorf("%s: %w", msg, err)
	return &c
}
func (e *Error) WithText(text string) happy.Error {
	c := *e
	c.err = fmt.Errorf("%w: %s", e.err, text)
	return &c
}
