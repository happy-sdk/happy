// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package command

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/logging"
)

var (
	Error          = errors.New("command error")
	ErrFlags       = errors.New("command flags error")
	ErrHasNoParent = errors.New("command has no parent command")
)

type Config struct {
	Name             settings.String `key:"name"`
	Usage            settings.String `key:"usage" mutation:"once"`
	HideDefaultUsage settings.Bool   `key:"hide_default_usage" default:"false"`
	Category         settings.String `key:"category"`
	Description      settings.String `key:"description"`
	// MinArgs Minimum argument count for command
	MinArgs    settings.Uint `key:"min_args" default:"0" mutation:"once"`
	MinArgsErr settings.String
	// MaxArgs Maximum argument count for command
	MaxArgs    settings.Uint `key:"max_args" default:"0" mutation:"once"`
	MaxArgsErr settings.String
	// SharedBeforeAction share Before action for all its subcommands
	SharedBeforeAction settings.Bool `key:"shared_before_action" default:"false"`
	// Indicates that the command should be executed immediately, without waiting for the full runtime setup.
	Immediate settings.Bool `key:"immediate" default:"false"`
	// SkipSharedBefore indicates that the BeforeAlways any shared before actions provided
	// by parent commands should be skipped.
	SkipSharedBefore settings.Bool `key:"skip_shared_before" default:"false"`
}

