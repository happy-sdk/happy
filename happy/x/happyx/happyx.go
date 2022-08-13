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
)

func OptionFunc(vfunc happy.VariableParseFunc) happy.OptionWriteFunc {
	return func(opts happy.OptionWriter) happy.Error {
		return ErrNotImplemented
		// v, err := vfunc()
		// if err != nil {
		// 	return err
		// }
		// return opts.SetOption(v)
	}
}

func Option(k string, v any) happy.OptionWriteFunc {
	return OptionFunc(nil)
}

func ReadOnlyOption(k string, v any) happy.OptionWriteFunc {
	return OptionFunc(nil)
}

// This could live in Session, but
// interface method must have no type parameters
func GetServiceAPI[SVC happy.Service](sess happy.Session, url string) (svc SVC, err Error) {
	return
}

// AsService(svc string, svc SVC) Error
// Session[SVC Service] interface {

func Errorf(format string, args ...any) happy.Error {
	return Error{
		err: fmt.Errorf(format, args...),
	}
}

type Error struct {
	err error
	// ts   int64
	code int
}

func NewError(text string) happy.Error {
	return Error{
		err: errors.New(text),
	}
}

func (e Error) Is(target error) bool {
	return errors.Is(e.err, target)
}

func (e Error) Code() int {
	return e.code
}

func (e Error) Error() string {
	return e.err.Error()
}
