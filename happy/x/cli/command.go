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
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"github.com/mkungla/happy/x/pkg/varflag"
)

// New returns new happy.Command with given slug and provided configuration options.
func NewCommand(cmd string, options ...happy.OptionSetFunc) (happy.Command, happy.Error) {
	s, err := happyx.NewSlug(cmd)
	if err != nil {
		return nil, ErrCommand.Wrap(err)
	}

	opts, err := happyx.OptionsToVariables(options...)
	if err != nil {
		return nil, ErrCommand.Wrap(err)
	}

	c := &Command{
		slug: s,
		opts: opts,
	}

	flags, ferr := varflag.NewFlagSetAs[
		happy.Flags,    // flagset
		happy.Flags,    // sub flagset
		happy.Flag,     // flag
		happy.Variable, // flag values
		happy.Value,    // arguements
	](s.String(), -1)
	if ferr != nil {
		return nil, ErrCommand.Wrap(err)
	}
	c.flags = flags

	return c, nil
}

type Command struct {
	slug             happy.Slug
	flags            happy.Flags
	opts             happy.Variables
	subCommands      map[string]happy.Command // subcommands
	parents          []string
	parent           happy.Command
	subCommand       happy.Command
	errs             []happy.Error
	services         []happy.URL
	isWrapperCommand bool
	hasSubcommands   bool

	beforeAction       happy.ActionCommandFunc
	doAction           happy.ActionCommandFunc
	afterSuccessAction happy.ActionFunc
	afterFailureAction happy.ActionWithErrorFunc
	afterAlwaysAction  happy.ActionWithErrorFunc

	desc      string
	category  string
	usageDesc string
}

func (c *Command) Slug() happy.Slug {
	if c.subCommand != nil {
		return c.subCommand.Slug()
	}
	return c.slug
}

func (c *Command) Category() string {
	return c.category
}

func (c *Command) Description() string {
	return c.desc
}

func (c *Command) UsageDescription() string {
	return c.usageDesc
}

func (c *Command) HasSubcommands() bool {
	if c.subCommand != nil {
		return c.subCommand.HasSubcommands()
	}
	return c.hasSubcommands
}

// AddFlag adds provided flag to command or subcommand.
// Invalid flag will add error to multierror and prevents application to start.
func (c *Command) AddFlag(f happy.Flag) {
	err := c.flags.Add(f)
	if err != nil {
		c.errs = append(c.errs, ErrCommandFlags.Wrap(err))
	}
}

func (c *Command) AddFlags(flagFuncs ...happy.FlagCreateFunc) {
	for _, flagFunc := range flagFuncs {
		flag, err := flagFunc()
		if err != nil {
			c.errs = append(c.errs, err)
			return
		}
		c.AddFlag(flag)
	}
}

// AddSubcommand to application which are verified in application startup.
func (c *Command) AddSubCommand(cmd happy.Command) {
	if cmd == nil {
		c.errs = append(c.errs, ErrCommand.WithText("adding <nil> subcommand to "+c.slug.String()))
		return
	}

	if c.subCommands == nil {
		c.subCommands = make(map[string]happy.Command)
	}

	cmd.SetParents(append(c.parents, c.slug.String()))

	// cmd.i18n = c.i18n

	c.subCommands[cmd.Slug().String()] = cmd

	if err := c.flags.AddSet(cmd.Flags()); err != nil {
		c.errs = append(c.errs, ErrCommand.WithTextf(
			"failed to attach subcommand %s flags to %s",
			cmd.Slug().String(),
			c.slug.String()))
		return
	}
	cmd.AttachParent(c)
	c.hasSubcommands = true
}

func (c *Command) AddSubCommands(cmdFuncs ...happy.CommandCreateFunc) {
	for _, cmdFunc := range cmdFuncs {
		cmd, err := cmdFunc()
		if err != nil {
			c.errs = append(c.errs, ErrCommand.Wrap(err))
			return
		}
		c.AddSubCommand(cmd)
	}
}

func (c *Command) AttachParent(parent happy.Command) {
	c.parent = parent
}

func (c *Command) Parent() (parent happy.Command) {
	return c.parent
}

// SetParents sets command parent cmds.
func (c *Command) SetParents(parents []string) {
	c.parents = parents
}

func (c *Command) Parents() (parents []string) {
	return c.parents
}

func (c *Command) Flags() happy.Flags {
	return c.flags
}

// Flag looks up flag with given name and returns flags.Interface.
// If no flag was found empty bool flag will be returned.
// Instead of returning error you can check returned flags .IsPresent.
func (c *Command) Flag(name string) happy.Flag {
	f, err := c.flags.Get(name)
	if err != nil {
		c.errs = append(c.errs, ErrCommandFlags.Wrap(err))
		vf, err := varflag.Bool(name, false, "")
		if err != nil {
			c.errs = append(c.errs, ErrCommandFlags.Wrap(err))
		}
		f = varflag.AsFlag[happy.Flag, happy.Variable, happy.Value](vf)
	}
	return f
}

// Before is first function called if it is set.
// It will continue executing worker queue set within this function until first
// failure occurs which is not allowed to continue.
func (c *Command) Before(action happy.ActionCommandFunc) {
	if c.beforeAction != nil {
		c.errs = append(c.errs, ErrCommand.WithText("attempt to override Before action for "+c.slug.String()))
		return
	}
	c.beforeAction = action
}

