// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package config

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

type CommandConfig struct {
	Name                 string
	Category             string
	Description          string
	Info                 string
	HideDefaultUsage     bool
	WithoutLsCommand     bool
	WithoutGetCommand    bool
	WithoutSetCommand    bool
	WithoutAddCommand    bool
	WithoutRemoveCommand bool
	WithoutOptsCommand   bool
	WithoutResetCommand  bool
	// Hide keys from listing. Hidden keys can still be interacted with other commands.
	HideKeys []string
	// Disable keys from any interactions, also hides the key from listing.
	DisableKeys []string
	// Secrets these values will be redacted in output
	// Config cmd get also will retrun redacted
	Secrets []string
	// SecretsPassword is the password used to display redacted secret value with config get
	SecretsPassword string
}

func DefaultCommandConfig() CommandConfig {
	return CommandConfig{
		Name:                 "config",
		Category:             "Configuration",
		Description:          "Manage and configure application settings",
		Info:                 "This command allows you to manage the application configuration settings and settings profiles.",
		HideDefaultUsage:     false,
		WithoutLsCommand:     false,
		WithoutGetCommand:    false,
		WithoutSetCommand:    false,
		WithoutAddCommand:    false,
		WithoutRemoveCommand: false,
		WithoutResetCommand:  false,
	}
}

func Command(cnf CommandConfig) *command.Command {

	cmd := command.New(cnf.Name,
		command.Config{
			Category:         settings.String(cnf.Category),
			Description:      settings.String(cnf.Description),
			Immediate:        true,
			SkipSharedBefore: true,
		})

	cmd.AddInfo(cnf.Info)

	var subcmds []*command.Command
	if !cnf.WithoutLsCommand {
		subcmds = append(subcmds, configLs(cnf.HideKeys, cnf.DisableKeys, cnf.Secrets))
	}
	if !cnf.WithoutGetCommand {
		subcmds = append(subcmds, configGet(cnf.DisableKeys, cnf.Secrets, cnf.SecretsPassword))
	}
	if !cnf.WithoutSetCommand {
		subcmds = append(subcmds, configSet(cnf.DisableKeys))
	}
	if !cnf.WithoutAddCommand {
		subcmds = append(subcmds, configAdd(cnf.DisableKeys))
	}
	if !cnf.WithoutRemoveCommand {
		subcmds = append(subcmds, configRemove(cnf.DisableKeys))
	}
	if !cnf.WithoutResetCommand {
		subcmds = append(subcmds, configReset())
	}

	cmd.WithSubCommands(subcmds...)

	return cmd
}

