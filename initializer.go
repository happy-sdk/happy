// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io/fs"
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
	"github.com/happy-sdk/happy/sdk/settings"
)

type initializer struct {
	mu         sync.Mutex
	logger     logging.Logger
	loaded     time.Time
	took       time.Duration
	settings   *Settings
	tick       ActionTick
	tock       ActionTock
	logQueue   chan logging.QueueRecord
	svcs       []*Service
	addons     []*Addon
	migrations *migration.Manager
	// pendingOpts contains options which are not yet applied.
	mainOpts    []OptionArg
	pendingOpts []OptionArg
}

func newInitializer(s *Settings) *initializer {
	return &initializer{
		settings: s,
		loaded:   time.Now(),
		logQueue: make(chan logging.QueueRecord, 100),
	}
}

func (i *initializer) SetLogger(l logging.Logger) {
	if l == nil {
		if i.logger != nil {
			i.logger.Warn("provided logger is nil")
		}
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.logger != nil {
		i.logger.Error("logger already set")
		return
	}
	i.logger = l
}

func (i *initializer) SetTick(t ActionTick) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.tick != nil {
		i.logger.Error("tick already set")
		return
	}
	i.tick = t
}

func (i *initializer) SetTock(t ActionTock) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.tock != nil {
		i.logger.Error("tock already set")
		return
	}
	i.tock = t
}

func (i *initializer) AddService(svc *Service) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if svc == nil {
		i.log(logging.NewQueueRecord(logging.LevelBUG, withServiceMsg, 4, slog.Any("service", nil)))
		return
	}
	i.log(logging.NewQueueRecord(logging.LevelSystemDebug, withServiceMsg, 4, slog.String("name", svc.name)))
	i.svcs = append(i.svcs, svc)
}

func (i *initializer) AddAddon(addon *Addon) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if addon == nil {
		i.log(logging.NewQueueRecord(logging.LevelBUG, withAddonMsg, 4, slog.Any("addon", nil)))
		return
	}
	i.log(logging.NewQueueRecord(logging.LevelSystemDebug, withAddonMsg, 4, slog.String("name", addon.info.Name)))

	i.addons = append(i.addons, addon)
}

// AddMigrations adds migration manager to application.
func (i *initializer) AddMigrations(mm *migration.Manager) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.migrations != nil {
		i.log(logging.NewQueueRecord(logging.LevelBUG, "migration manager already set", 4, slog.Any("migrations", nil)))
		return
	}
	if mm == nil {
		i.log(logging.NewQueueRecord(logging.LevelBUG, withMigrationsMsg, 4, slog.Any("migrations", nil)))
		return
	}
	i.log(logging.NewQueueRecord(logging.LevelSystemDebug, withMigrationsMsg, 4))
	i.migrations = mm
}

func (i *initializer) AddOptions(opts ...OptionArg) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.mainOpts = append(i.mainOpts, opts...)
}

func (i *initializer) Log(r logging.QueueRecord) {
	if i.logQueue == nil {
		if i.logger != nil {
			i.logger.BUG("log queue is already consumed", slog.Any("record", r.Record()))
			return
		}
		return
	}
	i.log(r)
}

func (i *initializer) dispose() error {
	i.took = time.Since(i.loaded)

	i.logger.SystemDebug("initialization done", slog.Duration("took", i.took))

	i.logger = nil
	i.tick = nil
	i.tock = nil
	i.svcs = nil
	i.addons = nil
	i.migrations = nil
	i.pendingOpts = nil

	return nil
}

func (i *initializer) log(r logging.QueueRecord) {
	i.logQueue <- r
}

var errExitSuccess = errors.New("exit status 0")

