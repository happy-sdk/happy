// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy

import (
	"errors"
	"fmt"
	"sync"

	"log/slog"

	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk"
)

var (
	ErrCommand       = errors.New("command error")
	ErrCommandFlags  = errors.New("command flags error")
	ErrCommandAction = errors.New("command action error")
)

type Command struct {
	mu       sync.Mutex
	name     string
	usage    string
	desc     string
	category string

	flags       varflag.Flags
	parent      *Command
	subCommands map[string]*Command

	beforeAction       ActionWithArgs
	doAction           ActionWithArgs
	afterSuccessAction Action
	afterFailureAction ActionWithPrevErr
	afterAlwaysAction  ActionWithPrevErr

	isWrapperCommand    bool
	allowOnFreshInstall bool
	skipAddons          bool

	errs []error

	parents []string

	argcmax uint
	argcmin uint
}

func NewCommand(name string, options ...OptionArg) *Command {
	c := &Command{}

	n, err := vars.ParseKey(name)
	c.errs = append(c.errs, err)
	c.name = n

	opts, err := NewOptions(fmt.Sprintf("cmd.%s", name), getDefaultCommandOpts())
	c.errs = append(c.errs, err)

	for _, opt := range options {
		if err := opt.apply(opts); err != nil {
			c.errs = append(c.errs, err)
		}
	}

	if err := opts.setDefaults(); err != nil {
		c.errs = append(c.errs, err)
	}
	c.argcmin = opts.Get("argc.min").Uint()
	c.argcmax = opts.Get("argc.max").Uint()

	if c.argcmin > c.argcmax {
		c.argcmax = c.argcmin
	}

	flags, err := varflag.NewFlagSet(name, int(c.argcmax))
	c.errs = append(c.errs, err)
	c.flags = flags

	c.usage = opts.Get("usage").String()
	c.desc = opts.Get("description").String()
	c.category = opts.Get("category").String()
	c.allowOnFreshInstall = opts.Get("init.allowed").Bool()
	c.skipAddons = opts.Get("skip.addons").Bool()

	return c
}

func (c *Command) Name() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.name
}

func (c *Command) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return errors.Join(c.errs...)
}

func (c *Command) Usage() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.usage
}

func (c *Command) Description() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.desc
}

func (c *Command) Parents() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.parents
}

func (c *Command) tryLock(method string) bool {
	if !c.mu.TryLock() {
		slog.Warn(
			"wrong command usage, should not be called when application is running",
			slog.String("command", c.name),
			slog.String("method", method),
		)
		return false
	}
	return true
}

func (c *Command) AddFlag(f varflag.Flag) {
	if !c.tryLock("AddFlag") {
		return
	}
	defer c.mu.Unlock()

	err := c.flags.Add(f)
	if err != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: %s", ErrCommandFlags, err.Error()))
	}
}

func (c *Command) Before(a ActionWithArgs) {
	if !c.tryLock("Before") {
		return
	}
	defer c.mu.Unlock()

	if c.beforeAction != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: attempt to override Before action for %s", ErrCommand, c.name))
		return
	}
	c.beforeAction = a
}

func (c *Command) Do(action ActionWithArgs) {
	if !c.tryLock("Do") {
		return
	}
	defer c.mu.Unlock()
	if c.doAction != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: attempt to override Before action for %s", ErrCommand, c.name))
		return
	}
	c.doAction = action
}

func (c *Command) AfterSuccess(a Action) {
	if !c.tryLock("AfterSuccess") {
		return
	}
	defer c.mu.Unlock()
	if c.afterSuccessAction != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: attempt to override AfterSuccess action for %s", ErrCommand, c.name))
		return
	}
	c.afterSuccessAction = a
}

func (c *Command) AfterFailure(a ActionWithPrevErr) {
	if !c.tryLock("AfterFailure") {
		return
	}
	defer c.mu.Unlock()
	if c.afterFailureAction != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: attempt to override AfterFailure action for %s", ErrCommand, c.name))
		return
	}
	c.afterFailureAction = a
}