func configLs(hiddenKeys, disabledKeys, secrets []string) *command.Command {
	cmd := command.New("ls",
		command.Config{
			Description: "List settings for current profile",
			Usage:       "[-a|--all]",
		})

	cmd.AddUsage("--profile=<profile-name> [flags]")

	cmd.WithFlags(
		varflag.BoolFunc("all", false, "List all settings (default current profile), including internal settings", "a"),
		varflag.BoolFunc("describe", false, "Describe settings", "d"),
		varflag.StringFunc("prefix", "", "Display settings with a specific prefix", "p"),
	)

	cmd.Do(func(sess *session.Context, args action.Args) error {
		type appConfigRow struct {
			Key,
			Value,
			Default,
			Desc,
			Kind,
			Mutability,
			IsSet string
		}
		var (
			appConfig      []appConfigRow
			profileConfig  []appConfigRow
			onlyWithPrefix = args.Flag("prefix").Present()
			withPrefix     = args.Flag("prefix").String()
		)

		describeFlagTrue := args.Flag("describe").Var().Bool()
		for s := range sess.Settings().All() {
			if (slices.Contains(disabledKeys, s.Key())) ||
				(!describeFlagTrue && slices.Contains(hiddenKeys, s.Key()) ||
					onlyWithPrefix && !strings.HasPrefix(s.Key(), withPrefix)) {
				continue
			}

			var (
				defval string
				value  = s.Value().String()
			)
			if s.Default().String() != s.Value().String() {
				defval = s.Default().String()
			}
			if slices.Contains(secrets, s.Key()) {
				defval = ""
				value = "<redacted>"
			}

			row := appConfigRow{
				Key:        s.Key(),
				Value:      value,
				Default:    defval,
				Desc:       sess.Describe(s.Key()),
				Kind:       s.Kind().String(),
				IsSet:      fmt.Sprint(s.IsSet()),
				Mutability: s.Mutability().String(),
			}

			if s.Persistent() {
				profileConfig = append(profileConfig, row)
				continue
			}
			appConfig = append(appConfig, row)
		}

		sess.Opts().Range(func(opt options.Option) bool {
			if slices.Contains(hiddenKeys, opt.Key()) ||
				slices.Contains(disabledKeys, opt.Key()) ||
				(onlyWithPrefix && !strings.HasPrefix(opt.Key(), withPrefix)) {
				return true
			}

			value := opt.Value().String()
			if slices.Contains(secrets, opt.Key()) {
				value = "<redacted>"
			}
			row := appConfigRow{
				Key:   opt.Key(),
				Kind:  opt.Default().Kind().String(),
				Value: value,
				Desc:  opt.Description(),
			}
			appConfig = append(appConfig, row)
			return true
		})

		sort.Slice(appConfig, func(i, j int) bool {
			return appConfig[i].Key < appConfig[j].Key
		})

		// Descriptions
		if describeFlagTrue {
			desctable := textfmt.NewTable(
				textfmt.TableTitle("Settings Descriptions"),
				textfmt.TableWithHeader(),
			)
			desctable.AddRow("KEY", "DESCRIPTION")

			descbatch := textfmt.NewTableBatchOp()
			for _, c := range profileConfig {
				descbatch.AddRow(c.Key, c.Desc)
			}
			if args.Flag("all").Var().Bool() {
				for _, c := range appConfig {
					descbatch.AddRow(c.Key, c.Desc)
				}
			}
			desctable.Batch(descbatch)

			fmt.Println(desctable.String())
			return nil
		}

		configTable := textfmt.NewTable(
			textfmt.TableTitle(fmt.Sprintf("Configuration of %s", sess.Get("app.name").String())),
		)
		// Profile settings

		profileTable := textfmt.NewTable(
			textfmt.TableTitle(fmt.Sprintf("Settings for current PROFILE: %s", sess.Settings().Name())),
			textfmt.TableWithHeader(),
		)
		profileTable.AddRow("KEY", "KIND", "IS SET", "MUTABILITY", "VALUE", "DEFAULT")
		profileBatch := textfmt.NewTableBatchOp()
		for _, c := range profileConfig {
			if c.Kind == "slice" && c.Value != "" {
				for _, v := range sess.Get(c.Key).Fields() {
					profileBatch.AddRow(c.Key+"[]", c.Kind+"(string)", c.IsSet, c.Mutability, v, c.Default)
				}
				continue
			}
			profileBatch.AddRow(c.Key, c.Kind, c.IsSet, c.Mutability, c.Value, c.Default)
		}
		profileTable.Batch(profileBatch)

		// App settings
		if !args.Flag("all").Var().Bool() {
			configTable.Append(profileTable)
			fmt.Println(configTable.String())
			return nil
		}

		appTable := textfmt.NewTable(
			textfmt.TableTitle("Application Config"),
			textfmt.TableWithHeader(),
		)
		appTable.AddRow("KEY", "KIND", "VALUE")
		appBatch := textfmt.NewTableBatchOp()
		for _, c := range appConfig {
			appBatch.AddRow(c.Key, c.Kind, c.Value)
		}
		appTable.Batch(appBatch)

		configTable.Append(appTable)
		configTable.Append(profileTable)

		fmt.Println(configTable.String())

		return nil
	})

	return cmd
}

func configSet(disabledKeys []string) *command.Command {
	cmd := command.New("set",
		command.Config{
			Description: "Set a setting value",
			MinArgs:     2,
		})

	cmd.AddUsage("--profile=<profile-name>")

	cmd.Do(func(sess *session.Context, args action.Args) error {
		key := args.Arg(0).String()
		if slices.Contains(disabledKeys, key) || !sess.Settings().Has(key) {
			return fmt.Errorf("setting %q does not exist", key)
		}
		value := args.Arg(1).String()

		if err := sess.Settings().ValidatePreference(key, value); err != nil {
			return err
		}

		prefs := sess.Settings().Preferences()
		prefs.Set(key, value)

		return savePreferences(sess, prefs)
	})

	return cmd
}

func configAdd(disabledKeys []string) *command.Command {
	cmd := command.New("add",
		command.Config{
			Description: "Add given value to setting value",
			MinArgs:     2,
		})

	cmd.AddUsage("--profile=<profile-name>")

	cmd.Do(func(sess *session.Context, args action.Args) error {
		key := args.Arg(0).String()
		if slices.Contains(disabledKeys, key) || !sess.Settings().Has(key) {
			return fmt.Errorf("setting %q does not exist", key)
		}
		value := args.Arg(1).String()

		if err := sess.Settings().ValidatePreference(key, value); err != nil {
			return err
		}

		curr := sess.Settings().Get(key)
		if curr.Kind() != settings.KindStringSlice {
			return fmt.Errorf("setting %q is not a slice", key)
		}
		values := curr.Value().Fields()
		if slices.Contains(values, value) {
			return fmt.Errorf("%q value %q already exists", key, value)
		}
		values = append(values, value)

		prefs := sess.Settings().Preferences()
		prefs.Set(key, strings.Join(values, "\x1f"))

		return savePreferences(sess, prefs)
	})

	return cmd
}

func configRemove(disabledKeys []string) *command.Command {
	cmd := command.New("remove",
		command.Config{
			Description: "Remove given value from setting value",
			MinArgs:     2,
		})

	cmd.AddUsage("--profile=<profile-name>")

	cmd.Do(func(sess *session.Context, args action.Args) error {
		key := args.Arg(0).String()
		if slices.Contains(disabledKeys, key) || !sess.Settings().Has(key) {
			return fmt.Errorf("setting %q does not exist", key)
		}
		value := args.Arg(1).String()

		curr := sess.Settings().Get(key)
		if curr.Kind() != settings.KindStringSlice {
			return fmt.Errorf("setting %q is not a slice", key)
		}
		oldValues := curr.Value().Fields()
		if !slices.Contains(oldValues, value) {
			return fmt.Errorf("%q value %q does not exist", key, value)
		}
		values := slices.DeleteFunc(oldValues, func(v string) bool {
			return v == value
		})

		prefs := sess.Settings().Preferences()
		prefs.Set(key, strings.Join(values, "\x1f"))

		return savePreferences(sess, prefs)
	})

	return cmd
}