func (i *initializer) Initialize(m *Main) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	i.mu.Lock()
	defer i.mu.Unlock()

	defer func() {
		i.dispose()
		m.init = nil
	}()

	m.sealed = true
	m.sess = newSession(i.unsafeInitLogger())
	m.engine = newEngine()

	settingsb, err := i.settings.Blueprint()
	if err != nil {
		return err
	}

	if err := i.unsafeInitSettings(m, settingsb); err != nil {
		return err
	}

	inst, err := instance.New(m.slug)
	if err != nil {
		return err
	}
	m.instance = inst
	m.sess.opts.set("app.address", inst.Address().String(), true)

	if err := i.unsafeInitAddonSettingsAndCommands(m, settingsb); err != nil {
		return err
	}

	if err := i.unsafeInitRootCommand(m); err != nil {
		return err
	}

	if m.root.flag("system-debug").Var().Bool() {
		m.sess.Log().SetLevel(logging.LevelSystemDebug)
	} else if m.root.flag("debug").Var().Bool() {
		m.sess.Log().SetLevel(logging.LevelDebug)
	} else if m.root.flag("verbose").Var().Bool() {
		m.sess.Log().SetLevel(logging.LevelInfo)
	}

	m.sess.opts.setDefaults()

	if m.root.flag("version").Present() {
		m.sess.Log().SetLevel(logging.LevelQuiet)
		m.printVersion()
		return errExitSuccess
	}

	if m.cmd.flag("help").Present() {
		m.sess.Log().SetLevel(logging.LevelAlways)
		if err := m.help(); err != nil {
			return err
		}
		return errExitSuccess
	}

	close(i.logQueue)
	for r := range i.logQueue {
		_ = i.logger.Handle(r.Record())
	}
	i.logQueue = nil
	if err := i.unsafeConfigure(m, settingsb); err != nil {
		return err
	}

	for _, opt := range i.pendingOpts {
		group := "option"
		if opt.kind&ConfigOption != 0 {
			group = "config"
		} else if opt.kind&SettingsOption != 0 {
			group = "settings"
		}
		m.sess.Log().Warn("option not used", slog.Group(group,
			slog.String("key", opt.key),
			slog.Any("value", opt.value),
			slog.Bool("readOnly", opt.kind&ReadOnlyOption == ReadOnlyOption),
		))
	}

	if i.migrations != nil {
		m.sess.Log().NotImplemented("migrations not supported at the moment")
	}

	return nil
}

func (i *initializer) unsafeInitSettings(m *Main, settingsb *settings.Blueprint) error {
	for key := range m.sess.opts.config {
		for _, opt := range i.mainOpts {
			if opt.key == key {
				val, err := vars.NewValue(opt.value)
				if err != nil {
					i.log(logging.NewQueueRecord(logging.LevelError, fmt.Sprintf("%s: failed to parse value - %s", opt.key, err.Error()), 4))
					continue
				}
				if err := m.sess.Set(key, val); err != nil {
					i.log(logging.NewQueueRecord(logging.LevelError, fmt.Sprintf("%s: failed to set value - %s", opt.key, err.Error()), 4))
					continue
				}
				break
			} else {
				// extend
				m.sess.opts.config[opt.key] = opt
			}

		}
		// if !provided {
		// 	if err := a.session.opts.Set(key, cnf.value); err != nil {
		// 		errs = append(errs, err)
		// 		continue
		// 	}
		// }
	}

	for _, opt := range i.mainOpts {
		if !m.sess.Has(opt.key) {
			i.pendingOpts = append(i.pendingOpts, opt)
		}
	}

	slugSpec, slugErr := settingsb.GetSpec("app.slug")
	if err := errors.Join(slugErr); err != nil {
		return err
	}
	m.slug = slugSpec.Value

	if len(m.slug) == 0 {
		m.slug = m.instance.Address().Instance()
	}
	if err := slugSpec.ValidateValue(m.slug); err != nil {
		return err
	}
	i.log(logging.NewQueueRecord(logging.LevelSystemDebug, "app slug set to", 2, slog.String("slug", m.slug)))
	return nil
}

func (i *initializer) unsafeInitLogger() logging.Logger {
	if i.logger == nil {
		i.logger = logging.Default(logging.LevelOk)
	}
	return i.logger
}

func (i *initializer) unsafeInitAddonSettingsAndCommands(m *Main, settingsb *settings.Blueprint) error {
	var provided bool
	for _, addon := range i.addons {
		for _, cmd := range addon.cmds {
			if err := cmd.Err(); err != nil {
				return err
			}
			m.WithCommand(cmd)
			provided = true
		}
		// settings
		if addon.settings != nil {
			if err := settingsb.Extend(addon.info.Name, addon.settings); err != nil {
				return err
			}
		}
	}

	if provided {
		i.log(logging.NewQueueRecord(
			logging.LevelSystemDebug, "attached commands provided by addons", 2))
	}
	return nil
}

func (i *initializer) unsafeInitRootCommand(m *Main) error {
	i.log(logging.NewQueueRecord(logging.LevelSystemDebug, "initializing root command", 3))
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
		m.root.AddFlag(f)
	}
	profileFlag, err := varflag.New("profile", "default", "session profile to be used")
	if err != nil {
		return err
	}
	m.root.AddFlag(profileFlag)

	if err := m.root.verify(); err != nil {
		return err
	}
	if err := m.root.flags.Parse(os.Args); err != nil {
		return err
	}

	settree := m.root.flags.GetActiveSets()
	name := settree[len(settree)-1].Name()
	if name == "/" {
		m.cmd = m.root
		// only set app tick tock if current command is root command
		m.engine.onTick(m.init.tick)
		m.engine.onTock(m.init.tock)
	}

	// Handle subcommand is set
	cmd, err := m.root.getActiveCommand()
	if err != nil {
		return err
	}
	m.cmd = cmd
	if m.cmd != m.root {
		if err := m.cmd.Err(); err != nil {
			return err
		}
	}

	return m.cmd.Err()
}

