// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package command

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"sync"

	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/internal"
	"github.com/happy-sdk/happy/sdk/session"
)

// Command is building command chain from provided root command.
func Compile(root *Command) (*Cmd, *logging.QueueLogger, error) {

	if err := root.verify(); err != nil {
		return nil, root.cnflog, err
	}

	if !root.tryLock("Compile") {
		return nil, root.cnflog, fmt.Errorf("%w: failed to compile the command %s", Error, root.name)
	}
	defer root.mu.Unlock()

	if err := root.flags.Parse(os.Args); err != nil {
		return nil, root.cnflog, err
	}

	acmd, err := root.getActiveCommand()
	if err != nil {
		return nil, root.cnflog, err
	}

	cmd := &Cmd{
		name: root.name,
	}

	if acmd == root {
		cmd.isRoot = true
		cmd.globalFlags = root.getGlobalFlags().Flags()
	} else {
		cmd.globalFlags = root.getGlobalFlags().Flags()
		cmd.ownFlags = acmd.flags.Flags()

		for _, flag := range cmd.globalFlags {
			if err := acmd.flags.Add(flag); err != nil {
				return nil, root.cnflog, fmt.Errorf("%w: %s: %s", Error, acmd.name, err.Error())
			}
		}
		sharedf, err := acmd.getSharedFlags()
		if err != nil && !errors.Is(err, ErrHasNoParent) {
			return nil, root.cnflog, fmt.Errorf("%w: %s: %s", Error, acmd.name, err.Error())
		}
		cmd.sharedFlags = sharedf.Flags()
		for _, flag := range cmd.sharedFlags {
			if err := acmd.flags.Add(flag); err != nil {
				return nil, root.cnflog, fmt.Errorf("%w: %s: %s", Error, acmd.name, err.Error())
			}
		}
		acmd.mu.Lock()
		defer acmd.mu.Unlock()
	}

	cmd.cnf = acmd.cnf
	cmd.flags = acmd.flags

	cmd.parents = acmd.parents
	cmd.isWrapperCommand = acmd.isWrapperCommand

	cmd.usage = acmd.usage
	cmd.info = acmd.info

	cmd.disableAction = acmd.disableAction
	cmd.beforeAction = acmd.beforeAction
	cmd.doAction = acmd.doAction
	cmd.afterSuccessAction = acmd.afterSuccessAction
	cmd.afterFailureAction = acmd.afterFailureAction
	cmd.afterAlwaysAction = acmd.afterAlwaysAction

	var catdesc = make(map[string]string)
	if acmd.parent != nil {
		cmd.parent = compileParent(acmd.parent)
		if acmd.parent.catdesc != nil {
			maps.Copy(catdesc, acmd.parent.catdesc)
		}
	}

	maps.Copy(catdesc, cmd.catdesc)

	for _, scmd := range acmd.subCommands {
		cmd.subcmds = append(cmd.subcmds, &SubCmdInfo{
			Name:          scmd.name,
			Description:   scmd.cnf.Get("description").String(),
			Category:      scmd.cnf.Get("category").String(),
			Disabled:      scmd.cnf.Get("disabled").Value().Bool(),
			disableAction: scmd.disableAction,
		})
		maps.Copy(catdesc, scmd.catdesc)
	}
	cmd.catdesc = catdesc

	return cmd, root.cnflog, nil
}

func compileParent(cmd *Command) *Cmd {
	c := &Cmd{
		name:             cmd.name,
		cnf:              cmd.cnf,
		parents:          cmd.parents,
		isWrapperCommand: cmd.isWrapperCommand,
		usage:            cmd.usage,
		info:             cmd.info,
		catdesc:          cmd.catdesc,
		flags:            cmd.flags,
	}

	if c.cnf.Get("shared_before_action").Value().Bool() {
		c.disableAction = cmd.disableAction
		c.beforeAction = cmd.beforeAction
	}

	if cmd.parent != nil {
		c.parent = compileParent(cmd.parent)
	}
	return c
}

type SubCmdInfo struct {
	Name          string
	Description   string
	Category      string
	Disabled      bool
	disableAction action.Action
}