func configGet(disabledKeys, secrets []string, secretsPassword string) *command.Command {
	cmd := command.New("get",
		command.Config{
			Description: "Get a setting or option value",
			MinArgs:     1,
		})

	cmd.AddUsage("--profile=<profile-name>")

	cmd.WithFlags(
		cli.NewBoolFlag("secret", false, "Prompts to enter Secrets password to display value. If password is not set it prints secret value instead <redacted> without the prompt"),
	)
	cmd.Do(func(sess *session.Context, args action.Args) error {
		key := args.Arg(0).String()
		if slices.Contains(disabledKeys, key) || !sess.Has(key) {
			return fmt.Errorf("setting %q does not exist", key)
		}
		if !slices.Contains(secrets, key) {
			fmt.Println(sess.Get(key).String())
			return nil
		}
		var canshow bool
		if args.Flag("secret").Present() {
			if secretsPassword == "" {
				canshow = true
			} else {
				if str := cli.AskForSecret("enter secrets password"); str != secretsPassword {
					return fmt.Errorf("invalid password")
				}
				canshow = true
			}
		}
		if !canshow {
			fmt.Println("<redacted>")
			return nil
		}
		fmt.Println(sess.Get(key).String())
		return nil
	})

	return cmd
}

func configReset() *command.Command {
	cmd := command.New("reset",
		command.Config{
			Description: "Reset a setting to its default value",
			MinArgs:     1,
		})

	cmd.AddUsage("--profile=<profile-name>")

	cmd.WithFlags(varflag.BoolFunc("all", false, "reset all settings", "a"))

	cmd.Do(func(sess *session.Context, args action.Args) error {
		if args.Flag("all").Present() {
			profileFilePath := filepath.Join(sess.Get("app.fs.path.profile.config").String(), "profile.preferences")
			sess.Log().Log(sess.Context(), logging.LevelHappy.Level(),
				"profile.save",
				slog.String("profile", sess.Get("app.profile.name").String()),
				slog.String("file", profileFilePath),
			)

			if err := os.WriteFile(profileFilePath, []byte{}, 0600); err != nil {
				return err
			}

			sess.Log().Log(sess.Context(), logging.LevelHappy.Level(),
				"saved profile",
				slog.String("profile", sess.Get("app.profile.name").String()),
				slog.String("file", profileFilePath),
			)
			return nil
		}

		key := args.Arg(0).String()
		if !sess.Settings().Has(key) {
			return fmt.Errorf("setting %q does not exist", key)
		}

		profileFilePath := filepath.Join(sess.Get("app.fs.path.profile.config").String(), "profile.preferences")
		sess.Log().Log(sess.Context(), logging.LevelHappy.Level(),
			"profile.save",
			slog.String("profile", sess.Get("app.profile.name").String()),
			slog.String("file", profileFilePath),
		)

		pd := vars.Map{}
		for setting := range sess.Settings().All() {
			if setting.Persistent() {
				if setting.Key() == key {
					continue
				} else if setting.IsSet() {
					if err := pd.Store(setting.Key(), setting.Value().String()); err != nil {
						return err
					}
				}
			}
		}
		pddata := pd.ToKeyValSlice()
		var dest bytes.Buffer
		enc := gob.NewEncoder(&dest)
		if err := enc.Encode(pddata); err != nil {
			return err
		}

		if err := os.WriteFile(profileFilePath, dest.Bytes(), 0600); err != nil {
			return err
		}

		sess.Log().Log(sess.Context(), logging.LevelHappy.Level(),
			"saved profile",
			slog.String("profile", sess.Get("app.profile.name").String()),
			slog.String("file", profileFilePath),
		)
		return nil
	})

	return cmd
}

func savePreferences(sess *session.Context, prefs *settings.Preferences) error {
	profileFilePath := filepath.Join(sess.Get("app.fs.path.profile.config").String(), "profile.preferences")

	var dest bytes.Buffer

	enc := gob.NewEncoder(&dest)

	if err := enc.Encode(prefs); err != nil {
		return err
	}

	sess.Log().Log(sess.Context(), logging.LevelHappy.Level(),
		"profile.save",
		slog.String("profile", sess.Get("app.profile.name").String()),
		slog.String("file", profileFilePath),
	)
	if err := os.WriteFile(profileFilePath, dest.Bytes(), 0600); err != nil {
		return err
	}

	sess.Log().Log(sess.Context(), logging.LevelHappy.Level(),
		"saved profile",
		slog.String("profile", sess.Get("app.profile.name").String()),
		slog.String("file", profileFilePath),
	)
	return nil
}
