// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/internal"
)

var (
	Error                = errors.New("command")
	ErrFlags             = fmt.Errorf("%w flags error", Error)
	ErrHasNoParent       = fmt.Errorf("%w has no parent command", Error)
	ErrCommandNotAllowed = fmt.Errorf("%w not allowed", Error)
	ErrNotImplemented    = fmt.Errorf("%w not implemented", Error)
)

type Config struct {
	Usage            settings.String `key:"usage" mutation:"once"`
	HideDefaultUsage settings.Bool   `key:"hide_default_usage" default:"false"`
	Category         settings.String `key:"category"`
	Description      settings.String `key:"description" i18n:"true"`
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
	// Disabled indicates that the command should be disabled in the command list.
	Disabled settings.Bool `key:"disabled" default:"false" mutation:"mutable"`
	// FailDisabled indicates that the command should fail when disabled.
	// If Disable action is set, the command will fail with an error message returned by action.
	// If Disable action is not set, but Disabled is true, the command will fail with an error message ErrCommandNotAllowed.
	FailDisabled settings.Bool `key:"fail_disabled" default:"false"`
}

func (s Config) Blueprint() (*settings.Blueprint, error) {

	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type originalSource struct {
	Before       string
	Do           string
	AfterSuccess string
	AfterFailure string
	AfterAlways  string
}

type Command struct {
	mu    sync.Mutex
	name  string
	cnf   *settings.Profile
	info  []string
	usage []string

	flags       varflag.Flags
	parent      *Command
	subCommands map[string]*Command

	beforeAction       action.WithArgs
	disableAction      action.Action
	doAction           action.WithArgs
	afterSuccessAction action.Action
	afterFailureAction action.WithPrevErr
	afterAlwaysAction  action.WithPrevErr

	isWrapperCommand bool

	parents []string

	catdesc map[string]string

	err error

	cnflog *logging.QueueLogger

	extraUsage []string

	sources originalSource
}

func New(name string, cnf Config) *Command {
	c := &Command{
		name:    name,
		catdesc: make(map[string]string),
		cnflog:  logging.NewQueueLogger(256),
	}

	if err := c.configure(&cnf); err != nil {
		c.error(fmt.Errorf("%w: %s", Error, err.Error()))
		return c.toInvalid()
	}

	maxArgs := c.cnf.Get("max_args").Value().Int()

	flags, err := varflag.NewFlagSet(c.name, maxArgs)
	if err != nil {
		if errors.Is(err, varflag.ErrInvalidFlagSetName) {
			c.error(fmt.Errorf("%w: invalid command name %q", Error, c.name))
		} else {
			c.error(fmt.Errorf("%w: failed to create FlagSet: %v", Error, err))
		}
		return c.toInvalid()
	}

	c.flags = flags

	return c
}

// Name returns the name of the command.
func (c *Command) Name() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.name
}

func (c *Command) SetCategory(category string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.cnf.Set("category", category); err != nil {
		c.error(fmt.Errorf("%w: failed to set category: %v", Error, err))
	}
}

func (c *Command) DescribeCategory(cat, desc string) *Command {
	if !c.tryLock("DescribeCategory") {
		return c
	}
	defer c.mu.Unlock()
	c.catdesc[strings.ToLower(cat)] = desc
	return c
}

