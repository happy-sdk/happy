// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package initializer

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/devel/goutils"
	"github.com/happy-sdk/happy/pkg/fsutils"
	"github.com/happy-sdk/happy/pkg/networking/address"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/cmd/config"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/instance"
	"github.com/happy-sdk/happy/sdk/internal"
	"github.com/happy-sdk/happy/sdk/session"
)

// defaults holds the default values for the application.
// until settings profile is loaded.
type defaults struct {
	configDisabled            bool
	slug                      string
	identifier                string
	configDefaultProfile      string
	configAdditionalProfiles  []string
	configAllowCustomProfiles bool
	configEnableProfileDevel  bool
	cliMainMinArgs            uint
	cliMainMaxArgs            uint
	cliWithConfigCmd          bool
	cliWithGlobalFlags        bool
	develAllowProd            bool
}

// initialize sets up the application logger, options, settings, and root command.
// and other minimal configurations required for the application configuration.
func (init *Initializer) initialize() {
	// Setup options and settings
	if err := init.initSettingsAndOpts(); err != nil {
		init.error(err)
		return
	}

	init.log.Debug(init.defaults.slug,
		slog.String("version", init.opts.Get("app.version").String()))

	// Set the application paths
	if err := init.initBasePaths(); err != nil {
		init.error(err)
		return
	}

	// Setup root command
	if err := init.initRootCommand(); err != nil {
		init.error(err)
		return
	}
}

