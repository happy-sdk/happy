// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package happy

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/instance"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/migration"
)

type Main struct {
	mu     sync.RWMutex
	sealed bool
	slug   string
	init   *initializer
	sess   *Session

	beforeAlways ActionWithFlags
	root         *Command
	cmd          *Command
	engine       *engine
	instance     *instance.Instance

	exitTrap bool
	exitFunc []func(sess *Session, code int) error
	exitCh   chan struct{}
}

// New is alias to prototype.New
func New(s Settings) *Main {
	m := &Main{
		init: newInitializer(&s),
		root: NewCommand(filepath.Base(os.Args[0])),
	}
	m.init.Log(logging.NewQueueRecord(logging.LevelSystemDebug, "creating new application", 3))
	return m
}

const withAddonMsg = "with addon"

// WithAddons adds addons to the application.
func (m *Main) WithAddon(addon *Addon) *Main {
	if init, ok := m.canConfigure(withAddonMsg); ok {
		init.AddAddon(addon)
	}
	return m
}

const withMigrationsMsg = "with migrations"

// WithMigrations adds migrations manager to the application.
func (m *Main) WithMigrations(mm *migration.Manager) *Main {
	if init, ok := m.canConfigure(withMigrationsMsg); ok {
		init.AddMigrations(mm)
	}
	return m
}

const withServiceMsg = "with service"

// WithService adds service to the application.
func (m *Main) WithService(svc *Service) *Main {
	if init, ok := m.canConfigure(withServiceMsg); ok {
		init.AddService(svc)
	}
	return m
}

// WithCommand adds command to the application.
func (m *Main) WithCommand(cmd *Command) *Main {
	const withCommandMsg = "with command"
	if init, ok := m.canConfigure(withCommandMsg); ok {
		if cmd == nil {
			init.Log(logging.NewQueueRecord(logging.LevelBUG, withCommandMsg, 3, slog.Any("command", nil)))
			return m
		}
		m.mu.RLock()
		m.root.AddSubCommand(cmd)
		m.mu.RUnlock()
		init.Log(logging.NewQueueRecord(logging.LevelSystemDebug, withCommandMsg, 3, slog.String("name", cmd.name)))
	}
	return m
}

// WithFlag adds flag to the application.
func (m *Main) WithFlag(flag varflag.Flag) *Main {
	const withFlagMsg = "with flag"
	if init, ok := m.canConfigure(withFlagMsg); ok {
		if flag == nil {
			init.Log(logging.NewQueueRecord(logging.LevelBUG, withFlagMsg, 3, slog.Any("flag", nil)))
			return m
		}
		m.mu.RLock()
		m.root.AddFlag(flag)
		m.mu.RUnlock()
		init.Log(logging.NewQueueRecord(logging.LevelSystemDebug, withFlagMsg, 3, slog.String("name", flag.Name())))
	}
	return m
}

// WithLogger sets application logger.
func (m *Main) WithLogger(l logging.Logger) *Main {
	if init, ok := m.canConfigure("adding logger"); ok {
		init.SetLogger(l)
	}
	return m
}

func (m *Main) WithOptions(opts ...OptionArg) *Main {
	if init, ok := m.canConfigure("with options"); ok {
		init.AddOptions(opts...)
	}
	return m
}

// BeforeAlways is executed before any command.
func (m *Main) BeforeAlways(a ActionWithFlags) *Main {
	if _, ok := m.canConfigure("adding BeforeAlways action"); ok {
		m.mu.Lock()
		m.beforeAlways = a
		m.mu.Unlock()
	}
	return m
}

// Before is executed only when no command is specified.
func (m *Main) Before(a ActionWithArgs) *Main {
	if _, ok := m.canConfigure("adding Before action"); ok {
		m.root.Before(a)
	}
	return m
}

// Do is executed only when no command is specified.
func (m *Main) Do(a ActionWithArgs) *Main {
	if _, ok := m.canConfigure("adding AfterSuccess action"); ok {
		m.root.Do(a)
	}
	return m
}

// AfterSuccess is executed after every successful application execution.
func (m *Main) AfterSuccess(a Action) *Main {
	if _, ok := m.canConfigure("adding AfterSuccess action"); ok {
		m.root.AfterSuccess(a)
	}
	return m
}

