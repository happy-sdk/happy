// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mkungla/happy/pkg/hlog"
	"github.com/mkungla/happy/pkg/varflag"
	"github.com/mkungla/happy/pkg/vars"
	"github.com/mkungla/happy/pkg/version"
	"golang.org/x/exp/slog"
	"golang.org/x/mod/semver"
)

type Application struct {
	session *Session
	engine  *Engine

	rootCmd   *Command
	activeCmd *Command
	addons    []*Addon

	// tick tock when defined are only
	// used when root command is called
	tickAction ActionTick
	tockAction ActionTock

	firstUseAction Action

	// pendingOpts contains options
	// which are not yet applied.
	pendingOpts []OptionArg

	// internals
	running     bool
	initialized time.Time

	// logger
	logger *hlog.Logger
	lvl    *slog.LevelVar

	// exit handler
	exitOs   bool
	exitFunc []func(code int) error
	exitCh   chan struct{}
	errs     []error
	firstuse bool
	state    *persistentState
	isDev    bool
	profile  string
}

// New returns new happy application instance.
// It panics if there is critical internal error or bug.
func New(opts ...OptionArg) *Application {
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

	if len(a.pendingOpts) > 0 {
		for _, opt := range a.pendingOpts {
			// apply if it is custom glopal setting
			a.session.Log().SystemDebug("opt", slog.Any(opt.key, opt.value))
			if _, ok := a.session.opts.config[opt.key]; ok {
				if err := opt.apply(a.session.opts); err != nil {
					a.session.Log().Error("failed to apply option", err)
					a.exit(1)
				}
				continue
			}

			group := "option"
			if opt.kind&ConfigOption != 0 {
				group = "config"
			} else if opt.kind&SettingsOption != 0 {
				group = "settings"
			}
			a.logger.Warn("option not used", slog.Group(group,
				slog.String("key", opt.key),
				slog.Any("value", opt.value),
				slog.Bool("readOnly", opt.kind&ReadOnlyOption == ReadOnlyOption),
			))
		}
	}
	if a.activeCmd.doAction == nil {
		a.cliCmdHelp(a.activeCmd)
		return
	}
	// Start application main process
	go a.execute()

	// handle os specific main thread blocking
	osmain(a.exitCh)
}

func (a *Application) Before(action ActionWithArgs) {
	if a.rootCmd != nil {
		a.rootCmd.Before(action)
	}
}

func (a *Application) Do(action ActionWithArgs) {
	if a.rootCmd != nil {
		a.rootCmd.Do(action)
	}
}

func (a *Application) AfterSuccess(action Action) {
	if a.rootCmd != nil {
		a.rootCmd.AfterSuccess(action)
	}
}

func (a *Application) AfterFailure(action func(s *Session, err error) error) {
	if a.rootCmd != nil {
		a.rootCmd.AfterFailure(action)
	}
}

func (a *Application) AfterAlways(action Action) {
	if a.rootCmd != nil {
		a.rootCmd.AfterAlways(action)
	}
}

func (a *Application) OnTick(action ActionTick) {
	a.tickAction = action
}

func (a *Application) OnTock(action ActionTock) {
	a.tockAction = action
}

func (a *Application) OnFirstUse(action Action) {
	a.firstUseAction = action
}

func (a *Application) Cron(setup func(schedule CronScheduler)) {
	a.logger.NotImplemented("use service for cron")
}

func (a *Application) WithAddons(addon ...*Addon) {
	if addon != nil {
		a.addons = append(a.addons, addon...)
	}
}

func (a *Application) RegisterService(svc *Service) {
	if svc == nil {
		a.errs = append(a.errs, fmt.Errorf("%w: atemt to register <nil> service", ErrService))
	}
	if a.engine == nil {
		a.errs = append(a.errs, fmt.Errorf("%w: engine was not ready when registering service %s", ErrEngine, svc.name))
	}
	if err := a.engine.serviceRegister(a.session, svc); err != nil {
		a.errs = append(a.errs, err)
	}
}

func (a *Application) AddCommand(cmd *Command) {
	if a.rootCmd != nil {
		a.rootCmd.AddSubCommand(cmd)
	}
}

