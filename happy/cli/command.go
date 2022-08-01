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
	"fmt"
	"strings"
	"sync"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/config"
	"github.com/mkungla/varflag/v5"
	"github.com/mkungla/vars/v5"
)

type Command struct {
	name      string
	category  string
	shortDesc string
	longDesc  string
	usage     string
	subCmd    happy.Command // set if subcommand was called

	beforeFn       happy.Action
	doFn           happy.Action
	afterFailureFn happy.Action
	afterSuccessFn happy.Action
	afterAlwaysFn  happy.Action

	subCommands map[string]happy.Command // subcommands
	parent      happy.Command
	parents     []string

	flags varflag.Flags

	isWrapperCommand bool
	errs             error

	services []string
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

func (c *Command) Category() string {
	return c.category
}

// SetShortDesc sets commands short description
// used when describing command within list.
func (c *Command) SetShortDesc(desc string) {
	c.shortDesc = desc
}

func (c *Command) ShortDesc() string {
	return c.shortDesc
}

// Before is first function called if it is set.
// It will continue executing worker queue set within this function until first
// failure occurs which is not allowed to continue.
func (c *Command) Before(action happy.Action) {
	c.beforeFn = action
}

// Do should contain main of this command
// This function is called when:
//   - BeforeFunc is not set
//   - BeforeFunc succeeds
//   - BeforeFunc fails but failed tasks have status "allow failure"
func (c *Command) Do(action happy.Action) {
	c.doFn = action
}

// AfterSuccess is called when AfterFailure states that there has been no failures.
// This function is called when:
//   - AfterFailure states that there has been no fatal errors
func (c *Command) AfterSuccess(action happy.Action) {
	c.afterSuccessFn = action
}

// AfterFailure is called when DoFunc fails.
// This function is called when:
//   - DoFunc is not set (this case default AfterFailure function is called)
//   - DoFunc task fails which has no mark "allow failure"
func (c *Command) AfterFailure(action happy.Action) {
	c.afterFailureFn = action
}

// AfterAlways is final function called and is waiting until all tasks whithin
// AfterFailure and/or AfterSuccess functions are done executing.
// If this function if set then it is called always regardless what was the final state of
// any previous phase.
func (c *Command) AfterAlways(action happy.Action) {
	c.afterAlwaysFn = action
}

// AddFlag adds provided flag to command or subcommand.
// Invalid flag will add error to multierror and prevents application to start.
func (c *Command) AddFlag(f varflag.Flag) error {
	// Add flag if there was no errors while verifying the flag
	c.flags.Add(f)
	return nil
}

func (c *Command) Flags() varflag.Flags {
	return c.flags
}

// AcceptsFlags returns true if command accepts any flags.
func (c *Command) AcceptsFlags() bool {
	if c.subCmd != nil {
		return c.subCmd.AcceptsFlags()
	}
	return c.flags != nil && c.flags.Len() > 0
}

// Verify ranges over command flags and the sub commands
//   - verify that commands are valid and have atleast Do function
//   - verify that subcommand do not shadow flags of any parent command
func (c *Command) Verify() error {
	if c.errs != nil {
		return fmt.Errorf("%w", c.errs)
	}
	// Chck commands name
	if !config.ValidSlug(c.name) {
		return fmt.Errorf("%w: command name (%s) is invalid - must match following regex %s", ErrCommand, c.name, config.SlugRe)
	}
	if c.doFn == nil { //nolint: nestif
		if !c.isWrapperCommand {
			c.isWrapperCommand = len(c.subCommands) > 0
		}
		// Wrpper prints help by default
		if c.isWrapperCommand { //nolint: gocritic
			c.doFn = func(ctx happy.Session) error {
				HelpCommand(ctx)
				return nil
			}
		} else if c.subCommands != nil {
			goto SubCommands
		} else {
			return fmt.Errorf("%w: command (%s) must have DoFn or configured as Wrapper Command", ErrCommand, c.name)
		}
	}

SubCommands:
	if c.subCommands != nil {
		for _, cmd := range c.subCommands {
			err := cmd.Verify()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Command) SubCommand(name string) (cmd happy.Command, exists bool) {
	if cmd, exists := c.subCommands[name]; exists {
		return cmd, exists
	}
	return
}

// HasSubcommands returns true if command has any subcommands.
func (c *Command) HasSubcommands() bool {
	if c.subCmd != nil {
		return c.subCmd.HasSubcommands()
	}
	return c.subCommands != nil && len(c.subCommands) > 0
}

func (c *Command) ExecuteBeforeFn(ctx happy.Session) error {
	if c.subCmd != nil {
		return c.subCmd.ExecuteBeforeFn(ctx)
	}
	if c.beforeFn == nil {
		return nil
	}
	return c.beforeFn(ctx)
}

func (c *Command) ExecuteDoFn(ctx happy.Session) error {

	if c.subCmd != nil {
		return c.subCmd.ExecuteDoFn(ctx)
	}

	if c.doFn == nil {
		return nil
	}
	var (
		wg  sync.WaitGroup
		err error
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %s", ErrPanic, fmt.Sprint(r))
		}
		err = c.doFn(ctx)
	}()
	wg.Wait()

	return err
}

func (c *Command) ExecuteAfterSuccessFn(ctx happy.Session) error {
	if c.subCmd != nil {
		return c.subCmd.ExecuteAfterSuccessFn(ctx)
	}
	if c.afterSuccessFn == nil {
		return nil
	}
	return c.afterSuccessFn(ctx)
}

func (c *Command) ExecuteAfterFailureFn(ctx happy.Session) error {
	if c.subCmd != nil {
		return c.subCmd.ExecuteAfterFailureFn(ctx)
	}
	if c.afterFailureFn == nil {
		return nil
	}
	return c.afterFailureFn(ctx)
}

func (c *Command) ExecuteAfterAlwaysFn(ctx happy.Session) error {
	if c.subCmd != nil {
		return c.subCmd.ExecuteAfterAlwaysFn(ctx)
	}

	if c.afterAlwaysFn == nil {
		return nil
	}
	return c.afterAlwaysFn(ctx)
}

// AcceptsFlags returns true if command accepts any flags.
func (c *Command) Args() []vars.Value {
	fs, ok := (c.flags).(*varflag.FlagSet)
	if !ok {
		return []vars.Value{}
	}
	return fs.Args()
}

// AcceptsArgs returns true if command accepts any arguments.
func (c *Command) AcceptsArgs() bool {
	return c.flags.AcceptsArgs()
}

// AddSubcommand to application which are verified in application startup.
func (c *Command) AddSubCommand(cmd happy.Command) {
	if cmd == nil {
		return
	}

	if c.subCommands == nil {
		c.subCommands = make(map[string]happy.Command)
	}

	cmd.SetParents(append(c.parents, c.name))

	// cmd.i18n = c.i18n

	c.subCommands[cmd.String()] = cmd

	c.flags.AddSet(cmd.Flags())
	cmd.SetParent(c)
}
func (c *Command) SetParent(parent happy.Command) {
	c.parent = parent
}

// GetSubcommands returns slice with all subcommands for the command.
func (c *Command) Subcommands() (scmd []happy.Command) {
	if c.subCmd != nil {
		return c.subCmd.Subcommands()
	}
	for _, cmd := range c.subCommands {
		scmd = append(scmd, cmd)
	}
	return scmd
}

// SetParents sets command parent cmds.
func (c *Command) SetParents(parents []string) {
	c.parents = parents
}

// GetParents return slice with names of parent commands.
func (c *Command) Parents() []string {
	if c.subCmd != nil {
		return c.subCmd.Parents()
	}
	return c.parents
}

func (c *Command) Parent() happy.Command {
	return c.parent
}

// Usage adds extra usage string following the (app-name cmd-name ...)
// auto generated usage string would be something like:
//   `app-name cmd-name [flags] [subcommand] [flags] [args]`
// this text would appear next line
// Usage returns commands usage string.
func (c *Command) Usage(usage ...string) string {
	if c.subCmd != nil {
		return c.subCmd.Usage(usage...)
	}
	if len(usage) > 0 {
		c.usage = strings.TrimSpace(usage[0])
	}
	if c.usage != "" {
		return c.usage
	}
	return ""
}

// LongDesc returns commands long description.
func (c *Command) LongDesc(desc ...string) string {
	if c.subCmd != nil {
		return c.subCmd.LongDesc(desc...)
	}
	if len(desc) > 0 {
		c.longDesc = desc[0]
	}
	return c.longDesc
}

func (c *Command) RequireServices(srvs ...string) {
	c.services = append(c.services, srvs...)
}

func (c *Command) ServiceLoaders() (urls []string) {
	return c.services
}
