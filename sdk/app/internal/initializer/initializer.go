// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package initializer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/happy-sdk/happy/pkg/branding"
	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/logging"
	consoleadapter "github.com/happy-sdk/happy/pkg/logging/adapters/console"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/tui/ansicolor"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/addon"
	"github.com/happy-sdk/happy/sdk/app/internal/application"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/internal"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

var Error = errors.New("initialization error")

const i18np = "com.github.happy-sdk.happy.sdk.app.internal.initializer"

type Initializer struct {
	mu  sync.RWMutex
	log *logging.QueueLogger

	// user defined logger or default logger for runtime
	logger  logging.Logger
	execlvl logging.Level

	opts      *options.Spec
	settings  settings.Settings
	settingsb *settings.Blueprint
	profile   *settings.Profile
	session   *session.Context
	addonm    *addon.Manager

	errs []error

	// root command configurator
	main *command.Command
	// parsed active command
	cmd *command.Cmd

	mainOptSpecs []*options.OptionSpec
	pendingOpts  []options.Arg

	brand        *branding.Brand
	brandBuilder *branding.Builder

	sessionReadyEvent events.Event
	evch              chan events.Event

	mw middleware

	// failed indicates that initialization failed,
	// and attemts to wait until logger becomes available
	failed bool

	pid       int
	createdAt time.Time

	rt *application.Runtime

	defaults *defaults

	context    context.Context
	cancelFunc context.CancelFunc
}

type fallbackLanguageGetter interface {
	GetFallbackLanguage() string
}

func New(s settings.Settings, rt *application.Runtime, log *logging.QueueLogger) *Initializer {
	init := &Initializer{
		log:       log,
		settings:  s,
		addonm:    addon.NewManager(),
		pid:       os.Getpid(),
		createdAt: time.Now(),
		rt:        rt,
		defaults:  &defaults{},
		execlvl:   logging.LevelQuiet,
	}

	init.context, init.cancelFunc = context.WithCancel(context.Background())

	if i18n.Enabled {
		fallbackLang := language.English
		if langGetter, ok := s.(fallbackLanguageGetter); ok {
			lang, err := language.Parse(langGetter.GetFallbackLanguage())
			if err == nil {
				fallbackLang = lang
			}
		}
		internal.Log(log, "loading i18n")
		i18n.Initialize(fallbackLang)
	}

	init.log.LogDepth(3, logging.LevelDebug, i18n.PTD(i18np, "initializing", "initializing"), slog.String("pid", fmt.Sprint(init.pid)))
	init.initialize()
	return init
}

func (init *Initializer) HasFailed() bool {
	return len(init.errs) > 0
}

// ////////////////////////////////////////////////////////////////////////////
// Configuration middlewares
type middleware struct {
	mainAfterAlways  string
	mainAfterFailure string
	mainAfterSuccess string
	mainBefore       string
	beforeAlways     string
}

func (init *Initializer) MainAddInfo(paragraph string) {
	init.mu.RLock()
	defer init.mu.RUnlock()
	init.main.AddInfo(paragraph)
}

func (init *Initializer) MainAfterAlways(a action.WithPrevErr) {
	init.mu.Lock()
	defer init.mu.Unlock()
	if a == nil {
		init.bug(2, "attached <nil>", slog.String("action", "MainAfterAlways"))
		return
	}

	if init.mw.mainAfterAlways != "" {
		init.errAllowedOnce(fmt.Sprintf("%s AfterAlways action can only be set once", init.defaults.slug), init.mw.mainAfterAlways)
		return
	}

	init.main.AfterAlways(a)
	var ok bool
	init.mw.mainAfterAlways, ok = internal.RuntimeCallerStr(3)
	if !ok {
		init.bug(2, "MainAfterAlways: failed to get runtime caller")
	}
}

