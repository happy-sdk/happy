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

package sdk

import "github.com/mkungla/happy"

func OptionFunc(vfunc happy.VariableParseFunc) happy.OptionWriteFunc {
	return func(opts happy.OptionWriter) happy.Error {
		v, err := vfunc()
		if err != nil {
			return err
		}
		return opts.SetOption(v)
	}
}

func Option(k string, v any) happy.OptionWriteFunc {
	return OptionFunc(nil)
}

func ReadOnlyOption(k string, v any) happy.OptionWriteFunc {
	return OptionFunc(nil)
}
