// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mkungla/happy/pkg/hlog"
	"github.com/mkungla/happy/pkg/varflag"
	"github.com/mkungla/happy/pkg/vars"
	"golang.org/x/exp/slog"
)

type Application struct {
	session *Session
	engine  *Engine

	rootCmd   *Command
	activeCmd *Command
	addons    []Addon

	// tick tock when defined are only
	// used when root command is called
	tickAction ActionTick
	tockAction ActionTock

	// pendingOpts contains options
	// which are not yet applied.
	pendingOpts []OptionAttr

	// internals
	running     bool
	initialized time.Time

	// logger
	logger *hlog.Logger
	lvl    *slog.LevelVar

	// exit handler
	exitOs   bool
	exitFunc []func(code int)
	exitCh   chan struct{}
}

// New returns new happy application instance.
// It panics if there is critical internal error or bug.
func New(opts ...OptionAttr) *Application {
	a := &Application{
		engine:      newEngine(),
		initialized: time.Now(),
		exitOs:      true,
		lvl:         &slog.LevelVar{},
	}
	err := a.configureApplication(opts)

	a.configureLogger()

	if err != nil {
		a.logger.Error("config error", err)
		a.exit(1)
	}

	if err := a.configureRootCommand(); err != nil {
		a.logger.Error("failed to create root command", err)
	}

	return a
}

func (a *Application) Main() {
	if a.running {
		a.logger.Warn("multiple calls to app.Main() prohibited")
		return
	}
	a.running = true

	// when we disable os.Exit e.g. for tests then create
	// channel which would block main thread
	if !a.exitOs {
		a.exitCh = make(chan struct{}, 1)
		defer close(a.exitCh)
	}

	// initialize application
	if err := a.initialize(); err != nil {
		a.logger.Error("initialization failed", err)
		a.exit(1)
		return
	}

	// check for config options which were not used
	if len(a.pendingOpts) > 0 {
		for _, opt := range a.pendingOpts {
			group := "option"
			if opt.kind&optionKindConfig == optionKindConfig {
				group = "config"
			} else if opt.kind&optionKindSetting == optionKindSetting {
				group = "settings"
			}
			a.logger.Warn("option not used", slog.Group(group,
				slog.String("key", opt.key),
				slog.Any("value", opt.value),
				slog.Bool("readOnly", opt.kind&optionKindReadOnly == optionKindReadOnly),
			))
		}
	}
	// Start application main process
	go a.execute()

	// handle os specific main thread blocking
	osmain(a.exitCh)
}

func (a *Application) Before(action ActionWithArgs) {
	a.rootCmd.Before(action)
}

func (a *Application) Do(action ActionWithArgs) {
	a.rootCmd.Do(action)
}

func (a *Application) AfterSuccess(action Action) {
	a.rootCmd.AfterSuccess(action)
}

func (a *Application) AfterFailure(action func(s *Session, err error) error) {
	a.rootCmd.AfterFailure(action)
}

func (a *Application) AfterAlways(action Action) {
	a.rootCmd.AfterAlways(action)
}

func (a *Application) OnTick(action ActionTick) {
	a.tickAction = action
}

func (a *Application) OnTock(action ActionTock) {
	a.tockAction = action
}

func (a *Application) Cron(setup func(schedule CronScheduler)) {
	a.logger.NotImplemented("use service for cron")
}

func (a *Application) WithAddons(addon ...Addon) {
	if addon != nil {
		a.addons = append(a.addons, addon...)
	}
}

func (a *Application) RegisterService(svc *Service) error {
	return a.engine.serviceRegister(a.session, svc)
}

func (a *Application) AddCommand(cmd *Command) {
	a.rootCmd.AddSubCommand(cmd)
}

func (a *Application) AddFlag(f varflag.Flag) {
	a.rootCmd.AddFlag(f)
}

func (a *Application) shutdown() {
	if err := a.engine.stop(a.session); err != nil {
		a.logger.Error("failed to stop engine", err)
	}
	// Destroy session
	a.session.Destroy(nil)
	if err := a.session.Err(); err != nil && !errors.Is(err, ErrSessionDestroyed) {
		a.logger.Warn("session", slog.String("err", err.Error()))
	}

}

func (a *Application) exit(code int) {
	a.logger.SystemDebug("shutting down", slog.Int("exit.code", code))

	for _, fn := range a.exitFunc {
		fn(code)
	}

	a.shutdown()

	a.logger.SystemDebug("shutdown complete", slog.Duration("uptime", a.engine.uptime()))

	if a.exitCh != nil {
		a.exitCh <- struct{}{}
	}
	if a.exitOs {
		os.Exit(code)
	}
}