func (init *Initializer) MainAfterFailure(a action.WithPrevErr) {
	init.mu.Lock()
	defer init.mu.Unlock()
	if a == nil {
		init.bug(2, "attached <nil>", slog.String("action", "MainAfterFailure"))
		return
	}

	if init.mw.mainAfterFailure != "" {
		init.errAllowedOnce(fmt.Sprintf("%s AfterFailure action can only be set once", init.defaults.slug), init.mw.mainAfterFailure)
		return
	}

	init.main.AfterFailure(a)
	var ok bool
	init.mw.mainAfterFailure, ok = internal.RuntimeCallerStr(3)
	if !ok {
		init.bug(2, "MainAfterFailure: failed to get runtime caller")
	}
}

func (init *Initializer) MainAfterSuccess(a action.Action) {
	init.mu.Lock()
	defer init.mu.Unlock()

	if a == nil {
		init.bug(2, "attached <nil>", slog.String("action", "MainAfterSuccess"))
		return
	}

	if init.mw.mainAfterSuccess != "" {
		init.errAllowedOnce(fmt.Sprintf("%s AfterSuccess action can only be set once", init.defaults.slug), init.mw.mainAfterSuccess)
		return
	}

	init.main.AfterSuccess(a)
	var ok bool
	init.mw.mainAfterSuccess, ok = internal.RuntimeCallerStr(3)
	if !ok {
		init.bug(2, "attachRootAfterSuccess: failed to get runtime caller")
	}
}

func (init *Initializer) MainBefore(a action.WithArgs) {
	init.mu.Lock()
	defer init.mu.Unlock()

	if a == nil {
		init.bug(2, "attached <nil>", slog.String("action", "Before"))
		return
	}

	if init.mw.mainBefore != "" {
		init.errAllowedOnce(fmt.Sprintf("%s Before action can only be set once", init.defaults.slug), init.mw.mainBefore)
		return
	}

	init.main.Before(a)
	var ok bool
	init.mw.mainBefore, ok = internal.RuntimeCallerStr(3)
	if !ok {
		init.bug(2, "attachRootBefore: failed to get runtime caller")
	}
}

func (init *Initializer) MainBeforeAlways(rt *application.Runtime, a action.WithArgs) {
	init.mu.Lock()
	defer init.mu.Unlock()
	if a == nil {
		init.bug(1, "attached <nil>", slog.String("action", "BeforeAlways"))
		return
	}
	if init.mw.beforeAlways != "" {
		init.errAllowedOnce(fmt.Sprintf("%s BeforeAlways action can only be set once", init.defaults.slug), init.mw.beforeAlways)
		return
	}
	if err := rt.WidthBeforeAlways(a); err != nil {
		init.error(err)
	}
	var ok bool
	init.mw.beforeAlways, ok = internal.RuntimeCallerStr(3)
	if !ok {
		init.bug(1, "MainBeforeAlways: failed to get runtime caller")
	}
}

func (init *Initializer) SetOptions(a ...options.Arg) {
	init.mu.Lock()
	defer init.mu.Unlock()
	init.pendingOpts = append(init.pendingOpts, a...)
}

func (init *Initializer) MainTick(a action.Tick) {
	init.mu.Lock()
	defer init.mu.Unlock()
	init.rt.SetMainTick(a)
}

func (init *Initializer) MainTock(a action.Tock) {
	init.mu.Lock()
	defer init.mu.Unlock()
	init.rt.SetMainTock(a)
}

func (init *Initializer) MainAddCommands(cmds []*command.Command) {
	init.mu.RLock()
	defer init.mu.RUnlock()
	init.main.WithSubCommands(cmds...)
}

func (init *Initializer) MainAddFlags(ffns []varflag.FlagCreateFunc) {
	init.mu.RLock()
	defer init.mu.RUnlock()
	init.main.WithFlags(ffns...)
}

func (init *Initializer) WithAddon(a *addon.Addon) {
	if err := init.addonm.Add(a); err != nil {
		init.bug(1, err.Error())
	}
}
func (init *Initializer) WithBrand(b *branding.Builder) {
	init.brandBuilder = b
}

func (init *Initializer) MainDo(a action.WithArgs) {
	init.mu.Lock()
	defer init.mu.Unlock()
	init.main.Do(a)
}

func (init *Initializer) SetLogger(logger logging.Logger) {
	init.mu.Lock()
	defer init.mu.Unlock()
	init.logger = logger
}