// configure is called after logger is set to correct level.
func (i *initializer) unsafeConfigure(m *Main, settingsb *settings.Blueprint) error {
	if err := m.sess.opts.set("app.main.exec.x", m.root.flag("x").Present(), true); err != nil {
		return err
	}

	var profileName string
	profileName = "default"

	profileName = m.root.flag("profile").String()
	if m.sess.Get("app.devel").Bool() {
		profileName += "-devel"
	}
	m.sess.opts.set("app.profile.name", profileName, true)

	if err := i.unsafeConfigurePaths(m, settingsb); err != nil {
		return err
	}

	if err := i.unsafeConfigureProfile(m, settingsb); err != nil {
		return err
	}

	if err := i.unsafeConfigureSystemEvents(m); err != nil {
		return err
	}

	for _, svc := range m.init.svcs {
		if err := m.engine.registerService(m.sess, svc); err != nil {
			return err
		}
	}

	m.sess.Log().LogDepth(3, logging.LevelSystemDebug, "initialize", slog.Bool("firstuse", m.sess.Get("app.firstuse").Bool()))

	if err := i.unsafeConfigureOptions(m); err != nil {
		return err
	}

	if err := i.unsafeConfigureAddons(m); err != nil {
		return err
	}

	return nil
}

func (i *initializer) unsafeConfigurePaths(m *Main, settingsb *settings.Blueprint) error {
	profileName := m.sess.Get("app.profile.name").String()
	if profileName == "" {
		return fmt.Errorf("%w: profile name is empty", Error)
	}

	m.sess.Log().LogDepth(3, logging.LevelSystemDebug, "initializing paths")
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := m.sess.opts.set("app.fs.path.pwd", wd, true); err != nil {
		return err
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err := m.sess.opts.set("app.fs.path.home", userHomeDir, true); err != nil {
		return err
	}

	if len(m.slug) == 0 {
		return fmt.Errorf("%w: slug is empty", Error)
	}

	dir := m.slug

	if profileName != "default" {
		dir = filepath.Join(dir, "profiles", profileName)
	}

	// tmp
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d", dir, time.Now().UnixMilli()))
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return err
	}
	if err := m.sess.opts.set("app.fs.path.tmp", tempDir, true); err != nil {
		return err
	}
	m.exitFunc = append(m.exitFunc, func(sess *Session, code int) error {
		tmp := os.TempDir()
		if !strings.HasPrefix(tempDir, tmp) {
			return fmt.Errorf("%w: invalid tmp dir %s", Error, tempDir)
		}
		sess.Log().SystemDebug("removing temp dir", slog.String("dir", tempDir))
		return os.RemoveAll(tempDir)
	})

	// cache dir
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
	}
	if err := m.sess.opts.set("app.fs.path.cache", appCacheDir, true); err != nil {
		return err
	}

	// config dir
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
		if err := m.sess.opts.set("app.firstuse", true, true); err != nil {
			return err
		}
	}

	if err := m.sess.opts.set("app.fs.path.config", appConfigDir, true); err != nil {
		return err
	}
	return nil
}

func (i *initializer) unsafeConfigureProfile(m *Main, settingsb *settings.Blueprint) error {
	profileName := m.sess.Get("app.profile.name").String()
	if profileName == "" {
		return fmt.Errorf("%w: profile name is empty", Error)
	}
	m.sess.Log().SystemDebug("load app profile", slog.String("profile", profileName))
	m.sess.opts.set("app.profile.name", profileName, true)

	cpath := m.sess.Get("app.fs.path.config").String()
	if cpath == "" {
		return fmt.Errorf("%w: config path empty", Error)
	}
	cfile := filepath.Join(cpath, "profile.preferences")
	m.sess.opts.set("app.profile.file", cfile, true)

	var pref *settings.Preferences
	if _, err := os.Stat(cfile); err != nil {
		m.sess.Log().SystemDebug("no profile found", slog.String("path", cfile))
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := m.sess.opts.set("app.firstuse", true, true); err != nil {
			return err
		}
	} else {
		m.sess.Log().SystemDebug("loading preferences from", slog.String("path", cfile))
		prefFile, err := os.Open(cfile)
		if err != nil {
			return err
		}
		defer prefFile.Close()
		var (
			data []string
		)
		dataDecoder := gob.NewDecoder(prefFile)
		if err = dataDecoder.Decode(&data); err != nil {
			return err
		}
		prefsMap, err := vars.ParseMapFromSlice(data)
		if err != nil {
			return err
		}
		pref = settings.NewPreferences()

		for _, d := range prefsMap.All() {
			pref.Set(d.Name(), d.Value().String())
		}
	}

	schema, err := settingsb.Schema(m.sess.Get("app.module").String(), m.sess.Get("app.version").String())
	if err != nil {
		return err
	}
	profile, err := schema.Profile(profileName, pref)
	if err != nil {
		return err
	}
	if !profile.Get("app.copyright.since").IsSet() {
		if err := profile.Set("app.copyright.since", settings.Uint(time.Now().Year())); err != nil {
			return err
		}
	}
	m.sess.setProfile(profile)
	m.sess.Log().SystemDebug("loaded profile", slog.String("name", profile.Name()))
	return nil
}