// Do should contain main of this command
// This function is called when:
//   - BeforeFunc is not set
//   - BeforeFunc succeeds
//   - BeforeFunc fails but failed tasks have status "allow failure"
func (c *Command) Do(action happy.ActionCommandFunc) {
	if c.doAction != nil {
		c.errs = append(c.errs, ErrCommand.WithText("attempt to override Do action for "+c.slug.String()))
		return
	}
	c.doAction = action
}

// AfterSuccess is called when AfterFailure states that there has been no failures.
// This function is called when:
//   - AfterFailure states that there has been no fatal errors
func (c *Command) AfterSuccess(action happy.ActionFunc) {
	if c.afterSuccessAction != nil {
		c.errs = append(c.errs, ErrCommand.WithText("attempt to override AfterSuccess action for "+c.slug.String()))
		return
	}
	c.afterSuccessAction = action
}

// AfterFailure is called when DoFunc fails.
// This function is called when:
//   - DoFunc is not set (this case default AfterFailure function is called)
//   - DoFunc task fails which has no mark "allow failure"
func (c *Command) AfterFailure(action happy.ActionWithErrorFunc) {
	if c.afterFailureAction != nil {
		c.errs = append(c.errs, ErrCommand.WithText("attempt to override AfterFailure action for "+c.slug.String()))
		return
	}
	c.afterFailureAction = action
}

// AfterAlways is final function called and is waiting until all tasks whithin
// AfterFailure and/or AfterSuccess functions are done executing.
// If this function if set then it is called always regardless what was the final state of
// any previous phase.
func (c *Command) AfterAlways(action happy.ActionWithErrorFunc) {
	if c.afterAlwaysAction != nil {
		c.errs = append(c.errs, ErrCommand.WithText("attempt to override AfterAlways action for "+c.slug.String()))
		return
	}
	c.afterAlwaysAction = action
}

func (c *Command) WithServices(urls ...happy.URL) {
	c.services = append(c.services, urls...)
}

func (c *Command) Err() happy.Error {
	if len(c.errs) == 0 {
		return nil
	}
	err := ErrCommand
	for _, e := range c.errs {
		err = err.Wrap(e)
	}
	return err
}

// Verify veifies command,  flags and the sub commands
//   - verify that commands are valid and have atleast Do function
//   - verify that subcommand do not shadow flags of any parent command
func (c *Command) Verify() happy.Error {
	if len(c.errs) > 0 {
		return c.Err()
	}

	c.usageDesc = c.opts.Get("usage.decription").String()
	c.category = c.opts.Get("category").String()
	c.desc = c.opts.Get("description").String()

	if c.doAction == nil { //nolint: nestif
		if !c.isWrapperCommand {
			c.isWrapperCommand = len(c.subCommands) > 0
		}
		c.doAction = func(session happy.Session, flags happy.Flags, assets happy.FS, status happy.ApplicationStatus) error {
			HelpCommand(session, c)
			return nil
		}

		// Wrpper prints help by default
		if c.subCommands != nil {
			goto SubCommands
		} else {
			return ErrCommand.WithTextf("command (%s) must have Do action or atleeast one subcommand", c.slug.String())
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

// SubCommands returns slice with all subcommands for the command.
func (c *Command) SubCommands() (scmd []happy.Command) {
	if c.subCommand != nil {
		return c.subCommand.SubCommands()
	}
	for _, cmd := range c.subCommands {
		scmd = append(scmd, cmd)
	}
	return scmd
}

func (c *Command) SubCommand(name string) (cmd happy.Command, exists bool) {
	if cmd, exists := c.subCommands[name]; exists {
		return cmd, exists
	}
	return
}

func (c *Command) ExecuteBeforeAction(session happy.Session, assets happy.FS, status happy.ApplicationStatus) happy.Error {
	if c.beforeAction == nil {
		return nil
	}

	if err := c.beforeAction(session, c.flags, assets, status); err != nil {
		return ErrCommandAction.WithText(c.slug.String()).Wrap(err)
	}
	return nil
}

func (c *Command) ExecuteDoAction(session happy.Session, assets happy.FS, status happy.ApplicationStatus) (err happy.Error) {
	if c.doAction == nil {
		return nil
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = ErrPanic.WithTextf("%s(panic): %v", r)
			}
		}()
		if e := c.doAction(session, c.flags, assets, status); e != nil {
			err = ErrCommandAction.WithText(c.slug.String()).Wrap(e)
		}
	}()
	return
}

func (c *Command) ExecuteAfterFailureAction(sess happy.Session, err happy.Error) happy.Error {
	if c.afterFailureAction == nil {
		return nil
	}

	if e := c.afterFailureAction(sess, err); e != nil {
		return ErrCommandAction.WithText(c.slug.String()).Wrap(e)
	}
	return nil
}

func (c *Command) ExecuteAfterSuccessAction(sess happy.Session) happy.Error {
	if c.afterSuccessAction == nil {
		return nil
	}

	if e := c.afterSuccessAction(sess); e != nil {
		return ErrCommandAction.WithText(c.slug.String()).Wrap(e)
	}
	return nil
}

func (c *Command) ExecuteAfterAlwaysAction(sess happy.Session, err happy.Error) happy.Error {
	if c.afterAlwaysAction == nil {
		return nil
	}

	if e := c.afterAlwaysAction(sess, err); e != nil {
		return ErrCommandAction.WithText(c.slug.String()).Wrap(e)
	}
	return nil
}