func (a *Application) AddFlag(f varflag.Flag) {
	if a.rootCmd != nil {
		a.rootCmd.AddFlag(f)
	}
}

func (a *Application) Setting(key string, value any, description string, validator OptionValueValidator) {
	if strings.HasPrefix(key, "app.") {
		a.errs = append(a.errs, fmt.Errorf("%w: custom option %q can not start with app.", ErrOption, key))
		return
	}
	if strings.HasPrefix(key, "log.") {
		a.errs = append(a.errs, fmt.Errorf("%w: custom option %q can not start with log.", ErrOption, key))
		return
	}
	opt, ok := a.session.opts.config[key]
	if ok {
		a.errs = append(a.errs, fmt.Errorf("%w: option %q already in use (%s)", ErrOption, key, opt.desc))
		return
	}
	a.session.opts.config[key] = OptionArg{
		key:       key,
		value:     value,
		desc:      description,
		kind:      SettingsOption,
		validator: validator,
	}
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
		if err := fn(code); err != nil {
			a.logger.Error("exit func", err)
		}
	}

	a.shutdown()

	a.logger.SystemDebug("shutdown complete", slog.Duration("uptime", a.engine.uptime()))

	if a.exitCh != nil {
		a.exitCh <- struct{}{}
	}

	if err := a.save(); err != nil {
		a.logger.Error("failed to save state", err)
	}
	if a.exitOs {
		os.Exit(code)
	}
}

