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

package flags

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"github.com/mkungla/happy/x/pkg/varflag"
)

func VersionFlag() happy.Flag {
	f, _ := varflag.Bool(
		"version",
		false,
		"display current application version",
	)
	flag := varflag.AsFlag[happy.Flag, happy.Variable, happy.Value](f)
	return flag
}

func XFlag() happy.Flag {
	f, _ := varflag.Bool(
		"x",
		false,
		"The -x flag prints all the external commands as they are executed.",
	)
	flag := varflag.AsFlag[happy.Flag, happy.Variable, happy.Value](f)
	return flag
}

func HelpFlag() happy.Flag {
	f, _ := varflag.Bool(
		"help",
		false,
		"display help or help for the command. [...command --help]",
		"h",
	)
	return varflag.AsFlag[happy.Flag, happy.Variable, happy.Value](f)
}

func LoggerFlags() (flags []happy.FlagCreateFunc) {
	varflags := []varflag.FlagCreateFunc{
		varflag.BoolFunc(
			"system-debug",
			false,
			"enable system debug log level (very verbose)",
		),
		varflag.BoolFunc(
			"debug",
			false,
			"enable debug log level. when debug flag is after the command then debugging will be enabled only for that command",
		),
		varflag.BoolFunc(
			"verbose",
			false,
			"enable verbose log level",
			"v",
		),
	}

	for _, vfunc := range varflags {
		flags = append(flags, flagCreateFunc(vfunc))
	}
	return flags
}

func flagCreateFunc(vfunc varflag.FlagCreateFunc) happy.FlagCreateFunc {
	return func() (happy.Flag, happy.Error) {
		f, err := vfunc()
		if err != nil {
			return nil, happyx.NewError("flags error").Wrap(err)
		}
		return varflag.AsFlag[happy.Flag, happy.Variable, happy.Value](f), nil
	}
}