func (i *initializer) unsafeConfigureSystemEvents(m *Main) error {
	var sysevs = []Event{
		registrableEvent("services", "start.services", "starts local or connects remote service defined in payload", nil),
		registrableEvent("services", "stop.services", "stops local or disconnects remote service defined in payload", nil),
		registrableEvent("services", "service.started", "triggered when service has been started", nil),
		registrableEvent("services", "service.stopped", "triggered when service has been stopped", nil),
	}

	for _, rev := range sysevs {
		if err := m.engine.registerEvent(rev); err != nil {
			return err
		}
	}
	m.sess.Log().SystemDebug("registered system events", slog.Int("count", len(sysevs)))
	return nil
}

func (i *initializer) unsafeConfigureOptions(m *Main) error {
	// apply options
	var pendingOpts []OptionArg
	for _, opt := range i.pendingOpts {
		// apply if it is custom glopal option
		m.sess.Log().SystemDebug("opt", slog.Any(opt.key, opt.value))
		if _, ok := m.sess.opts.config[opt.key]; ok {
			if err := opt.apply(m.sess.opts); err != nil {
				return err
			}
			continue
		}
		pendingOpts = append(pendingOpts, opt)
	}
	i.pendingOpts = pendingOpts
	return nil
}

func (i *initializer) unsafeConfigureAddons(m *Main) error {
	var provided bool

	for _, addon := range i.addons {
		if len(addon.errs) > 0 {
			return errors.Join(addon.errs...)
		}

		opts, err := NewOptions(addon.info.Name, addon.acceptsOpts)
		if err != nil {
			return err
		}
		// first use
		rtopts := m.sess.Opts()
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
			if eopt, ok := m.sess.opts.config[gkey]; ok {
				return fmt.Errorf("%w: option %q already in use (%s)", ErrOption, gkey, eopt.desc)
			}
			m.sess.opts.config[gkey] = OptionArg{
				key:       gkey,
				desc:      gopt.desc,
				value:     gopt.value,
				kind:      gopt.kind,
				validator: gopt.validator,
			}
		}

		// apply options
		var pendingOpts []OptionArg

		for _, opt := range i.pendingOpts {
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
			if err := m.sess.Set(globalkey, opt.value); err != nil {
				return err
			}
		}

		if len(pendingOpts) != len(i.pendingOpts) {
			i.pendingOpts = pendingOpts
		}

		if err := opts.setDefaults(); err != nil {
			return err
		}

		if addon.registerAction != nil && !m.cmd.skipAddons {
			if err := addon.registerAction(m.sess, opts); err != nil {
				return err
			}
		}

		// Apply initial value
		for _, opt := range opts.db.All() {
			key := addon.info.Name + "." + opt.Name()
			if err := m.sess.Set(key, opt.Any()); err != nil {
				return err
			}
		}

		provided = true
		m.sess.Log().Debug(
			"registered addon",
			slog.String("name", addon.info.Name),
			slog.String("version", addon.info.Version.String()),
			slog.String("module", addon.info.Module),
		)

		if !m.cmd.skipAddons {
			for _, svc := range addon.svcs {
				if err := m.engine.registerService(m.sess, svc); err != nil {
					return err
				}
				m.sess.Log().SystemDebug("registered service", slog.String("service", svc.name))
			}
		}

		for _, ev := range addon.events {
			if err := m.engine.registerEvent(ev); err != nil {
				return err
			}
		}

		if addon.api != nil {
			if err := m.sess.registerAPI(addon.info.Name, addon.api); err != nil {
				return err
			}
		}
	}
	if provided {
		m.sess.Log().SystemDebug("registeration of addons completed")
	}

	return nil
}