func (a *Application) initializePaths() error {
	if !a.session.Get("app.fs.enabled").Bool() {
		return nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	// exceptions which by pass validation
	if err := a.session.opts.db.Store("app.path.wd", wd); err != nil {
		return err
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := a.session.opts.db.Store("app.path.home", userHomeDir); err != nil {
		return err
	}
	slug := a.session.Get("app.slug").String()

	if slug == "" {
		return fmt.Errorf("%w: invalid slug %s", ErrApplication, slug)
	}
	dir := slug
	if a.profile != "default" {
		dir = filepath.Join(dir, a.profile)
	}

	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d", dir, time.Now().UnixMilli()))
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return err
	}
	if err := a.session.opts.db.Store("app.path.tmp", tempDir); err != nil {
		return err
	}
	a.exitFunc = append(a.exitFunc, func(code int) error {
		tmp := os.TempDir()
		if !strings.HasPrefix(tempDir, tmp) {
			return fmt.Errorf("%w: invalid tmp dir %s", ErrApplication, tempDir)
		}
		return os.RemoveAll(tempDir)
	})

	// app cache dir
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	appCacheDir := filepath.Join(userCacheDir, dir)
	_, err = os.Stat(appCacheDir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(appCacheDir, 0700); err != nil {
			return err
		}
		a.firstuse = true
	}
	if err := a.session.opts.db.Store("app.path.cache", appCacheDir); err != nil {
		return err
	}

	// app config dir
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	appConfigDir := filepath.Join(userConfigDir, dir)
	_, err = os.Stat(appConfigDir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(appConfigDir, 0700); err != nil {
			return err
		}
		a.firstuse = true
	}

	if err := a.session.opts.db.Store("app.path.config", appConfigDir); err != nil {
		return err
	}
	return nil
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
	if a.rootCmd == nil {
		return fmt.Errorf("%w: root command was not initialized on time", ErrHappy)
	}
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
	if a.rootCmd.flag("system-debug").Var().Bool() {
		a.lvl.Set(slog.Level(hlog.LevelSystemDebug))
	} else if a.rootCmd.flag("debug").Var().Bool() {
		a.lvl.Set(slog.Level(hlog.LevelDebug))
	} else if a.rootCmd.flag("verbose").Var().Bool() {
		a.lvl.Set(slog.Level(hlog.LevelInfo))
	}

	a.profile = a.rootCmd.flag("profile").Var().String()
	a.logger.SystemDebug("using profile", slog.String("profile", a.profile))
	if err := a.initializePaths(); err != nil {
		return err
	} else {
		if err := a.load(); err != nil {
			return err
		}
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

	// loaded persistent state
	if a.state != nil {
		var direction string
		if a.state.migration == -1 {
			direction = "up"
		} else if a.state.migration == 1 {
			direction = "down"
		}
		if a.state.migration != 0 {
			a.logger.SystemDebug("migrate",
				slog.String("direction", direction),
				slog.String("from", a.state.Version.String()),
				slog.String("to", a.session.Get("app.version").String()),
			)
		}
		a.logger.SystemDebug("loaded settings from",
			slog.String("file", a.state.cfile),
		)
	}

	if err := a.registerInternalEvents(); err != nil {
		return err
	}

	// check for config options which were not used
	a.logger.SystemDebug("initialize", slog.Bool("first.use", a.firstuse))
	if a.firstuse {
		if err := a.firstUse(); err != nil {
			return err
		}
	}
	a.session.opts.setDefaults()

	if err := a.registerAddons(); err != nil {
		return err
	}

	if err := a.save(); err != nil {
		return err
	}
	return errors.Join(a.errs...)
}

func (a *Application) firstUse() error {
	if !a.activeCmd.allowOnFirstUse {
		return fmt.Errorf("%w: command %q is not allowed on first time application use", ErrCommand, a.activeCmd.name)
	}
	a.logger.NotImplemented("first use")
	if err := a.session.opts.db.Store("app.first.use", true); err != nil {
		return err
	}

	if a.firstUseAction != nil {
		if err := a.firstUseAction(a.session); err != nil {
			return err
		}
	}
	return nil
}

type persistentState struct {
	Date      time.Time         `json:"date"`
	Version   version.Version   `json:"version"`
	Settings  []persistentValue `json:"settings"`
	migration int
	cfile     string
}

type persistentValue struct {
	Key   string `json:"key"`
	Kind  uint8  `json:"kind"`
	Value any    `json:"value"`
}

func (a *Application) load() error {
	if !a.session.Get("app.fs.enabled").Bool() {
		return nil
	}
	cpath := a.session.Get("app.path.config").String()
	if cpath == "" {
		return fmt.Errorf("%w: config path empty", ErrApplication)
	}
	cfile := filepath.Join(cpath, "state.happy")
	_, err := os.Stat(cfile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			a.firstuse = true
			return nil
		}
		return err
	}
	data, err := os.ReadFile(cfile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &a.state); err != nil {
		return err
	}
	a.state.cfile = cfile

	currver := a.session.Get("app.version").String()
	a.state.migration = semver.Compare(a.state.Version.String(), currver)

	for _, setting := range a.state.Settings {
		// override predef options
		varval, err := vars.NewValueAs(setting.Value, vars.Kind(setting.Kind))
		if err != nil {
			return err
		}
		if a.session.Has(setting.Key) {
			if err := a.session.opts.set(setting.Key, varval.Any(), true); err != nil {
				return err
			}
			continue
		}
		// override predef pending opts
		found := false
		for i, opt := range a.pendingOpts {
			if opt.key == setting.Key {
				a.pendingOpts[i].value = varval.Any()
				found = true
				break
			}
		}
		if !found {
			a.pendingOpts = append(a.pendingOpts, Option(setting.Key, varval.Any()))
		}
	}
	return nil
}

func (a *Application) save() error {
	if !a.session.Get("app.fs.enabled").Bool() {
		return nil
	}
	if a.activeCmd == nil {
		return nil
	}
	if !a.activeCmd.allowOnFirstUse {
		a.logger.SystemDebug("skip saving")
		return nil
	}
	cpath := a.session.Get("app.path.config").String()
	if cpath == "" {
		return fmt.Errorf("%w: config path empty", ErrApplication)
	}
	cstat, err := os.Stat(cpath)
	if err != nil {
		return err
	}
	if !cstat.IsDir() {
		return fmt.Errorf("%w: invalid config directory %s", ErrApplication, cpath)
	}

	cfile := filepath.Join(cpath, "state.happy")

	verstr := a.session.Get("app.version").String()
	ver, err := version.Parse(verstr)
	if err != nil {
		return err
	}
	ps := &persistentState{
		Date:    time.Now().UTC(),
		Version: ver,
	}
	settings := a.session.Settings()
	settings.Range(func(v vars.Variable) bool {
		ps.Settings = append(ps.Settings, persistentValue{
			Key:   v.Name(),
			Kind:  uint8(v.Kind()),
			Value: v.Any(),
		})
		return true
	})
	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cfile, data, 0600)
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

	if a.isDev {
		a.logger.Notice("development mode",
			slog.Bool("enabled", true),
			slog.String("profile", a.profile),
		)
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

func (a *Application) configureApplication(opts []OptionArg) (err error) {
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
					errs = append(errs, errors.Join(fmt.Errorf("%w: config.%s validation failed", ErrOption, opt.key), err))
					continue
				}
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

	a.isDev = version.IsDev(a.session.Get("app.version").String())

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
		Option("usage", a.session.Get("app.usage")),
		Option("category", ""),
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
		{"no-color", false, "disable colored output", nil},
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

	profile := "default"
	if a.isDev {
		profile = "devel"
	}

	profileFlag, err := varflag.New("profile", profile, "session profile to be used")
	if err != nil {
		return err
	}
	rootCmd.AddFlag(profileFlag)
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
		for _, cmd := range addon.cmds {
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
		opts, err := NewOptions(addon.info.Name, addon.acceptsOpts)
		if err != nil {
			return err
		}
		// first use
		rtopts := a.session.RuntimeOpts()
		if rtopts != nil {
			for _, rtopt := range rtopts.All() {
				if !strings.HasPrefix(rtopt.Name(), addon.info.Name+".") {
					continue
				}
				key := strings.TrimPrefix(rtopt.Name(), addon.info.Name+".")
				if err := opts.Set(key, rtopt); err != nil {
					return err
				}
			}
		}

		// map addon settings to session options
		for _, gopt := range addon.acceptsOpts {
			gkey := addon.info.Name + "." + gopt.key
			if eopt, ok := a.session.opts.config[gkey]; ok {
				return fmt.Errorf("%w: option %q already in use (%s)", ErrOption, gkey, eopt.desc)
			}
			a.session.opts.config[gkey] = OptionArg{
				key:       gkey,
				desc:      gopt.desc,
				value:     gopt.value,
				kind:      gopt.kind,
				validator: gopt.validator,
			}
		}

		// apply options
		var pendingOpts []OptionArg

		for _, opt := range a.pendingOpts {
			if !strings.HasPrefix(opt.key, addon.info.Name+".") {
				pendingOpts = append(pendingOpts, opt)
				continue
			}

			key := strings.TrimPrefix(opt.key, addon.info.Name+".")
			if !opts.Accepts(key) {
				pendingOpts = append(pendingOpts, opt)
				continue
			}
			globalkey := opt.key
			opt.key = key
			if err := opt.apply(opts); err != nil {
				return err
			}

			// save it to session
			if err := a.session.Set(globalkey, opt.value); err != nil {
				return err
			}

		}
		if len(pendingOpts) != len(a.pendingOpts) {
			a.pendingOpts = pendingOpts
		}

		if err := opts.setDefaults(); err != nil {
			return err
		}

		if addon.registerAction != nil && !a.activeCmd.skipAddons {
			if err := addon.registerAction(a.session, opts); err != nil {
				return err
			}
		}
		provided = true
		a.logger.Debug(
			"registered addon",
			slog.Group("addon",
				slog.String("name", addon.info.Name),
				slog.String("version", addon.info.Version.String()),
			),
		)

		if !a.activeCmd.skipAddons {
			for _, svc := range addon.svcs {
				a.RegisterService(svc)
			}
		}

		for _, ev := range addon.events {
			if err := a.engine.registerEvent(ev); err != nil {
				return err
			}
		}

		if addon.API != nil {
			if err := a.session.registerAPI(addon.info.Name, addon.API); err != nil {
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
		registerEvent("services", "start.services", "starts local or connects remote service defined in payload", nil),
		registerEvent("services", "stop.services", "stops local or disconnects remote service defined in payload", nil),
		registerEvent("services", "service.started", "triggered when service has been started", nil),
		registerEvent("services", "service.stopped", "triggered when service has been stopped", nil),
	}

	for _, rev := range sysevs {
		if err := a.engine.registerEvent(rev); err != nil {
			return err
		}
	}
	a.logger.SystemDebug("registered system events")
	return nil
}
