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

package commands

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/cli"
	"github.com/mkungla/happy/x/happyx"
)

func BashCompletion() happy.Command {
	cmd, err := cli.NewCommand(
		"bash-completion",
		happyx.ReadOnlyOption("usage.decription", "bash-completion helper command."),
		happyx.ReadOnlyOption("category", "internal"),
		happyx.ReadOnlyOption(
			"decription",
			"This command can be called by bash completion mechanism to provide shell completions",
		),
	)
	if err != nil {
		return nil
	}

	cmd.Do(func(session happy.Session, flags happy.Flags, assets happy.FS, status happy.ApplicationStatus) error {
		return happyx.NotImplementedError("bash-completion command do action")
	})

	return cmd
}