func (init *Initializer) WithOptions(opts []*options.OptionSpec) {
	init.mu.Lock()
	defer init.mu.Unlock()
	init.mainOptSpecs = append(init.mainOptSpecs, opts...)
}

func (init *Initializer) WithSetup(action action.Action) {
	init.mu.Lock()
	defer init.mu.Unlock()
	init.rt.SetSetup(action)
}

func (init *Initializer) WithSettings(s settings.Settings) {
	init.mu.Lock()
	defer init.mu.Unlock()

	if err := init.settingsb.Extend(init.defaults.slug, s); err != nil {
		init.error(err)
	}
}

func (init *Initializer) Configure() (err error) {
	defer func() {
		if init.logger != nil {
			init.rt.SetLogger(init.logger)
		}
	}()
	errs := errors.Join(init.errs...)
	if errs != nil {
		return errs
	}

	// Setup addons
	if err := init.configureAddons(); err != nil {
		return err
	}

	// Add custom global options
	for _, opt := range init.mainOptSpecs {
		if err := init.opts.Add(opt); err != nil {
			return err
		}
	}
	init.mainOptSpecs = nil

	// parse commandline arguments and get active command
	clierr := init.configureCli()

	if err := init.configureProfile(); err != nil {
		return err
	}
	// Setup brand
	if err := init.configureBrand(); err != nil {
		return err
	}
	// Configure logger
	if err := init.configureLogger(); err != nil {
		return err
	}
	if clierr != nil {
		return clierr
	}
	if init.failed {
		return fmt.Errorf("%w: initialization failed", Error)
	}

	// Apply custom application options
	if err := init.configureApplyCustomOptions(); err != nil {
		return err
	}

	errs = errors.Join(init.errs...)
	if errs != nil {
		return errs
	}

	if err := init.configureSession(); err != nil {
		return err
	}

	internal.LogInit(init.session.Log(), "configuration completed")
	return
}

func (init *Initializer) Finalize() (err error) {
	for _, opt := range init.pendingOpts {
		init.session.Log().Warn("option not used",
			slog.String("key", opt.Key()),
			slog.Any("value", opt.Value()),
		)
	}
	init.pendingOpts = nil

	init.rt.SetMain(init.cmd)
	init.cmd = nil

	session := init.session
	init.rt.SetSession(session)
	init.session = nil

	init.rt.SetBrand(init.brand)
	init.brand = nil

	init.rt.SetSessionReady(init.evch, init.sessionReadyEvent)
	init.sessionReadyEvent = nil
	init.evch = nil

	init.rt.SetAddonManager(init.addonm)
	init.addonm = nil

	took := time.Since(init.createdAt)
	init.rt.InitStats(init.createdAt, took)
	init.rt.SetExecLogLevel(init.execlvl)

	session.Log().LogDepth(1, logging.LevelDebug, i18n.PTD(i18np, "initialization_completed", "initialization completed"), slog.String("took", took.String()))

	return nil
}

func (init *Initializer) SystemDebug(r slog.Record) {
	if init.logger != nil {
		_ = init.logger.Logger().Handler().Handle(context.Background(), r)
	} else {
		_ = slog.Default().Handler().Handle(context.Background(), r)
	}
}

// ////////////////////////////////////////////////////////////////////////////
// Configuration stage

func (init *Initializer) configureAddons() error {
	internal.LogInitDepth(init.log, 1, "configuring addons")
	if err := init.addonm.ExtendSettings(init.settingsb); err != nil {
		return err
	}
	if err := init.addonm.ExtendOptions(init.opts); err != nil {
		return err
	}
	commands := init.addonm.Commands()
	init.main.WithSubCommands(commands...)

	init.rt.AddServices(init.addonm.Services())

	if len(commands) > 0 {
		internal.Log(init.log, "added addons commands", slog.Int("count", len(commands)))
	}
	return nil
}

