// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package command

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/app/session"
	"github.com/happy-sdk/happy/sdk/logging"
)

// Command is building command chain from provided root command.
func Compile(root *Command) (*Cmd, *logging.QueueLogger, error) {

	if err := root.verify(); err != nil {
		return nil, root.cnflog, err
	}

	if !root.tryLock("Compile") {
		return nil, root.cnflog, fmt.Errorf("%w: failed to compile the command %s", Error, root.logName)
	}
	defer root.mu.Unlock()

	if err := root.flags.Parse(os.Args); err != nil {
		return nil, root.cnflog, err
	}

	acmd, err := root.getActiveCommand()
	if err != nil {
		return nil, root.cnflog, err
	}

	cmd := &Cmd{}

	if acmd == root {
		cmd.isRoot = true
		cmd.globalFlags = root.getGlobalFlags().Flags()
	} else {
		cmd.globalFlags = root.getGlobalFlags().Flags()
		cmd.ownFlags = acmd.flags.Flags()

		for _, flag := range cmd.globalFlags {
			if err := acmd.flags.Add(flag); err != nil {
				return nil, root.cnflog, fmt.Errorf("%w: %s: %s", Error, acmd.cnf.Get("name").String(), err.Error())
			}
		}
		sharedf, err := acmd.getSharedFlags()
		if err != nil && !errors.Is(err, ErrHasNoParent) {
			return nil, root.cnflog, fmt.Errorf("%w: %s: %s", Error, acmd.cnf.Get("name").String(), err.Error())
		}
		cmd.sharedFlags = sharedf.Flags()
		for _, flag := range cmd.sharedFlags {
			if err := acmd.flags.Add(flag); err != nil {
				return nil, root.cnflog, fmt.Errorf("%w: %s: %s", Error, acmd.cnf.Get("name").String(), err.Error())
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

	cmd.beforeAction = acmd.beforeAction
	cmd.doAction = acmd.doAction
	cmd.afterSuccessAction = acmd.afterSuccessAction
	cmd.afterFailureAction = acmd.afterFailureAction
	cmd.afterAlwaysAction = acmd.afterAlwaysAction

	var catdesc = make(map[string]string)
	if acmd.parent != nil {
		cmd.parent = compileParent(acmd.parent)
		if acmd.parent.catdesc != nil {
			for k, v := range acmd.parent.catdesc {
				catdesc[k] = v
			}
		}
	}

	for k, v := range cmd.catdesc {
		catdesc[k] = v
	}
	for _, scmd := range acmd.subCommands {
		cmd.subcmds = append(cmd.subcmds, SubCmdInfo{
			Name:        scmd.cnf.Get("name").String(),
			Description: scmd.cnf.Get("description").String(),
			Category:    scmd.cnf.Get("category").String(),
		})
		for k, v := range scmd.catdesc {
			catdesc[k] = v
		}
	}
	cmd.catdesc = catdesc

	return cmd, root.cnflog, nil
}

func compileParent(cmd *Command) *Cmd {
	c := &Cmd{
		cnf:              cmd.cnf,
		parents:          cmd.parents,
		isWrapperCommand: cmd.isWrapperCommand,
		usage:            cmd.usage,
		info:             cmd.info,
		catdesc:          cmd.catdesc,
		flags:            cmd.flags,
	}

	if c.cnf.Get("shared_before_action").Value().Bool() {
		c.beforeAction = cmd.beforeAction
	}

	if cmd.parent != nil {
		c.parent = compileParent(cmd.parent)
	}
	return c
}

type SubCmdInfo struct {
	Name        string
	Description string
	Category    string
}

type Cmd struct {
	mu    sync.Mutex
	cnf   *settings.Profile
	flags varflag.Flags

	isRoot           bool
	sharedCalled     bool
	parents          []string
	isWrapperCommand bool
	catdesc          map[string]string
	usage            []string
	info             []string

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

	subcmds []SubCmdInfo
}

func (c *Cmd) IsRoot() bool {
	return c.isRoot
}

func (c *Cmd) Name() string {
	return c.cnf.Get("name").String()
}

func (c *Cmd) Usage() []string {
	return c.usage
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

func (c *Cmd) SubCommands() []SubCmdInfo {
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

	args := action.NewArgs(c.flags)
	argnmin := c.cnf.Get("min_args").Value().Uint()
	argnmax := c.cnf.Get("max_args").Value().Uint()
	name := c.cnf.Get("name").String()

	if argnmin == 0 && argnmax == 0 && args.Argn() > 0 {
		return fmt.Errorf("%w: %s does not accept arguments", Error, name)
	}

	if args.Argn() < argnmin {
		return fmt.Errorf("%w: %s: requires min %d arguments, %d provided", Error, name, argnmin, args.Argn())
	}

	if args.Argn() > argnmax {
		return fmt.Errorf("%w: %s: accepts max %d arguments, %d provided, extra %v", Error, name, argnmax, args.Argn(), args.Args()[argnmax:args.Argn()])
	}

	if c.parent != nil && !c.sharedCalled && !c.cnf.Get("skip_shared_before").Value().Bool() {
		if err := c.parent.callSharedBeforeAction(sess); err != nil {
			return err
		}
		// dereference parent
		c.parent = nil
	}

	if c.beforeAction == nil {
		return nil
	}
	if err := c.beforeAction(sess, args); err != nil {
		return fmt.Errorf("%w: %s: %w", Error, name, err)
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

	args := action.NewArgs(c.flags)
	argnmin := c.cnf.Get("min_args").Value().Uint()
	argnmax := c.cnf.Get("max_args").Value().Uint()
	name := c.cnf.Get("name").String()

	if argnmin == 0 && argnmax == 0 && args.Argn() > 0 {
		return fmt.Errorf("%w: %s does not accept arguments", Error, name)
	}

	if args.Argn() < argnmin {
		return fmt.Errorf("%w: %s: requires min %d arguments, %d provided", Error, name, argnmin, args.Argn())
	}

	if args.Argn() > argnmax {
		return fmt.Errorf("%w: %s: accepts max %d arguments, %d provided, extra %v", Error, name, argnmax, args.Argn(), args.Args()[argnmax:args.Argn()])
	}

	if err := c.doAction(sess, args); err != nil {
		return fmt.Errorf("%w: %s: %w", Error, c.Name(), err)
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
		return fmt.Errorf("%w: %s: %w", Error, c.Name(), err)
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
		return fmt.Errorf("%w: %s: %w", Error, c.Name(), err)
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
		return fmt.Errorf("%w: %s: %w", Error, c.Name(), err)
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
	if c.cnf.Get("shared_before_action").Value().Bool() {
		c.sharedCalled = true
		if err := c.beforeAction(sess, action.NewArgs(c.flags)); err != nil {
			return fmt.Errorf("%w: %s: %w", Error, c.cnf.Get("name").String(), err)
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
