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
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/app/session"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/config"
	"github.com/happy-sdk/happy/sdk/devel"
	"github.com/happy-sdk/happy/sdk/instance"
	"github.com/happy-sdk/happy/sdk/internal"
	"github.com/happy-sdk/happy/sdk/networking/address"
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
	binName                   string
	cliMainMinArgs            uint
	cliMainMaxArgs            uint
	cliWithoutConfigCmd       bool
	cliWithoutGlobalFlags     bool
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

	internal.LogInit(init.log, init.defaults.slug,
		slog.String("version", init.opts.Get("app.version").String()),
		slog.Int("pid", init.pid))

	// Set the application process id
	if err := init.opts.Set("app.pid", init.pid); err != nil {
		init.error(err)
		return
	}

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

	// Load settings blueprint
	if init.settingsb, err = init.settings.Blueprint(); err != nil {
		init.error(err)
		return
	}

	// Load defaults before profile is loaded
	configDisabledSpec, err := init.settingsb.GetSpec("app.config.disabled")
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
	configDefaultProfileSpec, err := init.settingsb.GetSpec("app.config.default_profile")
	if err != nil {
		return err
	}
	configAdditionalProfilesSpec, err := init.settingsb.GetSpec("app.config.additional_profiles")
	if err != nil {
		return err
	}
	configAllowCustomProfilesSpec, err := init.settingsb.GetSpec("app.config.allow_custom_profiles")
	if err != nil {
		return err
	}
	configEnableProfileDevelSpec, err := init.settingsb.GetSpec("app.config.enable_profile_devel")
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
	cliWithoutConfigCmdSpec, err := init.settingsb.GetSpec("app.cli.without_config_cmd")
	if err != nil {
		return err
	}
	cliWithoutGlobalFlagsSpec, err := init.settingsb.GetSpec("app.cli.without_global_flags")
	if err != nil {
		return err
	}
	develAllowProdSpec, err := init.settingsb.GetSpec("app.devel.allow_prod")
	if err != nil {
		return err
	}
	binNameSpec, err := init.settingsb.GetSpec("app.cli.name")
	if err != nil {
		return err
	}

	init.defaults.configDisabled = configDisabledSpec.Value == "true"
	init.defaults.slug = slugSpec.Value
	init.defaults.binName = binNameSpec.Value
	init.defaults.identifier = identifierSpec.Value
	init.defaults.cliMainMinArgs = uint(cliMainMinArgs)
	init.defaults.cliMainMaxArgs = uint(cliMainMaxArgs)
	init.defaults.cliWithoutConfigCmd = cliWithoutConfigCmdSpec.Value == "true"
	init.defaults.cliWithoutGlobalFlags = cliWithoutGlobalFlagsSpec.Value == "true"
	init.defaults.develAllowProd = develAllowProdSpec.Value == "true"

	if init.defaults.configDisabled {
		init.defaults.configDefaultProfile = configDefaultProfileSpec.Default
		init.defaults.configAdditionalProfiles = strings.Split(configAdditionalProfilesSpec.Default, "|")
	} else {
		init.defaults.configDefaultProfile = configDefaultProfileSpec.Value
		init.defaults.configAdditionalProfiles = strings.Split(configAdditionalProfilesSpec.Value, "|")
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
			init.defaults.slug = addr.Instance()
		}
		if err := init.settingsb.SetDefault("app.slug", init.defaults.slug); err != nil {
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

	if err := init.settingsb.SetDefault("app.identifier", init.defaults.identifier); err != nil {
		return err
	}

	if err := slugSpec.ValidateValue(init.defaults.slug); err != nil {
		return err
	}

	// Set binName from valid slug if not set.
	if init.defaults.binName == "" {
		init.defaults.binName = init.defaults.slug
	}

	if err := init.settingsb.SetDefault("app.copyright_since", fmt.Sprint(time.Now().Year())); err != nil {
		return err
	}

	optSpecs := []options.Spec{
		options.NewOption(
			"app.is_devel",
			version.IsDev(ver.String()),
			"Is application in development mode",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.version",
			ver.String(),
			"Application version",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.wd",
			"",
			"Current working directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.home",
			"",
			"Current user home directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.tmp",
			"",
			"Runtime tmp directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.cache",
			"",
			"Application cache directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.config",
			"",
			"Application configuration directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.pids",
			"",
			"Application pids directory",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.fs.path.profile",
			"",
			"Base directory of loaded profile",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.main.exec.x",
			"",
			"-x flag is set to print all commands as executed",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.profile.name",
			init.defaults.configDefaultProfile,
			"Name of current settings profile",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.dosetup",
			false,
			"Application setup will be executed if true",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.module",
			module,
			"Application module",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.address",
			addr.String(),
			"Application address",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.pid",
			0,
			"Application process id",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.instance.id",
			"xxxxxxxx",
			"Application instance id",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.os.user.uid",
			os.Geteuid(),
			"Geteuid returns the numeric effective user id of the caller running the app",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
		options.NewOption(
			"app.os.user.gid",
			os.Getegid(),
			"Getegid returns the numeric effective group id of the caller running the app",
			options.KindConfig|options.KindReadOnly,
			options.NoopValueValidator,
		),
	}

	init.opts, err = options.New("app", optSpecs)
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

	instanceID := instance.NewID()
	if err := init.opts.Set("app.instance.id", instanceID); err != nil {
		return err
	}

	tempDir := filepath.Join(os.TempDir(), init.defaults.slug, fmt.Sprintf("instance-%s", instanceID))
	if err := init.utilMkdir("create tmp directory", tempDir, 0700); err != nil {
		return err
	}
	if err := init.opts.Set("app.fs.path.tmp", tempDir); err != nil {
		return err
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
	if errors.Is(err, fs.ErrNotExist) {
		if err := init.utilMkdir("create config dir", appConfigDir, 0700); err != nil {
			return err
		}
		if err := init.opts.Set("app.dosetup", true); err != nil {
			return err
		}
	}

	if err := init.opts.Set("app.fs.path.config", appConfigDir); err != nil {
		return err
	}

	pidsDir := filepath.Join(appConfigDir, "pids")
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
		if err := init.settingsb.SetDefault("app.default_profile", profileName); err != nil {
			return err
		}
		if err := init.opts.Set("app.profile.name", profileName); err != nil {
			return fmt.Errorf("%w: unable to update default profile name %s %s", Error, profileName, err.Error())
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
				internal.Log(sess.Log(), "successfully deleted temp dir", slog.String("tempdir", tempDir))
			}
		}
		return nil
	})

	return nil
}