func (init *Initializer) initSettingsAndOpts() (err error) {
	internal.LogInitDepth(init.log, 1, "initializing application options and settings")

	if init.settings == nil {
		err = fmt.Errorf("%w: settings is <nil>", Error)
		return
	}

	// Load settings blueprint
	if init.settingsb, err = init.settings.Blueprint(); err != nil {
		return
	}

	// Load defaults before profile is loaded
	configDisabledSpec, err := init.settingsb.GetSpec("app.profiles.disabled")
	if err != nil {
		return err
	}
	slugSpec, err := init.settingsb.GetSpec("app.slug")
	if err != nil {
		return err
	}
	identifierSpec, err := init.settingsb.GetSpec("app.identifier")
	if err != nil {
		return err
	}
	configDefaultProfileSpec, err := init.settingsb.GetSpec("app.profiles.default")
	if err != nil {
		return err
	}
	configAdditionalProfilesSpec, err := init.settingsb.GetSpec("app.profiles.additional")
	if err != nil {
		return err
	}
	configAllowCustomProfilesSpec, err := init.settingsb.GetSpec("app.profiles.allow_custom")
	if err != nil {
		return err
	}
	configEnableProfileDevelSpec, err := init.settingsb.GetSpec("app.profiles.enable_devel")
	if err != nil {
		return err
	}
	cliMainMinArgsSpec, err := init.settingsb.GetSpec("app.cli.main_min_args")
	if err != nil {
		return err
	}
	cliMainMinArgs, err := strconv.ParseUint(cliMainMinArgsSpec.Value, 10, 64)
	if err != nil {
		return err
	}
	cliMainMaxArgsSpec, err := init.settingsb.GetSpec("app.cli.main_max_args")
	if err != nil {
		return err
	}
	cliMainMaxArgs, err := strconv.ParseUint(cliMainMaxArgsSpec.Value, 10, 64)
	if err != nil {
		return err
	}
	cliWithConfigCmdSpec, err := init.settingsb.GetSpec("app.cli.with_config_cmd")
	if err != nil {
		return err
	}
	cliWithGlobalFlagsSpec, err := init.settingsb.GetSpec("app.cli.with_global_flags")
	if err != nil {
		return err
	}
	develAllowProdSpec, err := init.settingsb.GetSpec("app.devel.allow_prod")
	if err != nil {
		return err
	}

	init.defaults.configDisabled = configDisabledSpec.Value == "true"
	init.defaults.slug = slugSpec.Value
	init.defaults.identifier = identifierSpec.Value
	init.defaults.cliMainMinArgs = uint(cliMainMinArgs)
	init.defaults.cliMainMaxArgs = uint(cliMainMaxArgs)
	init.defaults.cliWithConfigCmd = cliWithConfigCmdSpec.Value == "true"
	init.defaults.cliWithGlobalFlags = cliWithGlobalFlagsSpec.Value == "true"
	init.defaults.develAllowProd = develAllowProdSpec.Value == "true"

	if init.defaults.configDisabled {
		init.defaults.configDefaultProfile = configDefaultProfileSpec.Default
		if len(configAdditionalProfilesSpec.Default) > 0 {
			init.defaults.configAdditionalProfiles = strings.Split(configAdditionalProfilesSpec.Default, "|")
		}
	} else {
		init.defaults.configDefaultProfile = configDefaultProfileSpec.Value
		if len(configAdditionalProfilesSpec.Value) > 0 {
			init.defaults.configAdditionalProfiles = strings.Split(configAdditionalProfilesSpec.Value, "|")
		}
		init.defaults.configAllowCustomProfiles = configAllowCustomProfilesSpec.Value == "true"
		init.defaults.configEnableProfileDevel = configEnableProfileDevelSpec.Value == "true"
	}

	var (
		module string
		addr   *address.Address
		ver    = version.Current()
	)

	addr, err = address.Current()
	if err != nil {
		return err
	}

	module = addr.Module()

	if len(init.defaults.slug) == 0 {
		if testing.Testing() {
			addr, err = address.CurrentForDepth(2)
			if err != nil {
				return err
			}

			init.defaults.slug = addr.Instance()

			module = addr.Module()
			init.defaults.identifier = addr.ReverseDNS()
		} else {
			init.defaults.slug = path.Base(module)
		}
		if err := init.settingsb.SetDefaultFromString("app.slug", init.defaults.slug); err != nil {
			return err
		}
	} else {
		if addr, err = addr.Parse(init.defaults.slug); err != nil {
			return err
		}
	}

	if len(init.defaults.identifier) == 0 {
		init.defaults.identifier = addr.ReverseDNS()
		if len(init.defaults.identifier) == 0 {
			return fmt.Errorf("could not find app.identifier")
		}
	}

	if err := init.settingsb.SetDefaultFromString("app.identifier", init.defaults.identifier); err != nil {
		return err
	}

	if err := slugSpec.ValidateValue(init.defaults.slug); err != nil {
		return err
	}

	if err := init.settingsb.SetDefaultFromString("app.copyright_since", fmt.Sprint(time.Now().Year())); err != nil {
		return err
	}

	binaryName := filepath.Base(os.Args[0]) + filepath.Ext(os.Args[0])
	if testing.Testing() {
		binaryName = "testing"
	}

	optSpecs := []*options.OptionSpec{
		options.NewOption("app.is_devel", goutils.IsGoRun()).
			Description("Is application in development mode").
			Flags(options.ReadOnly),
		options.NewOption("app.cli.binary_name", binaryName).
			Description("Application binary name").
			Flags(options.ReadOnly),
		options.NewOption("app.version", ver.String()).
			Description("Application version").
			Flags(options.ReadOnly),
		options.NewOption("app.module", module).
			Description("Application module").
			Flags(options.ReadOnly),
		options.NewOption("app.address", addr.String()).
			Description("Application address").
			Flags(options.ReadOnly),
		options.NewOption("app.pid", init.pid).
			Description("Application process id").
			Flags(options.ReadOnly),
		options.NewOption("app.instance.id", instance.NewID()).
			Description("Application instance id").
			Flags(options.ReadOnly),
		options.NewOption("app.fs.path.wd", "").
			Description("Current working directory").
			Flags(options.Mutable),
		options.NewOption("app.fs.path.home", "").
			Description("Current user home directory").
			Flags(options.Once),
		options.NewOption("app.fs.path.tmp", "").
			Description("Runtime tmp directory").
			Flags(options.Once),
		options.NewOption("app.fs.path.run", "").
			Description("Runtime directory").
			Flags(options.Once),
		options.NewOption("app.fs.path.profile.run", "").
			Description("Profile runtime directory").
			Flags(options.Once),
		options.NewOption("app.fs.path.cache", "").
			Description("Application shared cache directory").
			Flags(options.Once),
		options.NewOption("app.fs.path.profile.cache", "").
			Description("Application profile cache directory").
			Flags(options.Once),
		options.NewOption("app.fs.path.data", "").
			Description("Applciation shared persistent data directory").
			Flags(options.Once),
		options.NewOption("app.fs.path.profile.data", "").
			Description("Profile-specific persistent data").
			Flags(options.Once),
		options.NewOption("app.fs.path.state", "").
			Description("Applciation shared state data").
			Flags(options.Once),
		options.NewOption("app.fs.path.profile.state", "").
			Description("Profile-specific state, e.g., logs").
			Flags(options.Once),
		options.NewOption("app.fs.path.logs", "").
			Description("Shared logs").
			Flags(options.Once),
		options.NewOption("app.fs.path.profile.logs", "").
			Description("Profile-specific logs").
			Flags(options.Once),
		options.NewOption("app.fs.path.config", "").
			Description("Application configuration directory").
			Flags(options.Once),
		options.NewOption("app.fs.path.pids", "").
			Description("Application pids directory").
			Flags(options.Once),
		options.NewOption("app.fs.path.profile.config", "").
			Description("Base directory of loaded profile").
			Flags(options.Once),
		options.NewOption("app.main.exec.x", "").
			Description("-x flag is set to print all commands as executed").
			Flags(options.Once),
		options.NewOption("app.profile.name", init.defaults.configDefaultProfile).
			Description("Name of current settings profile").
			Flags(options.Once),
		options.NewOption("app.firstrun", false).
			Description("Application first run detected").
			Flags(options.Once),
		options.NewOption("app.os.user.uid", os.Geteuid()).
			Description("Geteuid returns the numeric effective user id of the caller running the app").
			Flags(options.ReadOnly),
		options.NewOption("app.os.user.gid", os.Getegid()).
			Description("Getegid returns the numeric effective group id of the caller running the app").
			Flags(options.ReadOnly),
	}

	init.opts, err = options.New("app", optSpecs...)
	return err
}

