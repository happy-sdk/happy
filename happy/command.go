// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"errors"
	"fmt"

	"github.com/mkungla/happy/pkg/happylog"
	"github.com/mkungla/happy/pkg/varflag"
	"github.com/mkungla/happy/pkg/vars"
)

type Command struct {
	name        string
	usage       string
	desc        string
	category    string
	flags       varflag.Flags
	parent      *Command
	subCommands map[string]Command

	beforeAction       Action
	doAction           Action
	afterSuccessAction Action
	afterFailureAction func(s *Session, err error) error
	afterAlwaysAction  Action

	isWrapperCommand bool
	errs             []error

	parents []string
}

func NewCommand(name string, options ...OptionAttr) (Command, error) {
	s, err := vars.ParseKey(name)
	if err != nil {
		return Command{}, errors.Join(ErrCommand, err)
	}

	flags, err := varflag.NewFlagSet(name, -1)
	if err != nil {
		return Command{}, errors.Join(ErrCommand, err)
	}
	c := Command{
		name:  s,
		flags: flags,
	}

	opts, err := NewOptions("command", getDefaultCommandOpts())
	if err != nil {
		return Command{}, errors.Join(ErrCommand, err)
	}
	for _, opt := range options {
		if err := opt.apply(opts); err != nil {
			return Command{}, errors.Join(ErrCommand, err)
		}
	}

	c.usage = opts.Get("usage").String()
	c.desc = opts.Get("description").String()
	c.category = opts.Get("description").String()

	return c, nil
}

func (c *Command) Name() string {
	return c.name
}

func (c *Command) Usage() string {
	return c.usage
}

func (c *Command) AddFlag(f varflag.Flag) {
	err := c.flags.Add(f)
	if err != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: %s", ErrCommandFlags, err.Error()))
	}
}

func (c *Command) Before(action func(s *Session) error) {
	if c.beforeAction != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: attempt to override Before action for %s", ErrCommand, c.name))
		return
	}
	c.beforeAction = action
}

func (c *Command) Do(action func(s *Session) error) {
	if c.doAction != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: attempt to override Before action for %s", ErrCommand, c.name))
		return
	}
	c.doAction = action
}

func (c *Command) AfterSuccess(action func(s *Session) error) {
	if c.afterSuccessAction != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: attempt to override AfterSuccess action for %s", ErrCommand, c.name))
		return
	}
	c.afterSuccessAction = action
}

func (c *Command) AfterFailure(action func(s *Session, err error) error) {
	if c.afterFailureAction != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: attempt to override AfterFailure action for %s", ErrCommand, c.name))
		return
	}
	c.afterFailureAction = action
}

func (c *Command) AfterAlways(action func(s *Session) error) {
	if c.afterAlwaysAction != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: attempt to override AfterAlways action for %s", ErrCommand, c.name))
		return
	}
	c.afterAlwaysAction = action
}

func (c *Command) AddSubCommand(cmd Command) {
	if c.subCommands == nil {
		c.subCommands = make(map[string]Command)
	}
	cmd.parents = append(c.parents, c.name)

	if err := c.flags.AddSet(cmd.flags); err != nil {
		c.errs = append(c.errs, fmt.Errorf(
			"%w: failed to attach subcommand %s flags to %s",
			ErrCommand,
			cmd.name,
			c.name,
		))
		return
	}
	c.subCommands[cmd.name] = cmd
}

// Verify veifies command,  flags and the sub commands
//   - verify that commands are valid and have atleast Do function
//   - verify that subcommand do not shadow flags of any parent command
func (c *Command) verify() error {
	if len(c.errs) > 0 {
		return errors.Join(c.errs...)
	}

	if c.doAction == nil {
		if !c.isWrapperCommand {
			c.isWrapperCommand = len(c.subCommands) > 0
		}

		c.doAction = func(sess *Session) error {
			happylog.NotImplemented("should show command help")
			return nil
		}
		if c.subCommands != nil {
			goto SubCommands
		} else {
			return fmt.Errorf("%w: command (%s) must have Do action or atleeast one subcommand", ErrCommand, c.name)
		}
	}
SubCommands:
	if c.subCommands != nil {
		for _, cmd := range c.subCommands {
			err := cmd.verify()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Flag looks up flag with given name and returns flags.Interface.
// If no flag was found empty bool flag will be returned.
// Instead of returning error you can check returned flags .IsPresent.
func (c *Command) flag(name string) varflag.Flag {
	f, err := c.flags.Get(name)
	if err != nil {
		f, err = varflag.Bool(name, false, "")
		if err != nil {
			c.errs = append(c.errs, fmt.Errorf("%w: %s", ErrCommandFlags, err.Error()))
		}
	}

	return f
}

func (c *Command) getSubCommand(name string) (cmd *Command, exists bool) {
	if cmd, exists := c.subCommands[name]; exists {
		return &cmd, exists
	}
	return
}

func (c *Command) callBeforeAction(session *Session) error {
	if c.beforeAction == nil {
		return nil
	}

	if err := c.beforeAction(session); err != nil {
		return fmt.Errorf("%w: %s", ErrCommandAction, c.name)
	}
	return nil
}

func (c *Command) callDoAction(session *Session) error {
	if c.doAction == nil {
		return nil
	}

	if err := c.doAction(session); err != nil {
		return fmt.Errorf("%w: %s", ErrCommandAction, c.name)
	}
	return nil
}

func (c *Command) callAfterFailureAction(session *Session, err error) error {
	if c.afterFailureAction == nil {
		return nil
	}

	if err := c.afterFailureAction(session, err); err != nil {
		return fmt.Errorf("%w: %s", ErrCommandAction, c.name)
	}
	return nil
}

func (c *Command) callAfterSuccessAction(session *Session) error {
	if c.afterSuccessAction == nil {
		return nil
	}

	if err := c.afterSuccessAction(session); err != nil {
		return fmt.Errorf("%w: %s", ErrCommandAction, c.name)
	}
	return nil
}

func (c *Command) callAfterAlwaysAction(session *Session) error {
	if c.afterAlwaysAction == nil {
		return nil
	}

	if err := c.afterAlwaysAction(session); err != nil {
		return fmt.Errorf("%w: %s", ErrCommandAction, c.name)
	}
	return nil
}
