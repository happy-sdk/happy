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

package cli

import (
	"strings"

	"github.com/mkungla/happy"
	"github.com/mkungla/varflag/v5"
)

type Command struct {
	name      string
	category  string
	shortDesc string
	longDesc  string
	usage     string
	subCmd    *Command // set if subcommand was called

	beforeFn       happy.Action
	doFn           happy.Action
	afterFailureFn happy.Action
	afterSuccessFn happy.Action
	afterAlwaysFn  happy.Action

	subCommands map[string]*Command // subcommands
	parent      *Command
	parents     []string

	flags varflag.Flags

	isWrapperCommand bool
	errs             error
}

// NewCommand returns new command constructor.
func NewCommand(name string, argsn uint) (happy.Command, error) {
	fset, err := varflag.NewFlagSet(name, argsn)
	if err != nil {
		return nil, err
	}
	return &Command{
		name:  name,
		flags: fset,
	}, nil
}

// String returns name of the command.
// Or name of the active sub command in command chain.
func (c *Command) String() string {
	if c.subCmd != nil {
		return c.subCmd.String()
	}
	return c.name
}

// SetCategory sets help category to categorize
// commands in help output.
func (c *Command) SetCategory(category string) {
	c.category = strings.TrimSpace(category)
}

// SetShortDesc sets commands short description
// used when describing command within list.
func (c *Command) SetShortDesc(description string) {
	c.shortDesc = description
}

// Do should contain main of this command
// This function is called when:
//   - BeforeFunc is not set
//   - BeforeFunc succeeds

func (c *Command) Before(action happy.Action)       {}
func (c *Command) Do(action happy.Action)           {}
func (c *Command) AfterSuccess(action happy.Action) {}
func (c *Command) AfterFailure(action happy.Action) {}
func (c *Command) AfterAlways(action happy.Action)  {}