func (init *Initializer) initRootCommand() error {
	internal.LogInitDepth(init.log, 1, "initializing root command", slog.String("bin", init.defaults.binName))

	// Normalize os.Args
	var osargs []string
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			continue
		}
		osargs = append(osargs, arg)
	}

	if init.defaults.binName == "" {
		return fmt.Errorf("%w: unable to detemone bin name", Error)
	}

	osargs[0] = init.defaults.binName
	os.Args = osargs

	// Create root command
	root := command.New(command.Config{
		Name:    settings.String(init.defaults.binName),
		MinArgs: settings.Uint(init.defaults.cliMainMinArgs),
		MaxArgs: settings.Uint(init.defaults.cliMainMaxArgs),
	})

	if !init.defaults.cliWithoutGlobalFlags {
		root.WithFlags(
			cli.FlagVersion,
			cli.FlagHelp,
			cli.FlagX,
			cli.FlagSystemDebug,
			cli.FlagDebug,
			cli.FlagVerbose,
		)

		if !init.defaults.configDisabled {
			root.WithFlags(varflag.StringFunc("profile", init.defaults.configDefaultProfile, "session profile to be used"))
		}

	}

	if !init.defaults.cliWithoutGlobalFlags && init.defaults.develAllowProd {
		if init.opts.Get("app.is_devel").Bool() {
			root.WithFlags(devel.FlagXProd)
		}
	}

	if !init.defaults.cliWithoutConfigCmd {
		root.WithSubCommands(config.Command())
	}

	init.main = root
	return nil
}