func (a *Application) initialize() error {
	defer func() {
		dur := time.Since(a.initialized)
		a.logger.SystemDebug(
			"initialization",
			a.session.Get("app.version"),
			slog.Duration("took", dur),
		)
	}()

	if err := a.registerAddonCommands(); err != nil {
		return err
	}

	// Verify command chain
	// register commands

	// Fail fast if command or one of the sub commands has errors
	if err := a.rootCmd.verify(); err != nil {
		return err
	}

	if e := a.rootCmd.flags.Parse(os.Args); e != nil {
		return errors.Join(ErrApplication, e)
	}

	// print application version and exit
	if a.rootCmd.flag("version").Present() {
		a.lvl.Set(100)
		a.printVersion()
		a.exit(0)
	}

	// set log verbosity from flags
	if a.rootCmd.flag("system-debug").Present() {
		a.lvl.Set(slog.Level(hlog.LevelSystemDebug))
	} else if a.rootCmd.flag("debug").Present() {
		a.lvl.Set(slog.Level(hlog.LevelDebug))
	} else if a.rootCmd.flag("verbose").Present() {
		a.lvl.Set(slog.Level(hlog.LevelInfo))
	}

	if err := a.setActiveCommand(); err != nil {
		return err
	}

	// show help
	if a.rootCmd.flag("help").Present() {
		if err := a.clihelp(); err != nil {
			a.logger.Error("failed to create help view", err)
		}
		a.shutdown()
		os.Exit(0)
		return nil
	}

	a.logger.Debug(
		"enable logging",
		slog.String("level", hlog.Level(a.lvl.Level()).String()),
		slog.String("cmd", a.activeCmd.name),
	)

	if err := a.registerInternalEvents(); err != nil {
		return err
	}

	if err := a.registerAddons(); err != nil {
		return err
	}

	return nil
}

func (a *Application) setActiveCommand() error {
	settree := a.rootCmd.flags.GetActiveSets()
	name := settree[len(settree)-1].Name()
	if name == "/" {
		a.activeCmd = a.rootCmd
		// only set app tick tock if current command is root command
		a.engine.onTick(a.tickAction)
		a.engine.onTock(a.tockAction)
		return nil
	}

	var (
		activeCmd *Command
		exists    bool
	)

	// skip root cmd
	for _, set := range settree[1:] {
		name := set.Name()
		if activeCmd == nil {
			activeCmd, exists = a.rootCmd.getSubCommand(name)
			if !exists {
				return fmt.Errorf("%w: unknown command: %s", ErrApplication, name)
			}
			continue
		}
		activeCmd, exists = activeCmd.getSubCommand(set.Name())
		if !exists {
			return fmt.Errorf("%w: unknown subcommand: %s for %s", ErrApplication, name, activeCmd.name)
		}
		break
	}

	a.activeCmd = activeCmd

	return nil
}

func (a *Application) execute() {
	if err := a.session.start(); err != nil {
		a.logger.Error("failed to start session", err)
		a.exit(1)
		return
	}

	if err := a.engine.start(a.session); err != nil {
		a.logger.Error("failed to start the engine", err)
		a.exit(1)
		return
	}

	// execute before action chain
	if err := a.executeBeforeActions(); err != nil {
		a.logger.Error("prerequisites failed", err)
		a.exit(1)
		return
	}

	// block until session is ready
	a.logger.SystemDebug("waiting session...")
	<-a.session.Ready()

	if a.session.Err() != nil {
		a.exit(1)
		return
	}

	cmdtree := strings.Join(a.activeCmd.parents, ".") + "." + a.activeCmd.name
	a.logger.SystemDebug("session ready: execute", slog.String("action", "Do"), slog.String("command", cmdtree))

	err := a.activeCmd.callDoAction(a.session)
	if err != nil {
		a.executeAfterFailureActions(err)
	} else {
		a.executeAfterSuccessActions()
	}
	a.executeAfterAlwaysActions(err)
}

func (a *Application) printVersion() {
	fmt.Println(a.session.Get("app.version").String())
}

func (a *Application) configureApplication(opts []OptionAttr) (err error) {
	a.session = &Session{}
	a.session.opts, err = NewOptions("config", getDefaultApplicationConfig())
	if err != nil {
		return err
	}

	var errs []error

	for key, cnf := range a.session.opts.config {
		var provided bool
		for _, opt := range opts {
			if opt.key == key {
				val, err := vars.NewValue(opt.value)
				if err != nil {
					errs = append(errs, errors.Join(fmt.Errorf("%w: config.%s validation failed", ErrOptions, opt.key), err))
					continue
				}
				// validates with validator set by
				// getDefaultConfigOpts
				if err := a.session.Set(key, val); err != nil {
					errs = append(errs, err)
					continue
				}
				provided = true
				break
			}
		}
		if !provided {
			if err := a.session.Set(key, cnf.value); err != nil {
				errs = append(errs, err)
				continue
			}
		}
	}

	// populate pending queue
	for _, opt := range opts {
		if !a.session.Has(opt.key) {
			a.pendingOpts = append(a.pendingOpts, opt)
		}
	}

	return errors.Join(errs...)
}

func (a *Application) configureLogger() {
	a.lvl.Set(slog.Level(a.session.Get("log.level").Int()))
	secretsCnf := a.session.Get("log.secrets").String()
	var secrets []string
	if len(secretsCnf) > 0 {
		keys := strings.Split(secretsCnf, ",")
		for _, secret := range keys {
			secrets = append(secrets, strings.TrimSpace(secret))
		}
	}
	handler := hlog.Config{
		Options: slog.HandlerOptions{
			AddSource: a.session.Get("log.source").Bool(),
			Level:     a.lvl,
			// ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 	return a
			// },
		},
		Colors:  a.session.Get("log.colors").Bool(),
		Secrets: secrets,
		JSON:    false,
	}.NewHandler(os.Stdout)

	a.logger = hlog.New(handler)
	a.session.logger = a.logger
	hlog.SetDefault(a.logger, a.session.Get("log.stdlog").Bool())
}