type Cmd struct {
	mu    sync.Mutex
	name  string
	cnf   *settings.Profile
	flags varflag.Flags

	isRoot           bool
	sharedCalled     bool
	parents          []string
	isWrapperCommand bool
	catdesc          map[string]string
	usage            []string
	info             []string

	disableAction      action.Action
	beforeAction       action.WithArgs
	doAction           action.WithArgs
	afterSuccessAction action.Action
	afterFailureAction action.WithPrevErr
	afterAlwaysAction  action.WithPrevErr

	parent *Cmd

	// used in help menu
	globalFlags []varflag.Flag
	sharedFlags []varflag.Flag
	ownFlags    []varflag.Flag

	subcmds []*SubCmdInfo

	err error
}

func (c *Cmd) IsRoot() bool {
	return c.isRoot
}

func (c *Cmd) Name() string {
	return c.name
}

func (c *Cmd) Usage() []string {
	return c.usage
}

func (c *Cmd) Disabled() bool {
	return c.cnf.Get("disabled").Value().Bool()
}

func (c *Cmd) CheckDisabled(sess *session.Context) bool {
	if c.disableAction != nil {
		var disabled bool
		if err := c.disableAction(sess); err != nil {
			internal.LogInit(sess.Log(), fmt.Sprintf("hide(%s): %s", c.name, err.Error()))
			disabled = true
			c.err = err
		}
		if err := c.cnf.Set("disabled", disabled); err != nil {
			sess.Log().Error(err.Error())
		}
	}

	for _, scmd := range c.subcmds {
		if scmd.disableAction != nil {

			if err := scmd.disableAction(sess); err != nil {
				internal.LogInit(sess.Log(), fmt.Sprintf("disable-cmd(%s): %s", scmd.Name, err.Error()))
				scmd.Disabled = true
			}
		}
	}
	return c.Disabled()
}

func (c *Cmd) Info() []string {
	return c.info
}

// Flag looks up flag with given name and returns flags.Interface.
// If no flag was found empty bool flag will be returned.
// Instead of returning error you can check returned flags .IsPresent.
func (c *Cmd) Flag(name string) varflag.Flag {

	f, err := c.flags.Get(name)
	if err != nil {
		f, _ = varflag.Bool(name, false, "")
	}
	return f
}

func (c *Cmd) Flags() []varflag.Flag {
	return c.ownFlags
}

func (c *Cmd) GetFlagSet() varflag.Flags {
	return c.flags
}

func (c *Cmd) SharedFlags() []varflag.Flag {
	return c.sharedFlags
}

func (c *Cmd) GlobalFlags() []varflag.Flag {
	return c.globalFlags
}

func (c *Cmd) SubCommands() []*SubCmdInfo {
	return c.subcmds
}

func (c *Cmd) Categories() map[string]string {
	return c.catdesc
}

func (c *Cmd) IsImmediate() bool {
	return c.cnf.Get("immediate").Value().Bool()
}

func (c *Cmd) IsWrapper() bool {
	return c.isWrapperCommand
}

func (c *Cmd) ExecBefore(sess *session.Context) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.parent != nil && !c.sharedCalled && !c.cnf.Get("skip_shared_before").Value().Bool() {
		if err := c.parent.callSharedBeforeAction(sess); err != nil {
			return err
		}
		// dereference parent
		c.parent = nil
	}

	if c.CheckDisabled(sess) {
		if c.cnf.Get("fail_disabled").Value().Bool() {
			if c.err != nil {
				return c.err
			}
			return fmt.Errorf("%w: %s", ErrCommandNotAllowed, c.name)
		} else {
			internal.Log(sess.Log(), fmt.Sprintf("%s: %s", ErrCommandNotAllowed, c.name))
			return nil
		}
	}
	if c.beforeAction == nil {
		return nil
	}

	args, err := c.getArgs()
	if err != nil {
		return err
	}

	if err := c.beforeAction(sess, args); err != nil {
		internal.Log(
			sess.Log(),
			"before action",
			slog.String("cmd", c.name),
			slog.String("err", err.Error()),
		)
		return err
	}
	// dereference before action
	c.beforeAction = nil
	return nil
}

