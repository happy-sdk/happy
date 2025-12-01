// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

const i18np = "com.github.happy-sdk.happy.sdk.cli.cmd.config"

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
		Category:             i18np + ".category",
		Description:          i18np + ".description",
		Info:                 i18np + ".info",
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

	// Translate info on-the-fly when displaying, so store i18n key
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
			Description: settings.String(i18np + ".ls.description"),
			Usage:       i18np + ".ls.usage_flags",
		})

	cmd.AddUsage(i18np + ".ls.usage")

	cmd.WithFlags(
		varflag.BoolFunc("all", false, i18np+".ls.flag_all", "a"),
		varflag.BoolFunc("describe", false, i18np+".ls.flag_describe", "d"),
		varflag.StringFunc("prefix", "", i18np+".ls.flag_prefix", "p"),
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
				value  = s.Display() // Use Display() to get translated value
			)
			if s.Default().String() != s.Value().String() {
				defval = s.Default().Display() // Use Display() for default too
			}
			if slices.Contains(secrets, s.Key()) {
				defval = ""
				value = i18n.T(i18np + ".get.redacted")
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
				textfmt.TableTitle(i18n.T(i18np+".ls.table_settings_descriptions")),
				textfmt.TableWithHeader(),
			)
			desctable.AddRow(
				i18n.T(i18np+".ls.header_key"),
				i18n.T(i18np+".ls.header_description"),
			)

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

		appNameSetting := sess.Settings().Get("app.name")
		appName := appNameSetting.Display()
		configTable := textfmt.NewTable(
			textfmt.TableTitle(fmt.Sprintf(i18n.T(i18np+".ls.table_configuration_of"), appName)),
		)
		// Profile settings

		profileTable := textfmt.NewTable(
			textfmt.TableTitle(fmt.Sprintf(i18n.T(i18np+".ls.table_settings_for_profile"), sess.Settings().Name())),
			textfmt.TableWithHeader(),
		)
		profileTable.AddRow(
			i18n.T(i18np+".ls.header_key"),
			i18n.T(i18np+".ls.header_kind"),
			i18n.T(i18np+".ls.header_is_set"),
			i18n.T(i18np+".ls.header_mutability"),
			i18n.T(i18np+".ls.header_value"),
			i18n.T(i18np+".ls.header_default"),
		)
		profileBatch := textfmt.NewTableBatchOp()
		for _, c := range profileConfig {
			if c.Kind == "slice" && c.Value != "" {
				setting := sess.Settings().Get(c.Key)
				for _, v := range setting.Value().Fields() {
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
			textfmt.TableTitle(i18n.T(i18np+".ls.table_application_config")),
			textfmt.TableWithHeader(),
		)
		appTable.AddRow(
			i18n.T(i18np+".ls.header_key"),
			i18n.T(i18np+".ls.header_kind"),
			i18n.T(i18np+".ls.header_value"),
		)
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
			Description: settings.String(i18np + ".set.description"),
			MinArgs:     2,
		})

	cmd.AddUsage(i18np + ".usage")

	cmd.Do(func(sess *session.Context, args action.Args) error {
		key := args.Arg(0).String()
		if slices.Contains(disabledKeys, key) || !sess.Settings().Has(key) {
			return fmt.Errorf(i18n.T(i18np+".set.error_not_exists"), key)
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
			Description: settings.String(i18np + ".add.description"),
			MinArgs:     2,
		})

	cmd.AddUsage(i18np + ".usage")

	cmd.Do(func(sess *session.Context, args action.Args) error {
		key := args.Arg(0).String()
		if slices.Contains(disabledKeys, key) || !sess.Settings().Has(key) {
			return fmt.Errorf(i18n.T(i18np+".add.error_not_exists"), key)
		}
		value := args.Arg(1).String()

		if err := sess.Settings().ValidatePreference(key, value); err != nil {
			return err
		}

		curr := sess.Settings().Get(key)
		if curr.Kind() != settings.KindStringSlice {
			return fmt.Errorf(i18n.T(i18np+".add.error_not_slice"), key)
		}
		values := curr.Value().Fields()
		if slices.Contains(values, value) {
			return fmt.Errorf(i18n.T(i18np+".add.error_already_exists"), key, value)
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
			Description: settings.String(i18np + ".remove.description"),
			MinArgs:     2,
		})

	cmd.AddUsage(i18np + ".usage")

	cmd.Do(func(sess *session.Context, args action.Args) error {
		key := args.Arg(0).String()
		if slices.Contains(disabledKeys, key) || !sess.Settings().Has(key) {
			return fmt.Errorf(i18n.T(i18np+".remove.error_not_exists"), key)
		}
		value := args.Arg(1).String()

		curr := sess.Settings().Get(key)
		if curr.Kind() != settings.KindStringSlice {
			return fmt.Errorf(i18n.T(i18np+".remove.error_not_slice"), key)
		}
		oldValues := curr.Value().Fields()
		if !slices.Contains(oldValues, value) {
			return fmt.Errorf(i18n.T(i18np+".remove.error_value_not_exists"), key, value)
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
			Description: settings.String(i18np + ".get.description"),
			MinArgs:     1,
		})

	cmd.AddUsage(i18np + ".usage")

	cmd.WithFlags(
		cli.NewBoolFlag("secret", false, i18np+".get.flag_secret"),
	)
	cmd.Do(func(sess *session.Context, args action.Args) error {
		key := args.Arg(0).String()
		if slices.Contains(disabledKeys, key) || !sess.Has(key) {
			return fmt.Errorf(i18n.T(i18np+".get.error_not_exists"), key)
		}
		if !slices.Contains(secrets, key) {
			// Use Display() to get translated value if i18n is enabled
			if sess.Settings().Has(key) {
				setting := sess.Settings().Get(key)
				fmt.Println(setting.Display())
			} else {
				fmt.Println(sess.Get(key).String())
			}
			return nil
		}
		var canshow bool
		if args.Flag("secret").Present() {
			if secretsPassword == "" {
				canshow = true
			} else {
				if str := cli.AskForSecret(i18n.T(i18np + ".get.prompt_secrets_password")); str != secretsPassword {
					return errors.New(i18n.T(i18np + ".get.error_invalid_password"))
				}
				canshow = true
			}
		}
		if !canshow {
			fmt.Println(i18n.T(i18np + ".get.redacted"))
			return nil
		}
		if sess.Settings().Has(key) {
			setting := sess.Settings().Get(key)
			fmt.Println(setting.Display())
		} else {
			fmt.Println(sess.Get(key).String())
		}
		return nil
	})

	return cmd
}

func configReset() *command.Command {
	cmd := command.New("reset",
		command.Config{
			Description: settings.String(i18np + ".reset.description"),
			MinArgs:     1,
		})

	cmd.AddUsage(i18np + ".usage")

	cmd.WithFlags(varflag.BoolFunc("all", false, i18np+".reset.flag_all", "a"))

	cmd.Do(func(sess *session.Context, args action.Args) error {
		if args.Flag("all").Present() {
			// Create empty preferences with correct version
			prefs := sess.Settings().Preferences()
			// Clear all data but keep version
			emptyPrefs := settings.NewPreferences(prefs.SchemaVersion())
			return savePreferences(sess, emptyPrefs)
		}

		key := args.Arg(0).String()
		if !sess.Settings().Has(key) {
			return fmt.Errorf(i18n.T(i18np+".reset.error_not_exists"), key)
		}

		// Get current preferences and remove the key
		prefs := sess.Settings().Preferences()
		// Create new preferences with same version but without the reset key
		newPrefs := settings.NewPreferences(prefs.SchemaVersion())
		for setting := range sess.Settings().All() {
			if setting.Persistent() && setting.IsSet() {
				if setting.Key() != key {
					newPrefs.Set(setting.Key(), setting.Value().String())
				}
			}
		}

		return savePreferences(sess, newPrefs)
	})

	return cmd
}

func savePreferences(sess *session.Context, prefs *settings.Preferences) error {
	profileFilePath := filepath.Join(sess.Get("app.fs.path.profile.config").String(), "profile.preferences")

	// Use GobEncode() method directly to ensure proper encoding
	prefData, err := prefs.GobEncode()
	if err != nil {
		return fmt.Errorf("failed to encode preferences: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(profileFilePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create preferences directory: %w", err)
	}

	sess.Log().Log(sess.Context(), logging.LevelHappy.Level(),
		"profile.save",
		slog.String("profile", sess.Get("app.profile.name").String()),
		slog.String("file", profileFilePath),
		slog.Int("data_size", len(prefData)),
	)

	// Write file atomically: write to temp file, then rename
	tempFile := profileFilePath + ".tmp"
	if err := os.WriteFile(tempFile, prefData, 0600); err != nil {
		return fmt.Errorf("failed to write preferences file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, profileFilePath); err != nil {
		_ = os.Remove(tempFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename preferences file: %w", err)
	}

	sess.Log().Log(sess.Context(), logging.LevelHappy.Level(),
		"saved profile",
		slog.String("profile", sess.Get("app.profile.name").String()),
		slog.String("file", profileFilePath),
	)
	return nil
}