func (init *Initializer) initBasePaths() error {
	internal.LogInitDepth(init.log, 1, "initializing base paths")
	if len(init.defaults.slug) == 0 {
		return fmt.Errorf("%w: can not configure paths slug is empty", Error)
	}

	// User home
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := init.opts.Set("app.fs.path.home", userHomeDir); err != nil {
		return err
	}

	// Working directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := init.opts.Set("app.fs.path.wd", wd); err != nil {
		return err
	}

	instanceID := init.opts.Get("app.instance.id").String()

	tempDir := filepath.Join(os.TempDir(), init.defaults.slug, fmt.Sprintf("instance-%s", instanceID))
	if err := init.utilMkdir("create tmp directory", tempDir, 0700); err != nil {
		return err
	}
	if err := init.opts.Set("app.fs.path.tmp", tempDir); err != nil {
		return err
	}

	if init.defaults.configDisabled {
		pidsDir := filepath.Join(tempDir, "pids")
		_, err = os.Stat(pidsDir)
		if errors.Is(err, fs.ErrNotExist) {
			if err := init.utilMkdir("create tmp pids dir", pidsDir, 0700); err != nil {
				return err
			}
		}
		if err := init.opts.Set("app.fs.path.pids", pidsDir); err != nil {
			return err
		}
	}

	// config dir
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	var appConfigDir string
	if testing.Testing() {
		appConfigDir = filepath.Join(init.opts.Get("app.fs.path.tmp").String(), "config")
	} else {
		appConfigDir = filepath.Join(userConfigDir, init.defaults.slug)
	}

	_, err = os.Stat(appConfigDir)
	if errors.Is(err, fs.ErrNotExist) && !init.defaults.configDisabled {
		if err := init.utilMkdir("create config dir", appConfigDir, 0700); err != nil {
			return err
		}
	}

	// Runtime directory
	runDir := fsutils.RuntimeDir(init.defaults.slug)
	if err := init.opts.Set("app.fs.path.run", runDir); err != nil {
		return err
	}

	// Data directory
	dataDir := fsutils.DataDir(init.defaults.slug)
	if err := init.opts.Set("app.fs.path.data", dataDir); err != nil {
		return err
	}

	// State directory
	stateDir := fsutils.StateDir(init.defaults.slug)
	if err := init.opts.Set("app.fs.path.state", stateDir); err != nil {
		return err
	}

	// Logs directory
	if err := init.opts.Set("app.fs.path.logs", filepath.Join(stateDir, "logs")); err != nil {
		return err
	}

	// User cache directory
	var userCacheDir string
	if testing.Testing() {
		userCacheDir = filepath.Join(init.opts.Get("app.fs.path.tmp").String(), "cache")
	} else {
		userCacheDir, err = os.UserCacheDir()
		if err != nil {
			return fmt.Errorf("%w: failed to get user cache dir %s", Error, err)
		}
		userCacheDir = filepath.Join(userCacheDir, init.defaults.slug)
	}
	if err := init.opts.Set("app.fs.path.cache", userCacheDir); err != nil {
		return err
	}

	if !init.defaults.configDisabled {
		if err := init.opts.Set("app.fs.path.config", appConfigDir); err != nil {
			return err
		}
		pidsDir := filepath.Join(stateDir, "pids")
		_, err = os.Stat(pidsDir)
		if errors.Is(err, fs.ErrNotExist) {
			if err := init.utilMkdir("create pids dir", pidsDir, 0700); err != nil {
				return err
			}
		}

		if err := init.opts.Set("app.fs.path.pids", pidsDir); err != nil {
			return err
		}

		// Define default profile to load
		deafaultProfileFile := filepath.Join(appConfigDir, ".default.profile")
		if _, err = os.Stat(deafaultProfileFile); err == nil {
			deafaultProfileData, err := os.ReadFile(deafaultProfileFile)
			if err != nil {
				return fmt.Errorf("failed to read default profile file: %w", err)
			}
			profileName := strings.TrimSpace(string(deafaultProfileData))
			if len(profileName) == 0 {
				return fmt.Errorf("default profile file is empty")
			}
			if err := init.settingsb.SetDefaultFromString("app.default_profile", profileName); err != nil {
				return err
			}
			if err := init.opts.Set("app.profile.name", profileName); err != nil {
				return fmt.Errorf("%w: unable to update default profile name %s %s", Error, profileName, err.Error())
			}
		}
	}

	// Add an exit function to delete tmp dir
	init.rt.WithExitFunc(func(sess *session.Context, code int) error {
		if tempDir == "" {
			return fmt.Errorf("%w: missing temp dir path", Error)
		}
		if _, err := os.Stat(tempDir); err == nil {
			if err := os.RemoveAll(tempDir); err != nil {
				return fmt.Errorf("failed to delete temp dir %s: %w", tempDir, err)
			}
			if sess != nil {
				internal.Log(sess.Log(), "successfully deleted temp dir", slog.String("dir", tempDir))
			}
		}

		return nil
	})

	return nil
}