func (init *Initializer) configureCli() error {
	internal.LogInitDepth(init.log, 1, "configuring command line interface")

	cmd, cmdlog, err := command.Compile(init.main)
	logerr := init.log.ConsumeQueue(cmdlog)
	if logerr != nil {
		return fmt.Errorf("%w: failed to consume command log: %s", Error, logerr)
	}
	if err != nil {
		return err
	}

	if err := init.opts.Set("app.main.exec.x", cmd.Flag("x").Present()); err != nil {
		return fmt.Errorf("%w: unsafeConfigure %s", Error, err)
	}

	if cmd.Flag("x-prod").Var().Bool() {
		if err := init.opts.Set("app.is_devel", false); err != nil {
			return fmt.Errorf("%w: failed to set app.is_devel: %s", Error, err.Error())
		}
	}

	init.cmd = cmd
	init.main = nil

	if cmd.Flag("version").Present() {
		fmt.Println(init.opts.Get("app.version").String())
		return application.ErrExitSuccess
	}

	return nil
}

func (init *Initializer) configureProfile() (err error) {
	internal.LogInitDepth(init.log, 1, "configuring profile")
	const prefFilename = "profile.preferences"
	if init.opts.Get("app.fs.path.config").Value().Empty() {
		return fmt.Errorf("%w: config path is empty", Error)
	}
	var (
		isDevel     = init.opts.Get("app.is_devel").Variable().Bool()
		profilesDir = filepath.Join(init.opts.Get("app.fs.path.config").String(), "profiles")

		currentProfileName = init.opts.Get("app.profile.name").String()
		defaultProfileName = init.defaults.configDefaultProfile
		loadSlug           = init.defaults.configDefaultProfile

		pref *settings.Preferences
	)

	var profileExists = func(slug string) bool {
		if _, err := os.Stat(filepath.Join(profilesDir, slug, prefFilename)); err == nil {
			return true
		}
		return false
	}

	// Function check does given profile exists
	if init.defaults.configDisabled {
		loadProfileConfigDir := filepath.Join(profilesDir, loadSlug)
		if err := init.opts.Set("app.fs.path.profile.config", loadProfileConfigDir); err != nil {
			return err
		}
		goto LoadProfile
	}

	// Check which profile to load
	{
		// Determine profile to load
		if init.cmd != nil && init.cmd.Flag("profile").Present() {
			currentProfileName = init.cmd.Flag("profile").String()
			if len(currentProfileName) == 0 {
				return fmt.Errorf("%w: profile name is empty", Error)
			}

			// Check if loading other than default profile
			if currentProfileName != defaultProfileName {
				// When custom profiles are not allowed check if current profile is allowed
				if init.defaults.configAllowCustomProfiles {
					if !slices.Contains(init.defaults.configAdditionalProfiles, currentProfileName) {
						return fmt.Errorf("%w: profile %q is not allowed", Error, currentProfileName)
					}
				}
			}
		}

		// Get profile slug to load
		{
			var attrs []slog.Attr
			attrs = append(attrs, slog.String("profile", currentProfileName))
			loadSlug = currentProfileName
			mode := "production"
			if isDevel && init.defaults.configEnableProfileDevel {
				loadSlug += "-devel"
				mode = "devel"
			}
			attrs = append(attrs, slog.String("mode", mode))
			if currentProfileName != defaultProfileName {
				attrs = append(attrs, slog.String("default", defaultProfileName))
			}
			internal.LogInit(init.log, "load profile", attrs...)
		}
	}

	{
		// Check if requested profile exists
		if profileExists(loadSlug) {
			// Go straight to loading the profile
			// However currently that would not create other additional profiles on app version update
			goto LoadPreferences
		}

		// Check if default profile exists, if not then create it
		if currentProfileName == defaultProfileName {
			var defaultProfiles = []string{defaultProfileName}

			defaultProfiles = append(defaultProfiles, init.defaults.configAdditionalProfiles...)

			for _, dp := range defaultProfiles {
				if !profileExists(dp) && (dp == defaultProfileName) {
					if err := init.utilMkdir("create default profile directory", filepath.Join(profilesDir, dp), 0700); err != nil {
						return fmt.Errorf("%w: failed to create default profile directory %s", Error, err)
					}
					if err := init.utilWriteFile("write default profile preferences", filepath.Join(profilesDir, dp, prefFilename), []byte{}, 0600); err != nil {
						return fmt.Errorf("%w: failed to write default profile preferences %s", Error, err)
					}
					internal.LogInit(init.log, "created default profile", slog.String("profile", dp))
					if err := init.opts.Set("app.firstrun", true); err != nil {
						return err
					}
				}
			}
		}

		// Set init err when profile exists, but no development profile exists.
		if isDevel && profileExists(currentProfileName) && !profileExists(loadSlug) {

			if err := init.utilMkdir("create default profile directory", filepath.Join(profilesDir, loadSlug), 0700); err != nil {
				return fmt.Errorf("%w: failed to create development profile directory for %s profile: %s", Error, currentProfileName, err)
			}
			if err := init.utilWriteFile("write default profile preferences", filepath.Join(profilesDir, loadSlug, prefFilename), []byte{}, 0600); err != nil {
				return fmt.Errorf("%w: failed to write development profile preferences for %s profile:  %s", Error, currentProfileName, err)
			}
			goto LoadPreferences
		}
	}

LoadPreferences:
	{
		loadProfileConfigDir := filepath.Join(profilesDir, loadSlug)
		if err := init.opts.Set("app.fs.path.profile.config", loadProfileConfigDir); err != nil {
			return err
		}
		loadPrefFilePath := filepath.Join(loadProfileConfigDir, prefFilename)

		if _, err := os.Stat(loadPrefFilePath); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("%w: profile %q does not exist", Error, currentProfileName)
			}
			return fmt.Errorf("%w: profile %q loading error: %s", Error, currentProfileName, err.Error())
		} else {
			internal.LogInit(init.log, "loading preferencess from", slog.String("path", loadPrefFilePath))
			prefFile, err := os.ReadFile(loadPrefFilePath)
			if err != nil {
				return err
			}

			pref = settings.NewPreferences(version.Version("v1.0.0"))

			if err = pref.GobDecode(prefFile); err != nil && !errors.Is(err, io.EOF) {
				return err
			}
		}
	}