// AfterFailure is executed after every failed application execution.
func (m *Main) AfterFailure(a ActionWithPrevErr) *Main {
	if _, ok := m.canConfigure("adding AfterFailure action"); ok {
		m.root.AfterFailure(a)
	}
	return m
}

// AfterAlways is executed after every application execution.
func (m *Main) AfterAlways(a ActionWithPrevErr) *Main {
	if _, ok := m.canConfigure("adding AfterAlways action"); ok {
		m.root.AfterAlways(a)
	}
	return m
}

// Tick when set is executed on every tick defined by Settings.ThrottleTicks interval.
func (m *Main) Tick(a ActionTick) *Main {
	if i, ok := m.canConfigure("adding tick action"); ok {
		i.SetTick(a)
	}
	return m
}

// Tock when set is executed right after Tick.
// If Tick returns error then Tock is not executed.
func (m *Main) Tock(a ActionTock) *Main {
	if i, ok := m.canConfigure("adding tock action"); ok {
		i.SetTock(a)
	}
	return m
}

// Run starts the Application.
func (m *Main) Run() {
	m.mu.RLock()
	sealed := m.sealed
	m.mu.RUnlock()
	if sealed {
		m.log(logging.LevelError, "can not call .Run application is already sealed")
		return
	}

	// initialize (mutex is locked inside)
	if err := m.init.Initialize(m); err != nil {
		m.sess.Log().LogDepth(2, logging.LevelError, "initialization error", slog.String("err", err.Error()))
		if errors.Is(err, errExitSuccess) {
			m.exit(0)
			return
		}
		m.exit(1)
		return
	}

	// when we disable os.Exit e.g. for tests
	// and use channel which would block main thread.
	if m.exitTrap {
		m.exitCh = make(chan struct{}, 1)
		defer close(m.exitCh)
	}

	if m.cmd == nil {
		if err := m.help(); err != nil {
			m.sess.Log().Error("help error", slog.String("err", err.Error()))
			m.exit(1)
			return
		}
	}

	// Start application main process
	go m.run()

	// handle os specific main thread
	osmain(m.exitCh)
}

func (m *Main) printVersion() {
	fmt.Println(m.sess.Get("app.version").String())
}

func (m *Main) run() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.sess.start(); err != nil {
		m.sess.Log().Error("failed to start session", slog.String("err", err.Error()))
		m.exit(1)
		return
	}

	if m.sess.Get("app.devel").Bool() {
		m.sess.Log().Info("development mode",
			slog.String("profile", m.sess.Get("app.profile.name").String()),
		)
	}

	if err := m.engine.start(m.sess); err != nil {
		m.sess.Log().Error("failed to start the engine", slog.String("err", err.Error()))
		m.exit(1)
		return
	}

	if err := m.executeBeforeActions(); err != nil {
		m.sess.Log().Error("failed to call before always", slog.String("err", err.Error()))
		m.exit(1)
		return
	}

	m.sess.setReady()
	if m.sess.Err() != nil {
		m.exit(1)
		return
	}

	cmdtree := strings.Join(m.cmd.parents, ".") + "." + m.cmd.name
	m.sess.Log().SystemDebug("execute", slog.String("action", "Do"), slog.String("command", cmdtree))

	err := m.cmd.callDoAction(m.sess)

	if svcerr := m.engine.stop(m.sess); svcerr != nil {
		fmt.Println("happy-main.go: 513")
		m.sess.Log().Error("failed to stop engine", slog.String("err", svcerr.Error()))
	}

	if !m.sess.canRecover(err) {
		if err := m.cmd.callAfterFailureAction(m.sess, err); err != nil {
			m.sess.Log().Error("failed to call after failure", slog.String("err", err.Error()))
		}
	} else {
		if err := m.cmd.callAfterSuccessAction(m.sess); err != nil {
			m.sess.Log().Error("failed to call after success", slog.String("err", err.Error()))
		}
	}

	if m.sess.canRecover(err) {
		err = nil
	}
	if err := m.cmd.callAfterAlwaysAction(m.sess, err); err != nil {
		m.sess.Log().Error("failed to call cmd after always", slog.String("err", err.Error()))
	}

	if err != nil {
		m.exit(1)
	} else {
		m.exit(0)
	}
}