func (a *Application) configureRootCommand() error {
	rootCmd := NewCommand(
		filepath.Base(os.Args[0]),
		Option("description", a.session.Get("app.description")),
	)
	if err := rootCmd.Err(); err != nil {
		return err
	}

	var boolflags = []struct {
		Name    string
		Value   bool
		Usage   string
		Aliases []string
	}{
		{"version", false, "print application version", nil},
		{"x", false, "the -x flag prints all the external commands as they are executed.", nil},
		{"system-debug", false, "enable system debug log level (very verbose)", nil},
		{"debug", false, "enable debug log level. when debug flag is after the command then debug level will be enabled only for that command", nil},
		{"verbose", false, "enable verbose log level", []string{"v"}},
		{"help", false, "display help or help for the command. [...command --help]", []string{"h"}},
	}
	for _, flag := range boolflags {
		f, err := varflag.Bool(flag.Name, flag.Value, flag.Usage, flag.Aliases...)
		if err != nil {
			return err
		}
		rootCmd.AddFlag(f)
	}

	a.rootCmd = rootCmd
	return nil
}

func (a *Application) executeBeforeActions() error {
	a.logger.SystemDebug("execute before actions")
	if a.rootCmd != a.activeCmd {
		if err := a.rootCmd.callBeforeAction(a.session); err != nil {
			return err
		}
	}
	if err := a.activeCmd.callBeforeAction(a.session); err != nil {
		return err
	}
	return nil
}

func (a *Application) executeAfterFailureActions(err error) {
	a.logger.Error("execute after failure actions", err)

	if err := a.activeCmd.callAfterFailureAction(a.session, err); err != nil {
		a.logger.Error("command after failure action", err)
	}

	if a.rootCmd != a.activeCmd {
		if err := a.rootCmd.callAfterFailureAction(a.session, err); err != nil {
			a.logger.Error("app after failure action", err)
		}
	}
}

func (a *Application) executeAfterSuccessActions() {
	a.logger.SystemDebug("execute after success actions")
	if err := a.activeCmd.callAfterSuccessAction(a.session); err != nil {
		a.logger.Error("command after success action", err)
	}

	if a.rootCmd != a.activeCmd {
		if err := a.rootCmd.callAfterSuccessAction(a.session); err != nil {
			a.logger.Error("app after success action", err)
		}
	}
}

func (a *Application) executeAfterAlwaysActions(err error) {
	a.logger.SystemDebug("execute after always actions")

	if err := a.activeCmd.callAfterAlwaysAction(a.session); err != nil {
		a.logger.Error("command after always action", err)
	}

	if a.rootCmd != a.activeCmd {
		if err := a.rootCmd.callAfterAlwaysAction(a.session); err != nil {
			a.logger.Error("app after always action", err)
		}
	}

	if err != nil {
		a.exit(1)
	} else {
		a.exit(0)
	}
}

func (a *Application) registerAddonCommands() error {
	var provided bool
	for _, addon := range a.addons {
		for _, cmd := range addon.Commands() {
			if err := cmd.Err(); err != nil {
				return err
			}
			a.AddCommand(cmd)
			provided = true
		}
	}

	if provided {
		a.logger.SystemDebug("attached commands provided by addons")
	}

	return nil
}

func (a *Application) registerAddons() error {
	var provided bool

	for _, addon := range a.addons {
		info, err := addon.Register(a.session)
		if err != nil {
			return err
		}
		provided = true
		a.logger.Debug(
			"registered addon",
			slog.Group("addon",
				slog.String("name", info.Name),
				slog.String("version", info.Version),
			),
		)
		for _, svc := range addon.Services() {
			if err := a.RegisterService(svc); err != nil {
				return err
			}
		}

		for _, ev := range addon.Events() {
			if err := a.engine.registerEvent(ev); err != nil {
				return err
			}
		}

		if api := addon.API(); api != nil {
			if err := a.session.registerAPI(info.Name, api); err != nil {
				return err
			}
		}
	}
	if provided {
		a.logger.SystemDebug("registeration of addons completed")
	}

	return nil
}

func (a *Application) registerInternalEvents() error {
	var sysevs = []Event{
		RegisterEvent("services", "start.services", "starts local or connects remote service defined in payload", nil),
		RegisterEvent("services", "stop.services", "stops local or disconnects remote service defined in payload", nil),
		RegisterEvent("services", "service.started", "triggered when service has been started", nil),
		RegisterEvent("services", "service.stopped", "triggered when service has been stopped", nil),
	}

	for _, rev := range sysevs {
		if err := a.engine.registerEvent(rev); err != nil {
			return err
		}
	}
	a.logger.SystemDebug("registered system events")
	return nil
}