func (c *Cmd) ExecDo(sess *session.Context) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.doAction == nil {
		return nil
	}

	args, err := c.getArgs()
	if err != nil {
		return err
	}

	if err := c.doAction(sess, args); err != nil {
		internal.Log(
			sess.Log(),
			"do action",
			slog.String("cmd", c.name),
			slog.String("err", err.Error()),
		)
		return err
	}

	// dereference do action
	c.doAction = nil
	return err
}

func (c *Cmd) ExecAfterFailure(sess *session.Context, err error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.afterFailureAction == nil {
		return nil
	}

	if err := c.afterFailureAction(sess, err); err != nil {
		internal.Log(
			sess.Log(),
			"after failure action",
			slog.String("cmd", c.name),
			slog.String("err", err.Error()),
		)
		return err
	}
	// dereference after failure action
	c.afterFailureAction = nil
	return nil
}

func (c *Cmd) ExecAfterSuccess(sess *session.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.afterSuccessAction == nil {
		return nil
	}

	if err := c.afterSuccessAction(sess); err != nil {
		internal.Log(sess.Log(), "after success action",
			slog.String("cmd", c.name),
			slog.String("err", err.Error()),
		)
		return err
	}

	// dereference after success action
	c.afterSuccessAction = nil
	return nil
}

func (c *Cmd) ExecAfterAlways(sess *session.Context, err error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.afterAlwaysAction == nil {
		return nil
	}

	if err := c.afterAlwaysAction(sess, err); err != nil {
		internal.Log(sess.Log(), "after always action",
			slog.String("cmd", c.name),
			slog.String("err", err.Error()),
		)
		return err
	}

	// dereference after always action
	c.afterAlwaysAction = nil
	return nil
}

func (c *Cmd) callSharedBeforeAction(sess *session.Context) error {
	if c.parent != nil {
		if err := c.parent.callSharedBeforeAction(sess); err != nil {
			return err
		}
		// dereference parent
		c.parent = nil
	}
	if c.beforeAction == nil {
		return nil
	}

	// Is before action shared with sub commands
	if c.cnf.Get("shared_before_action").Value().Bool() {
		// Check is caller parent disabled and should fail.
		// Even if caller parent is disabled but fail_disabled for this parent
		// is not set the call it.
		fmt.Printf("%q.shared_before_action\n", c.name)
		if c.cnf.Get("fail_disabled").Value().Bool() && c.CheckDisabled(sess) {
			if c.err != nil {
				return c.err
			}
			return fmt.Errorf("%w: %s", ErrCommandNotAllowed, c.name)
		}
		c.sharedCalled = true
		if err := c.beforeAction(sess, action.NewArgs(c.flags)); err != nil {
			internal.Log(sess.Log(), "shared before action",
				slog.String("cmd", c.name),
				slog.String("err", err.Error()))
			return err
		}
		// dereference before action
		c.beforeAction = nil
	}
	return nil
}

func (c *Cmd) SkipSharedBeforeAction() bool {
	return c.cnf.Get("skip_shared_before").Value().Bool()
}

func (c *Cmd) HasBefore() bool {
	return c.beforeAction != nil
}

func (c *Cmd) Err() error {
	return c.err
}

func (c *Cmd) Config() *settings.Profile {
	return c.cnf
}

func (c *Cmd) getArgs() (action.Args, error) {
	args := action.NewArgs(c.flags)
	argnmin := c.cnf.Get("min_args").Value().Uint()
	argnmax := c.cnf.Get("max_args").Value().Uint()
	name := c.name

	if argnmin == 0 && argnmax == 0 && args.Argn() > 0 {
		return args, fmt.Errorf("%w: %s does not accept arguments", Error, name)
	}

	if args.Argn() < argnmin {
		if err := c.cnf.Get("min_args_err").Value(); !err.Empty() {
			return args, errors.New(err.String())
		}
		return args, fmt.Errorf("%w: %s: requires min %d arguments, %d provided", Error, name, argnmin, args.Argn())
	}
	if args.Argn() > argnmax {
		if err := c.cnf.Get("max_args_err").Value(); !err.Empty() {
			return args, errors.New(err.String())
		}
		return args, fmt.Errorf("%w: %s: accepts max %d arguments, %d provided, extra %v", Error, name, argnmax, args.Argn(), args.Args()[argnmax:args.Argn()])
	}

	return args, nil
}