func (c *Command) WithFlags(ffns ...varflag.FlagCreateFunc) *Command {
	for _, fn := range ffns {
		c.withFlag(fn)
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

func (c *Command) AddUsage(usage string) *Command {
	if !c.tryLock("Usage") {
		return c
	}
	defer c.mu.Unlock()
	c.extraUsage = append(c.extraUsage, usage)
	return c
}

func (c *Command) Disable(a action.Action) *Command {
	if !c.tryLock("Hide") {
		return c
	}
	defer c.mu.Unlock()

	if c.disableAction != nil {
		c.error(fmt.Errorf("%w: attempt to override Hide action for %s", Error, c.name))
		return c
	}
	c.disableAction = a
	return c
}
func (c *Command) Before(a action.WithArgs) *Command {
	if !c.tryLock("Before") {
		return c
	}
	defer c.mu.Unlock()

	if c.beforeAction != nil {
		c.error(fmt.Errorf("%w: attempt to override Before action for %s", Error, c.name))
		return c
	}
	c.beforeAction = a
	return c
}

func (c *Command) Do(action action.WithArgs) *Command {
	if !c.tryLock("Do") {
		return c
	}
	defer c.mu.Unlock()
	if c.doAction != nil {
		c.error(fmt.Errorf("%w: attempt to override Do action for %s", Error, c.name))
		return c
	}
	src, _ := internal.RuntimeCallerStr(2)
	c.sources.Do = src
	c.doAction = action
	return c
}

func (c *Command) AfterSuccess(a action.Action) *Command {
	if !c.tryLock("AfterSuccess") {
		return c
	}
	defer c.mu.Unlock()
	if c.afterSuccessAction != nil {
		c.error(fmt.Errorf("%w: attempt to override AfterSuccess action for %s", Error, c.name))
		return c
	}
	c.afterSuccessAction = a
	return c
}

func (c *Command) AfterFailure(a action.WithPrevErr) *Command {
	if !c.tryLock("AfterFailure") {
		return c
	}
	defer c.mu.Unlock()
	if c.afterFailureAction != nil {
		c.error(fmt.Errorf("%w: attempt to override AfterFailure action for %s", Error, c.name))
		return c
	}
	c.afterFailureAction = a
	return c
}

func (c *Command) AfterAlways(a action.WithPrevErr) *Command {
	if !c.tryLock("AfterAlways") {
		return c
	}
	defer c.mu.Unlock()
	if c.afterAlwaysAction != nil {
		c.error(fmt.Errorf("%w: attempt to override AfterAlways action for %s", Error, c.name))
		return c
	}
	c.afterAlwaysAction = a
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
			cmd.name,
			c.name,
		))
		return c
	}
	cmd.parent = c

	c.subCommands[cmd.name] = cmd
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

func (c *Command) tryLock(method string) bool {
	if !c.mu.TryLock() {
		c.cnflog.Log(
			context.Background(),
			logging.LevelBUG.Level(),
			"command configuration failed",
			slog.String("command", c.name),
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
		return fmt.Errorf("%w: failed to obtain lock to verify command (%s)", Error, c.name)
	}
	defer c.mu.Unlock()

	if c.err != nil {
		return c.err
	}

	var usage []string
	usage = append(usage, c.parents...)
	usage = append(usage, c.name)
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
		withargs = append(withargs, c.name)
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
		usage = append(usage, c.name)
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
			usage = append(usage, c.name)
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
			return fmt.Errorf("%w: command (%s) must have Do action or atleeast one subcommand", Error, c.name)
		}
	}

SubCommands:
	if c.subCommands != nil {
		for _, cmd := range c.subCommands {
			// Add subcommand loogs to parent command log queue
			if _, err := c.cnflog.Consume(cmd.cnflog); err != nil {
				return err
			}
			cmd.parents = append(c.parents, c.name)
			if err := cmd.verify(); err != nil {
				return err
			}
		}
	}
	return nil
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
		return nil, fmt.Errorf("%w: unknown subcommand: %s for %s", Error, args[0].String(), c.name)
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
		flags, err := varflag.NewFlagSet("x-"+c.name+"-noparent", 0)
		if err != nil {
			return nil, err
		}
		return flags, ErrHasNoParent
	}

	flags := c.parent.getFlags()
	if flags == nil {
		flags, _ = varflag.NewFlagSet(c.parent.name, 0)
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

func (c *Command) toInvalid() *Command {
	c.mu.Lock()
	defer c.mu.Unlock()

	defer func() {
		slog.Error("HAPPY COMMAND", slog.String("err", c.err.Error()))
		stackTrace := debug.Stack()
		slog.Error(string(stackTrace))
	}()

	// Ensure that the error field is set.
	if c.err == nil {
		c.error(fmt.Errorf("%w: command marked invalid", Error))
	}

	// Clear all actions to avoid any execution.
	c.beforeAction = nil
	c.doAction = nil
	c.afterSuccessAction = nil
	c.afterFailureAction = nil
	c.afterAlwaysAction = nil

	// Remove any subcommands.
	for _, subCommand := range c.subCommands {
		subCommand.toInvalid()
	}
	c.subCommands = nil

	// If flags is still nil, assign a dummy flag set to avoid nil dereference later.
	if c.flags == nil {
		// Use a dummy flag set. We assume that this call will succeed for a command marked as invalid.
		if dummy, err := varflag.NewFlagSet("invalid", 0); err == nil {
			c.flags = dummy
		} else {
			// If even this fails, log the error.
			c.error(fmt.Errorf("failed to create dummy flag set for invalid command %q",
				err.Error()))
		}
	}

	return c
}

func (c *Command) error(err error) {
	if c.cnflog != nil {
		c.cnflog.Error(err.Error())
	}
	c.err = err
}

func NotImplemented(msg string) error {
	if msg == "" {
		return ErrNotImplemented
	}
	return fmt.Errorf("%w: %s", ErrNotImplemented, msg)
}