func (m *Main) executeBeforeActions() error {
	if m.beforeAlways != nil {
		m.log(logging.LevelSystemDebug, "executing before always")
		if err := m.beforeAlways(m.sess, m.root.getFlags()); err != nil {
			return err
		}
	}
	return m.cmd.callBeforeAction(m.sess)
}

func (m *Main) help() error {
	m.sess.Log().Println("--")
	return nil
}

func (m *Main) log(lvl logging.Level, msg string, attrs ...slog.Attr) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.sess != nil {
		m.sess.Log().LogDepth(4, lvl, msg, attrs...)
		return
	}

	m.init.log(logging.NewQueueRecord(lvl, msg, 4, attrs...))
}

// canConfigure checks if application can be configured.
func (m *Main) canConfigure(errmsg string) (*initializer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.sealed {
		m.log(logging.LevelError, errmsg, slog.String("err", "application is aleady sealed"))
		return nil, false
	}

	if m.init == nil {
		m.log(logging.LevelError, errmsg, slog.String("err", "initializer is nil"))
		return nil, false
	}
	if m.root == nil {
		m.log(logging.LevelError, errmsg, slog.String("err", "root command is nil"))
		return nil, false
	}
	return m.init, true
}

// exit handles graceful shutdown on appliaction exit.
func (m *Main) exit(code int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.sess.Log().SystemDebug("shutting down", slog.Int("exit.code", code))

	for _, fn := range m.exitFunc {
		if err := fn(m.sess, code); err != nil {
			m.sess.Log().Error("exit func", slog.String("err", err.Error()))
		}
	}

	var uptime time.Duration
	if m.engine != nil {
		if err := m.engine.stop(m.sess); err != nil {
			m.sess.Log().Error("failed to stop engine", slog.String("err", err.Error()))
		}
	}

	m.sess.Destroy(nil)
	if err := m.sess.Err(); err != nil && !errors.Is(err, ErrSessionDestroyed) {
		m.sess.Log().Warn("session", slog.String("err", err.Error()))
	}

	if m.engine != nil {
		uptime = m.engine.uptime()
	}
	if m.exitCh != nil {
		m.exitCh <- struct{}{}
	}
	if err := m.save(); err != nil {
		m.sess.Log().Error("failed to save state", slog.String("err", err.Error()))
	}

	m.sess.Log().SystemDebug("shutdown complete", slog.String("uptime", uptime.String()))
	if !m.exitTrap {
		os.Exit(code)
	}
}

func (m *Main) save() error {
	profileFilePath := m.sess.Get("app.profile.file").String()
	m.sess.Log().SystemDebug("app.save",
		slog.String("profile", m.sess.Get("app.profile.name").String()),
		slog.String("file", profileFilePath),
	)
	if m.sess.Profile() == nil || !m.cmd.allowOnFreshInstall {
		m.sess.Log().SystemDebug("skip saving")
		return nil
	}

	profile := m.sess.Profile().All()

	pd := vars.Map{}
	for _, setting := range profile {
		if setting.Persistent() {
			pd.Store(setting.Key(), setting.Value().String())
		}
	}
	pddata := pd.ToKeyValSlice()

	cnfDir := m.sess.Get("app.fs.path.config").String()
	cstat, err := os.Stat(cnfDir)
	if err != nil {
		return err
	}
	if !cstat.IsDir() {
		return fmt.Errorf("%w: invalid config directory %s", Error, profileFilePath)
	}

	var dest bytes.Buffer
	enc := gob.NewEncoder(&dest)
	if err := enc.Encode(pddata); err != nil {
		return err
	}

	if err := os.WriteFile(profileFilePath, dest.Bytes(), 0600); err != nil {
		return err
	}

	m.sess.Log().SystemDebug(
		"saved profile",
		slog.String("profile", m.sess.Get("app.profile.name").String()),
		slog.String("file", profileFilePath),
	)

	return nil
}