func (init *Initializer) initRootCommand() error {
	internal.LogInitDepth(init.log, 1, "initializing root command", slog.String("bin", init.opts.Get("app.cli.binary_name").String()))

	// Normalize os.Args
	var osargs []string
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			continue
		}
		osargs = append(osargs, arg)
	}

	binName := init.opts.Get("app.cli.binary_name").String()
	if binName == "" {
		return fmt.Errorf("%w: unable to determine bin name", Error)
	}

	osargs[0] = binName
	os.Args = osargs

	// Create root command
	root := command.New(binName,
		command.Config{
			MinArgs: settings.Uint(init.defaults.cliMainMinArgs),
			MaxArgs: settings.Uint(init.defaults.cliMainMaxArgs),
		})

	if init.defaults.cliWithGlobalFlags {
		root.WithFlags(
			cli.FlagVersion,
			cli.FlagHelp,
			cli.FlagX,
			cli.FlagSystemDebug,
			cli.FlagDebug,
			cli.FlagVerbose,
		)
	}

	if !init.defaults.configDisabled &&
		(init.defaults.configAllowCustomProfiles || len(init.defaults.configAdditionalProfiles) > 0) {
		root.WithFlags(varflag.StringFunc("profile", init.defaults.configDefaultProfile, "session profile to be used"))
	}

	if !init.defaults.cliWithGlobalFlags && init.defaults.develAllowProd {
		if isDevel, err := init.opts.Get("app.is_devel").Value().Bool(); err != nil {
		} else if isDevel {
			root.WithFlags(cli.FlagXProd)
		}
	}

	if init.defaults.cliWithConfigCmd {
		root.WithSubCommands(config.Command(config.DefaultCommandConfig()))
	}

	init.main = root
	return nil
}