func (s Config) Blueprint() (*settings.Blueprint, error) {

	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type Command struct {
	mu    sync.Mutex
	cnf   *settings.Profile
	info  []string
	usage []string

	flags       varflag.Flags
	parent      *Command
	subCommands map[string]*Command

	beforeAction       action.WithArgs
	doAction           action.WithArgs
	afterSuccessAction action.Action
	afterFailureAction action.WithPrevErr
	afterAlwaysAction  action.WithPrevErr

	isWrapperCommand bool

	parents []string

	catdesc map[string]string

	logName string
	err     error

	cnflog *logging.QueueLogger

	extraUsage []string
}

func New(s Config) *Command {
	c := &Command{
		catdesc: make(map[string]string),
		cnflog:  logging.NewQueueLogger(),
	}
	if err := c.configure(&s); err != nil {
		c.err = fmt.Errorf("%w: %s", Error, err.Error())
		return c
	}

	name := c.cnf.Get("name").String()
	flags, err := varflag.NewFlagSet(name, c.cnf.Get("max_args").Value().Int())
	if errors.Is(err, varflag.ErrInvalidFlagSetName) {
		c.error(fmt.Errorf("%w: invalid command name %q", Error, name))
	}
	c.flags = flags

	return c
}

func (c *Command) AfterAlways(a action.WithPrevErr) *Command {
	if !c.tryLock("AfterAlways") {
		return c
	}
	defer c.mu.Unlock()
	if c.afterAlwaysAction != nil {
		c.error(fmt.Errorf("%w: attempt to override AfterAlways action for %s", Error, c.cnf.Get("name").String()))
		return c
	}
	c.afterAlwaysAction = a
	return c
}

func (c *Command) AfterFailure(a action.WithPrevErr) *Command {
	if !c.tryLock("AfterFailure") {
		return c
	}
	defer c.mu.Unlock()
	if c.afterFailureAction != nil {
		c.error(fmt.Errorf("%w: attempt to override AfterFailure action for %s", Error, c.cnf.Get("name").String()))
		return c
	}
	c.afterFailureAction = a
	return c
}

func (c *Command) AfterSuccess(a action.Action) *Command {
	if !c.tryLock("AfterSuccess") {
		return c
	}
	defer c.mu.Unlock()
	if c.afterSuccessAction != nil {
		c.error(fmt.Errorf("%w: attempt to override AfterSuccess action for %s", Error, c.cnf.Get("name").String()))
		return c
	}
	c.afterSuccessAction = a
	return c
}

func (c *Command) Before(a action.WithArgs) *Command {
	if !c.tryLock("Before") {
		return c
	}
	defer c.mu.Unlock()

	if c.beforeAction != nil {
		c.error(fmt.Errorf("%w: attempt to override Before action for %s", Error, c.cnf.Get("name").String()))
		return c
	}
	c.beforeAction = a
	return c
}

func (c *Command) DescribeCategory(cat, desc string) *Command {
	if !c.tryLock("DescribeCategory") {
		return c
	}
	defer c.mu.Unlock()
	c.catdesc[strings.ToLower(cat)] = desc
	return c
}

func (c *Command) Do(action action.WithArgs) *Command {
	if !c.tryLock("Do") {
		return c
	}
	defer c.mu.Unlock()
	if c.doAction != nil {
		c.err = fmt.Errorf("%w: attempt to override Before action for %s", Error, c.cnf.Get("name").String())
		return c
	}
	c.doAction = action
	return c
}

func (c *Command) WithFlags(ffns ...varflag.FlagCreateFunc) *Command {
	for _, fn := range ffns {
		c.withFlag(fn)
	}
	return c
}

func (c *Command) withFlag(ffn varflag.FlagCreateFunc) *Command {
	if !c.tryLock("WithFlag") {
		return c
	}
	defer c.mu.Unlock()

	f, cerr := ffn()
	if cerr != nil {
		c.error(fmt.Errorf("%w: %s", ErrFlags, cerr.Error()))
		return c
	}

	if err := c.flags.Add(f); err != nil {
		c.error(fmt.Errorf("%w: %s", ErrFlags, err.Error()))
	}
	return c
}

func (c *Command) AddInfo(paragraph string) *Command {
	if !c.tryLock("AddInfo") {
		return c
	}
	defer c.mu.Unlock()

	c.info = append(c.info, paragraph)
	return c
}

func (c *Command) WithSubCommands(cmds ...*Command) *Command {
	for _, cmd := range cmds {
		c.withSubCommand(cmd)
	}
	return c
}

func (c *Command) withSubCommand(cmd *Command) *Command {
	if !c.tryLock("WithSubCommand") {
		return c
	}
	defer c.mu.Unlock()
	if cmd == nil {
		return c
	}

	if cmd.err != nil {
		c.error(cmd.err)
		return c
	}
	if c.subCommands == nil {
		c.subCommands = make(map[string]*Command)
	}
	if err := c.flags.AddSet(cmd.flags); err != nil {
		c.error(fmt.Errorf(
			"%w: failed to attach subcommand %s flags to %s",
			Error,
			cmd.cnf.Get("name").String(),
			c.cnf.Get("name").String(),
		))
		return c
	}
	cmd.parent = c

	c.subCommands[cmd.cnf.Get("name").String()] = cmd
	return c
}

func (c *Command) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	for _, scmd := range c.subCommands {
		if err := scmd.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Command) tryLock(method string) bool {
	if !c.mu.TryLock() {
		c.cnflog.BUG(
			"command configuration failed",
			slog.String("command", c.logName),
			slog.String("method", method),
		)
		return false
	}
	return true
}

// configure is called in New so defering outside of lock is ok.
func (c *Command) configure(s *Config) error {
	defer func() {
		if c.cnf == nil {
			// set empty profile when settings have failed to load
			c.cnf = &settings.Profile{}
		}
	}()

	c.mu.Lock()

	bp, err := s.Blueprint()
	if err != nil {
		return err
	}

	// logName is used for logging purposes, when command configuration fails.
	logName, _ := bp.GetSpec("name")
	c.logName = logName.Value
	if len(c.logName) == 0 {
		c.logName = "command"
	}

	schema, err := bp.Schema("cmd", "v1")
	if err != nil {
		return err
	}
	c.cnf, err = schema.Profile("cli", nil)
	if err != nil {
		return err
	}

	if minargs := c.cnf.Get("min_args").Value().Int(); minargs > c.cnf.Get("max_args").Value().Int() {
		if err := c.cnf.Set("max_args", minargs); err != nil {
			return err
		}
	}

	// we dont defer unlock here because we want to keep it locked for tryLock on error.
	c.mu.Unlock()
	return nil
}

// Verify veifies command,  flags and the sub commands
//   - verify that commands are valid and have atleast Do function
//   - verify that subcommand do not shadow flags of any parent command
func (c *Command) verify() error {
	if !c.tryLock("verify") {
		return fmt.Errorf("%w: failed to obtain lock to verify command (%s)", Error, c.logName)
	}
	defer c.mu.Unlock()

	if c.err != nil {
		return c.err
	}
	name := c.cnf.Get("name").String()

	var usage []string
	usage = append(usage, c.parents...)
	usage = append(usage, name)
	if c.flags.Len() > 0 {
		usage = append(usage, "[flags]")
	}
	if c.subCommands != nil {
		usage = append(usage, "[subcommand]")
	}
	c.usage = append(c.usage, strings.Join(usage, " "))

	if c.flags.AcceptsArgs() {
		var withargs []string
		withargs = append(withargs, c.parents...)
		withargs = append(withargs, name)
		withargs = append(withargs, "[args...]")
		withargs = append(withargs, fmt.Sprintf(
			" // min %d max %d",
			c.cnf.Get("min_args").Value().Int(),
			c.cnf.Get("max_args").Value().Int(),
		))
		c.usage = append(c.usage, strings.Join(withargs, " "))
	}

	defineUsage := c.cnf.Get("usage").String()
	if defineUsage != "" {
		var usage []string
		usage = append(usage, c.parents...)
		usage = append(usage, name)
		usage = append(usage, defineUsage)
		if c.cnf.Get("hide_default_usage").Value().Bool() {
			c.usage = []string{strings.Join(usage, " ")}
		} else {
			c.usage = append(c.usage, strings.Join(usage, " "))
		}
	}

	if len(c.extraUsage) > 0 {
		for _, u := range c.extraUsage {
			var usage []string
			usage = append(usage, c.parents...)
			usage = append(usage, name)
			usage = append(usage, u)
			c.usage = append(c.usage, strings.Join(usage, " "))
		}
	}

	if c.err != nil {
		return c.err
	}

	if c.doAction == nil {
		if !c.isWrapperCommand {
			c.isWrapperCommand = len(c.subCommands) > 0
		}

		if c.subCommands != nil {
			goto SubCommands
		} else {
			return fmt.Errorf("%w: command (%s) must have Do action or atleeast one subcommand", Error, name)
		}
	}

SubCommands:
	if c.subCommands != nil {
		for _, cmd := range c.subCommands {
			// Add subcommand loogs to parent command log queue
			if err := c.cnflog.ConsumeQueue(cmd.cnflog); err != nil {
				return err
			}
			cmd.parents = append(c.parents, name)
			if err := cmd.verify(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Command) Usage(usage string) {
	if !c.tryLock("Usage") {
		return
	}
	defer c.mu.Unlock()
	c.extraUsage = append(c.extraUsage, usage)
}

func (c *Command) getActiveCommand() (*Command, error) {
	subtree := c.flags.GetActiveSets()

	// Skip self
	for _, subset := range subtree {
		cmd, exists := c.getSubCommand(subset.Name())
		if exists {
			return cmd.getActiveCommand()
		}
	}

	args := c.flags.Args()
	if !c.flags.AcceptsArgs() && len(args) > 0 {
		return nil, fmt.Errorf("%w: unknown subcommand: %s for %s", Error, args[0].String(), c.logName)
	}

	return c, nil
}

func (c *Command) getSubCommand(name string) (cmd *Command, exists bool) {
	if cmd, exists := c.subCommands[name]; exists {
		return cmd, exists
	}
	return
}

func (c *Command) getGlobalFlags() varflag.Flags {
	if c.parent == nil {
		return c.flags
	}
	return c.parent.getGlobalFlags()
}

func (c *Command) getSharedFlags() (varflag.Flags, error) {
	// ignore global flags
	if c.parent == nil || c.parent.parent == nil {
		flags, err := varflag.NewFlagSet("x-"+c.cnf.Get("name").String()+"-noparent", 0)
		if err != nil {
			return nil, err
		}
		return flags, ErrHasNoParent
	}

	flags := c.parent.getFlags()
	if flags == nil {
		flags, _ = varflag.NewFlagSet(c.parent.cnf.Get("name").String(), 0)
	}
	parentFlags, err := c.parent.getSharedFlags()
	if err != nil && !errors.Is(err, ErrHasNoParent) {
		return nil, err
	}

	if parentFlags != nil {
		for _, flag := range parentFlags.Flags() {
			if err := flags.Add(flag); err != nil {
				return nil, err
			}
		}
	}

	return flags, nil
}

func (c *Command) getFlags() varflag.Flags {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.flags
}

func (c *Command) error(e error) {
	if c.err != nil {
		return
	}
	c.err = e
}