func (c *Command) AfterAlways(a ActionWithPrevErr) {
	if !c.tryLock("AfterAlways") {
		return
	}
	defer c.mu.Unlock()
	if c.afterAlwaysAction != nil {
		c.errs = append(c.errs, fmt.Errorf("%w: attempt to override AfterAlways action for %s", ErrCommand, c.name))
		return
	}
	c.afterAlwaysAction = a
}

func (c *Command) AddSubCommand(cmd *Command) {
	if !c.tryLock("AddSubCommand") {
		return
	}
	defer c.mu.Unlock()
	if c.subCommands == nil {
		c.subCommands = make(map[string]*Command)
	}

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
		err := errors.Join(c.errs...)
		if err != nil {
			return err
		}
	}

	if c.doAction == nil {
		if !c.isWrapperCommand {
			c.isWrapperCommand = len(c.subCommands) > 0
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
			cmd.parents = append(c.parents, c.name)
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

func (c *Command) getFlags() varflag.Flags {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.flags
}

func (c *Command) getSubCommand(name string) (cmd *Command, exists bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cmd, exists := c.subCommands[name]; exists {
		return cmd, exists
	}
	return
}
func (c *Command) getSubCommands() (cmds []*Command) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cmd := range c.subCommands {
		cmds = append(cmds, cmd)
	}
	return
}

func (c *Command) getActiveCommand() (*Command, error) {
	subtree := c.flags.GetActiveSets()

	// Skip self
	for _, subset := range subtree[1:] {
		cmd, exists := c.getSubCommand(subset.Name())
		if exists {
			return cmd.getActiveCommand()
		}
	}

	args := c.flags.Args()
	if !c.flags.AcceptsArgs() && len(args) > 0 {
		return nil, fmt.Errorf("%w: unknown subcommand: %s for %s", ErrCommand, args[0].String(), c.name)
	}

	return c, nil
}

func (c *Command) callBeforeAction(sess *Session) error {
	if c.beforeAction == nil {
		return nil
	}

	args := sdk.NewArgs(c.flags.Args(), c.flags)

	if c.argcmin == 0 && c.argcmax == 0 && args.Argn() > 0 {
		return fmt.Errorf("%w: %s: %s", ErrCommandAction, c.name, "command does not accept arguments")
	}

	if args.Argn() < c.argcmin {
		return fmt.Errorf("%w: %s: command requires min %d arguments, %d provided", ErrCommandAction, c.name, c.argcmin, args.Argn())
	}
	if args.Argn() > c.argcmax {
		return fmt.Errorf("%w: %s: command accepts max %d arguments, %d provided", ErrCommandAction, c.name, c.argcmax, args.Argn())
	}

	if err := c.beforeAction(sess, args); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrCommandAction, c.name, err)
	}
	return nil
}

func (c *Command) callDoAction(session *Session) error {
	if c.doAction == nil {
		return nil
	}

	args := sdk.NewArgs(c.flags.Args(), c.flags)

	if err := c.doAction(session, args); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrCommandAction, c.name, err)
	}
	return nil
}

func (c *Command) callAfterFailureAction(session *Session, err error) error {
	if c.afterFailureAction == nil {
		return nil
	}

	if err := c.afterFailureAction(session, err); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrCommandAction, c.name, err)
	}
	return nil
}

func (c *Command) callAfterSuccessAction(session *Session) error {
	if c.afterSuccessAction == nil {
		return nil
	}

	if err := c.afterSuccessAction(session); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrCommandAction, c.name, err)
	}
	return nil
}

func (c *Command) callAfterAlwaysAction(session *Session, err error) error {
	if c.afterAlwaysAction == nil {
		return nil
	}

	if err := c.afterAlwaysAction(session, err); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrCommandAction, c.name, err)
	}
	return nil
}