LoadProfile:
	schema, err := init.settingsb.Schema(init.opts.Get("app.module").String(), version.Version("v1.0.0"))
	if err != nil {
		return err
	}

	init.profile, err = schema.Profile(currentProfileName, pref)
	if err != nil {
		return err
	}
	defer func() {
		// dereference the settings bluepirnt
		init.settings = nil
		init.settingsb = nil
	}()

	// Set profile cache directory
	profileCacheDir := filepath.Join(init.opts.Get("app.fs.path.cache").String(), "profiles", loadSlug)
	_, err = os.Stat(profileCacheDir)
	if errors.Is(err, fs.ErrNotExist) && !init.defaults.configDisabled {
		if err := init.utilMkdir("create cache directory", profileCacheDir, 0750); err != nil {
			return fmt.Errorf("%w: failed to create cache directory %s", Error, err)
		}
	}

	if err := init.opts.Set("app.fs.path.profile.cache", profileCacheDir); err != nil {
		return err
	}

	// Set profile run directory
	profileRunDir := filepath.Join(init.opts.Get("app.fs.path.run").String(), "profiles", loadSlug)
	_, err = os.Stat(profileRunDir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := init.utilMkdir("create profile run directory", profileRunDir, 0700); err != nil {
			return fmt.Errorf("%w: failed to create profile run  directory %s", Error, err)
		}
	}
	if err := init.opts.Set("app.fs.path.profile.run", profileRunDir); err != nil {
		return err
	}

	// Set profile data directory
	profileDataDir := filepath.Join(init.opts.Get("app.fs.path.data").String(), "profiles", loadSlug)
	if err := init.opts.Set("app.fs.path.profile.data", profileDataDir); err != nil {
		return err
	}
	_, err = os.Stat(profileDataDir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := init.utilMkdir("create profile data directory", profileDataDir, 0750); err != nil {
			return fmt.Errorf("%w: failed to create profile data directory %s", Error, err)
		}
	}

	// Set profile state directory
	profileStateDir := filepath.Join(init.opts.Get("app.fs.path.state").String(), "profiles", loadSlug)
	if err := init.opts.Set("app.fs.path.profile.state", profileStateDir); err != nil {
		return err
	}
	_, err = os.Stat(profileStateDir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := init.utilMkdir("create profile state directory", profileStateDir, 0750); err != nil {
			return fmt.Errorf("%w: failed to create profile profile directory %s", Error, err)
		}
	}

	// Set profile state directory
	profileLogsDir := filepath.Join(init.opts.Get("app.fs.path.state").String(), "profiles", loadSlug, "logs")
	if err := init.opts.Set("app.fs.path.profile.logs", profileLogsDir); err != nil {
		return err
	}
	_, err = os.Stat(profileLogsDir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := init.utilMkdir("create profile logs directory", profileLogsDir, 0750); err != nil {
			return fmt.Errorf("%w: failed to create profile logs directory %s", Error, err)
		}
	}

	return nil
}

