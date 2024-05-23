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
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/migration"
)

type initializer struct {
	mu         sync.Mutex
	logger     logging.Logger
	loaded     time.Time
	settings   *Settings
	tick       ActionTick
	tock       ActionTock
	logQueue   chan logging.QueueRecord
	svcs       []*Service
	addons     []*Addon
	migrations *migration.Manager
	// pendingOpts contains options which are not yet applied.
	mainOptSpecs []options.OptionSpec
	pendingOpts  []options.Arg
	took         time.Duration
	brand        Brand
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

func (i *initializer) AddOptions(opts ...options.OptionSpec) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.mainOptSpecs = append(i.mainOptSpecs, opts...)
}

func (i *initializer) SetOptions(args ...options.Arg) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.pendingOpts = append(i.pendingOpts, args...)
}

func (i *initializer) SetBrand(bfunc BrandFunc) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if bfunc == nil {
		i.log(logging.NewQueueRecord(logging.LevelBUG, "with brand", 4, slog.Any("brand", nil)))
		return
	}
	if brand, err := bfunc(); err != nil {
		i.log(logging.NewQueueRecord(logging.LevelError, "with brand", 4, slog.String("err", err.Error())))
	} else {
		i.brand = brand
	}
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

func (i *initializer) dispose() {
	i.logger = nil
	i.tick = nil
	i.tock = nil
	i.svcs = nil
	i.addons = nil
	i.migrations = nil
	i.pendingOpts = nil
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

	if err := i.unsafeInitBrand(m, settingsb); err != nil {
		return err
	}

	if err := i.unsafeInitAddonSettingsAndCommands(m, settingsb); err != nil {
		return err
	}

	if err := i.unsafeInitRootCommand(m, settingsb); err != nil {
		return err
	}

	if m.root.flag("system-debug").Var().Bool() {
		m.sess.Log().SetLevel(logging.LevelSystemDebug)
	} else if m.root.flag("debug").Var().Bool() {
		m.sess.Log().SetLevel(logging.LevelDebug)
	} else if m.root.flag("verbose").Var().Bool() {
		m.sess.Log().SetLevel(logging.LevelInfo)
	}

	if m.root.flag("version").Present() {
		m.sess.Log().SetLevel(logging.LevelQuiet)
		m.printVersion()
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

	if err := m.sess.opts.Set("app.pid", os.Getpid()); err != nil {
		return err
	}

	if err := m.sess.opts.Seal(); err != nil {
		return err
	}
	for _, opt := range i.pendingOpts {
		m.sess.Log().Warn("option not used",
			slog.String("key", opt.Key()),
			slog.Any("value", opt.Value()),
		)
	}

	m.sess.Log().SystemDebug("configuration done")

	m.root.desc = m.sess.Get("app.description").String()

	if !m.root.flag("help").Present() && i.migrations != nil {
		m.sess.Log().NotImplemented("migrations not supported at the moment")
	}

	if m.root.flag("help").Present() || m.cmd == nil || (m.cmd == m.root && m.root.doAction == nil) {
		if err := m.help(); err != nil {
			return fmt.Errorf("%w: failed to print help %w", Error, err)
		}
		return errExitSuccess
	}

	if err := m.sess.start(); err != nil {
		return fmt.Errorf("%w: failed to start session %w", Error, err)
	}

	i.took = time.Since(i.loaded)
	i.logger.SystemDebug("initialization done", slog.String("took", i.took.String()))
	return i.boot(m)
}

func (i *initializer) unsafeInitLogger() logging.Logger {
	if i.logger == nil {
		i.logger = logging.Default(logging.LevelOk)
	}
	slog.SetDefault(i.logger.Logger())
	return i.logger
}

func (i *initializer) unsafeInitAddonSettingsAndCommands(m *Main, settingsb *settings.Blueprint) error {
	var provided bool
	for _, addon := range i.addons {
		for _, cmd := range addon.cmds {
			if err := cmd.err(); err != nil {
				return err
			}
			m.root.AddSubCommand(cmd)
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

func (i *initializer) unsafeInitRootCommand(m *Main, settingsb *settings.Blueprint) error {
	i.log(logging.NewQueueRecord(logging.LevelSystemDebug, "initializing root command", 3))
	var boolflags = []struct {
		Name    string
		Value   bool
		Usage   string
		Aliases []string
	}{
		{"version", false, "print application version", nil},
		{"x", false, "the -x flag prints all the cli commands as they are executed.", nil},
		{"system-debug", false, "enable system debug log level (very verbose)", nil},
		{"debug", false, "enable debug log level. when debug flag is after the command then debug level will be enabled only for that command", nil},
		{"verbose", false, "enable verbose log level", []string{"v"}},
		{"help", false, "display help or help for the command. [...command --help]", []string{"h"}},
	}
	for _, flag := range boolflags {
		m.root.AddFlag(varflag.BoolFunc(flag.Name, flag.Value, flag.Usage, flag.Aliases...))
	}

	m.root.AddFlag(varflag.StringFunc("profile", "public", "session profile to be used"))

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

	return m.cmd.err()
}

// configure is called after logger is set to correct level.
func (i *initializer) unsafeConfigure(m *Main, settingsb *settings.Blueprint) error {
	if err := m.sess.opts.Set("app.main.exec.x", m.root.flag("x").Present()); err != nil {

		return fmt.Errorf("%w: unsafeConfigure %s", Error, err)
	}

	profileName := m.root.flag("profile").String()
	if m.sess.Get("app.devel").Bool() {
		profileName += "-devel"
	}
	if err := m.sess.opts.Set("app.profile.name", profileName); err != nil {
		return err
	}

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

	m.sess.Log().LogDepth(2, logging.LevelSystemDebug, "initialize", slog.Bool("firstuse", m.sess.Get("app.firstuse").Bool()))

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

	if len(m.slug) == 0 {
		return fmt.Errorf("%w: can not configure paths slug is empty", Error)
	}

	dir := m.slug
	if profileName != "public" {
		dir = filepath.Join(dir, "profiles", profileName)
	}

	// tmp
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d", dir, time.Now().UnixMilli()))
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return err
	}
	if err := m.sess.opts.Set("app.fs.path.tmp", tempDir); err != nil {
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

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := m.sess.opts.Set("app.fs.path.home", userHomeDir); err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := m.sess.opts.Set("app.fs.path.pwd", wd); err != nil {
		return err
	}

	var (
		userCacheDir  string
		userConfigDir string
	)

	if testing.Testing() {
		userCacheDir = filepath.Join(tempDir, "cache")
		userConfigDir = filepath.Join(tempDir, "config")
	} else {
		userCacheDir, err = os.UserCacheDir()
		if err != nil {
			return err
		}
		// config dir
		userConfigDir, err = os.UserConfigDir()
		if err != nil {
			return err
		}
	}

	//  cache dir
	appCacheDir := filepath.Join(userCacheDir, dir)
	_, err = os.Stat(appCacheDir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(appCacheDir, 0700); err != nil {
			return err
		}
	}
	if err := m.sess.opts.Set("app.fs.path.cache", appCacheDir); err != nil {
		return err
	}

	appConfigDir := filepath.Join(userConfigDir, dir)
	_, err = os.Stat(appConfigDir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(appConfigDir, 0700); err != nil {
			return err
		}
		if err := m.sess.opts.Set("app.firstuse", true); err != nil {
			return err
		}
	}

	if err := m.sess.opts.Set("app.fs.path.config", appConfigDir); err != nil {
		return err
	}

	pidsDir := filepath.Join(appConfigDir, "pids")
	_, err = os.Stat(pidsDir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(pidsDir, 0700); err != nil {
			return err
		}
	}

	if err := m.sess.opts.Set("app.fs.path.pids", pidsDir); err != nil {
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
	if err := m.sess.opts.Set("app.profile.name", profileName); err != nil {
		return err
	}

	cpath := m.sess.Get("app.fs.path.config").String()
	if cpath == "" {
		return fmt.Errorf("%w: config path empty", Error)
	}
	cfile := filepath.Join(cpath, "profile.preferences")
	if err := m.sess.opts.Set("app.profile.preferences", cfile); err != nil {
		return err
	}

	var pref *settings.Preferences
	if _, err := os.Stat(cfile); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: error loading profile %s - %s", Error, profileName, err.Error())
		}
		if err := m.sess.opts.Set("app.firstuse", true); err != nil {
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
	if !profile.Get("app.copyright_since").IsSet() {
		if err := profile.Set("app.copyright_since", settings.Uint(time.Now().Year())); err != nil {
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
	var pendingOpts []options.Arg
	for _, opt := range i.pendingOpts {
		m.sess.Log().SystemDebug("config", slog.Any(opt.Key(), opt.Value()))
		if m.sess.opts.Accepts(opt.Key()) {
			if err := m.sess.opts.Set(opt.Key(), opt.Value()); err != nil {
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

		if err := options.MergeOptions(m.sess.opts, addon.opts); err != nil {
			return err
		}

		// apply options
		var pendingOpts []options.Arg

		for _, opt := range i.pendingOpts {
			if !strings.HasPrefix(opt.Key(), addon.info.Name+".") {
				pendingOpts = append(pendingOpts, opt)
				continue
			}

			// save it to session
			if err := m.sess.Set(opt.Key(), opt.Value()); err != nil {
				return err
			}
		}

		if len(pendingOpts) != len(i.pendingOpts) {
			i.pendingOpts = pendingOpts
		}

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

		provided = true
		m.sess.Log().Debug(
			"registered addon",
			slog.String("name", addon.info.Name),
			slog.String("version", addon.info.Version.String()),
			slog.String("module", addon.info.Module),
		)
	}
	if provided {
		m.sess.Log().SystemDebug("registeration of addons completed")
	}

	return nil
}

// session has been started now
func (i *initializer) boot(m *Main) error {
	m.sess.stats.Update()
	var stats = map[string]any{
		"app.initialization.took": i.took,
		"app.created.at":          m.sess.time(m.createdAt).Format(time.RFC3339),
		"app.started.at":          m.sess.time(m.startedAt).Format(time.RFC3339),
	}

	for _, addon := range i.addons {
		started := time.Now()
		if !m.cmd.skipAddons {
			if err := addon.register(m.sess); err != nil {
				return err
			}
			m.sess.stats.Update()
		}
		stats[fmt.Sprintf("addon.%s.OnRegister.elapsed", addon.info.Name)] = time.Since(started)
	}

	for k, v := range stats {
		if err := m.sess.stats.Set(k, v); err != nil {
			return err
		}
	}

	if err := m.instance.Boot(m.sess.Get("app.fs.path.pids").String()); err != nil {
		return err
	}

	m.exitFunc = append(m.exitFunc, func(sess *Session, code int) error {
		sess.Log().SystemDebug("shutdown instance...")
		return m.instance.Shutdown()
	})
	return nil
}