func (init *Initializer) configureBrand() error {
	if init.brandBuilder != nil {
		internal.LogInitDepth(init.log, 1, "configuring custom brand")
		brand, err := init.brandBuilder.Build()
		if err != nil {
			return err
		}
		init.brand = brand
		return nil
	}
	internal.LogInitDepth(init.log, 1, "configuring default brand")

	builder := branding.New(branding.Info{
		Name:    init.opts.Get("app.name").String(),
		Slug:    init.opts.Get("app.slug").String(),
		Version: init.opts.Get("app.version").String(),
	})
	brand, err := builder.Build()
	if err != nil {
		return err
	}
	init.brand = brand
	return nil
}

func (init *Initializer) configureLogger() (err error) {
	internal.LogInitDepth(init.log, 1, "configuring logger")

	var (
		lvl             logging.Level
		noSlogDefault   bool
		withSource      bool
		tslocStr        string
		timestampFormat string
		noTimestamp     bool
	)
	if init.profile != nil {
		lvl, err = logging.LevelFromString(init.profile.Get("app.logging.level").Value().String())
		if err != nil {
			return err
		}
		noSlogDefault = init.profile.Get("app.logging.no_slog_default").Value().Bool()
		withSource = init.profile.Get("app.logging.with_source").Value().Bool()
		tslocStr = init.profile.Get("app.datetime.location").Value().String()
		timestampFormat = init.profile.Get("app.logging.timeestamp_format").Value().String()
		noTimestamp = init.profile.Get("app.logging.no_timestamp").Value().Bool()
	} else {
		lvl = logging.LevelDebug
		withSource = false
		tslocStr = "Local"
		timestampFormat = "15:04:05"
	}

	if init.cmd != nil {
		if init.cmd.Flag("system-debug").Var().Bool() {
			if init.cmd.Flag("system-debug").Global() {
				lvl = internal.LogLevelHappy
			} else {
				init.execlvl = internal.LogLevelHappy
			}
		} else if init.cmd.Flag("debug").Var().Bool() {
			if init.cmd.Flag("debug").Global() {
				lvl = logging.LevelDebug
			} else {
				init.execlvl = logging.LevelDebug
			}
		} else if init.cmd.Flag("verbose").Var().Bool() {
			if init.cmd.Flag("verbose").Global() {
				lvl = logging.LevelInfo
			} else {
				init.execlvl = logging.LevelInfo
			}
		}
	}

	if init.profile != nil {
		if err := init.profile.Set("app.logging.level", lvl); err != nil {
			return err
		}

	}

	slog.SetLogLoggerLevel(slog.Level(lvl))
	if init.logger != nil {
		init.logger.SetLevel(lvl)
		if err := init.logger.ConsumeQueue(init.log); err != nil {
			return fmt.Errorf("%w: failed to consume log queue: %s", Error, err)
		}
		init.log = nil
		return nil
	}

	logopts := logging.DefaultOptions()
	logopts.Level = lvl
	logopts.AddSource = withSource
	logopts.NoTimestamp = noTimestamp

	tsloc, err := time.LoadLocation(tslocStr)
	if err != nil {
		return err
	}
	logopts.TimeLocation = tsloc
	logopts.TimestampFormat = timestampFormat

	var theme ansicolor.Theme
	if init.brand != nil {
		theme = init.brand.ANSI()
	} else {
		theme = ansicolor.New()
	}

	logopts.SetSlogOutput = !noSlogDefault

	logger := logging.New(consoleadapter.New(
		init.context,
		os.Stdout,
		logopts,
		theme,
	))

	if err := logger.ConsumeQueue(init.log); err != nil {
		return fmt.Errorf("%w: failed to consume log queue: %s", Error, err)
	}
	init.log = nil

	init.logger = logger

	return nil
}

func (init *Initializer) configureApplyCustomOptions() error {
	internal.LogInitDepth(init.logger, 1, "configuring custom options", slog.Int("count", len(init.pendingOpts)))
	var pendingOpts []options.Arg
	for _, opt := range init.pendingOpts {
		if init.opts.Accepts(opt.Key()) {
			if err := init.opts.Set(opt.Key(), opt.Value()); err != nil {
				return err
			}
			continue
		}
		pendingOpts = append(pendingOpts, opt)
	}
	init.pendingOpts = pendingOpts
	return nil
}

func (init *Initializer) configureSession() error {
	internal.LogInitDepth(init.logger, 1, "configuring session")

	init.sessionReadyEvent = session.ReadyEvent()
	init.evch = make(chan events.Event, 1000)

	opts, err := init.opts.Seal()
	if err != nil {
		return err
	}
	sessconfig := session.Config{
		Profile:    init.profile,
		Logger:     init.logger,
		Opts:       opts,
		ReadyEvent: init.sessionReadyEvent,
		EventCh:    init.evch,
		APIs:       init.addonm.GetAPIs(),
		Context:    init.context,
		CancelFunc: init.cancelFunc,
	}

	session, err := sessconfig.Init()
	if err != nil {
		return err
	}

	init.session = session

	init.context = nil
	init.cancelFunc = nil
	init.profile = nil
	init.logger = nil
	init.opts = nil
	return nil
}

// ////////////////////////////////////////////////////////////////////////////
// Initializer utils

func (init *Initializer) utilMkdir(msg, path string, perm fs.FileMode) error {
	if path == "" {
		return fmt.Errorf("%w: %s (path is empty)", Error, msg)
	}
	internal.LogInitDepth(init.log, 1, msg, slog.String("dir", path))
	if err := os.MkdirAll(path, perm); err != nil {
		return err
	}
	return nil
}

func (init *Initializer) utilWriteFile(msg, name string, data []byte, perm fs.FileMode) error {
	if name == "" {
		return fmt.Errorf("%w: %s (file name is empty)", Error, msg)
	}
	internal.LogInitDepth(init.log, 1, msg, slog.String("file", name))
	if err := os.WriteFile(name, data, perm); err != nil {
		return err
	}
	internal.Log(init.log, msg, slog.String("file", name))
	return nil
}

func (init *Initializer) error(err error) {
	// skip lock if called by internal functions
	// which have already locked the mutex
	if init.mu.TryLock() {
		defer init.mu.Unlock()
	}
	if err != nil {
		init.errs = append(init.errs, err)
	}
}

func (init *Initializer) bug(depth int, msg string, attr ...slog.Attr) {
	// skip lock if called by internal functions
	// which have already locked the mutex
	if init.mu.TryLock() {
		defer init.mu.Unlock()
	}
	init.failed = true
	init.log.LogDepth(depth+1, logging.LevelBUG, msg, attr...)
}

func (init *Initializer) errAllowedOnce(msg string, prevcaller string) {
	init.errAllowedOnceDepth(2, msg, prevcaller)
}

func (init *Initializer) errAllowedOnceDepth(depth int, msg string, prevcaller string) {
	current, _ := internal.RuntimeCallerStr(depth)
	init.bug(6,
		msg,
		slog.String("previous", prevcaller),
		slog.String("current", current),
	)
}
