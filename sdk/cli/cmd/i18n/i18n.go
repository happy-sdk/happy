// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy/pkg/devel/goutils"
	"github.com/happy-sdk/happy/pkg/networking/address"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/mod/modfile"
	"golang.org/x/text/language"
)

const i18np = "com.github.happy-sdk.happy.sdk.cli.cmd.i18n"

type CommandConfig struct {
	Name             string
	Category         string
	Description      string
	Info             string
	WithoutReport    bool
	WithoutList      bool
	WithoutTranslate bool
}

func DefaultCommandConfig() CommandConfig {
	return CommandConfig{
		Name:             "i18n",
		Category:         i18np + ".category",
		Description:      i18np + ".description",
		Info:             i18np + ".info",
		WithoutReport:    false,
		WithoutList:      false,
		WithoutTranslate: false,
	}
}

func Command(cnf CommandConfig) *command.Command {
	cmd := command.New(cnf.Name,
		command.Config{
			Category:           settings.String(cnf.Category),
			Description:        settings.String(cnf.Description),
			Immediate:          true,
			SharedBeforeAction: true,
			FailDisabled:       true,
		})

	// Store i18n key for info text, which will be translated on-the-fly when displayed
	cmd.AddInfo(cnf.Info)

	// Disable command if not in a Go module matching app.module
	cmd.Disable(func(sess *session.Context) error {
		appModule := sess.Opts().Get("app.module").String()
		if appModule == "" {
			return fmt.Errorf("app.module is not set")
		}

		wd := sess.Opts().Get("app.fs.path.wd").String()
		if wd == "" {
			return fmt.Errorf("working directory is not set")
		}

		// Walk up the directory tree to find go.mod
		dir := wd
		for {
			gomodPath, found := goutils.ContainsGoModfile(dir)
			if !found {
				parent := filepath.Dir(dir)
				if parent == dir {
					// Reached root
					return fmt.Errorf("no go.mod file found in current directory or parent directories")
				}
				dir = parent
				continue
			}

			// Read and parse go.mod
			data, err := os.ReadFile(gomodPath)
			if err != nil {
				return fmt.Errorf("failed to read go.mod: %w", err)
			}

			modFile, err := modfile.Parse(gomodPath, data, nil)
			if err != nil {
				return fmt.Errorf("failed to parse go.mod: %w", err)
			}

			if modFile.Module == nil {
				return fmt.Errorf("go.mod does not contain module declaration")
			}

			// Check if module path matches app.module
			// app.module can be a subdirectory of the module path
			// e.g., module = "github.com/happy-sdk/banctl"
			// app.module = "github.com/happy-sdk/banctl/cmd/banctl"
			modulePath := modFile.Module.Mod.Path
			if modulePath == appModule || strings.HasPrefix(appModule, modulePath+"/") {
				// Found matching module, command is enabled
				return nil
			}

			// Module doesn't match, continue searching up
			parent := filepath.Dir(dir)
			if parent == dir {
				// Reached root
				return fmt.Errorf("no go.mod file found with module path matching app.module (%s)", appModule)
			}
			dir = parent
		}
	})

	var subcmds []*command.Command
	if !cnf.WithoutReport {
		subcmds = append(subcmds, i18nReport())
	}
	if !cnf.WithoutList {
		subcmds = append(subcmds, i18nList())
	}
	if !cnf.WithoutTranslate {
		subcmds = append(subcmds, i18nTranslate())
	}

	cmd.WithSubCommands(subcmds...)

	return cmd
}

// findModuleRoot finds the root directory of the Go module that matches app.module
func findModuleRoot(sess *session.Context) (string, error) {
	appModule := sess.Opts().Get("app.module").String()
	if appModule == "" {
		return "", fmt.Errorf("app.module is not set")
	}

	wd := sess.Opts().Get("app.fs.path.wd").String()
	if wd == "" {
		return "", fmt.Errorf("working directory is not set")
	}

	// Walk up the directory tree to find go.mod
	dir := wd
	for {
		gomodPath, found := goutils.ContainsGoModfile(dir)
		if !found {
			parent := filepath.Dir(dir)
			if parent == dir {
				return "", fmt.Errorf("no go.mod file found in current directory or parent directories")
			}
			dir = parent
			continue
		}

		// Read and parse go.mod
		data, err := os.ReadFile(gomodPath)
		if err != nil {
			return "", fmt.Errorf("failed to read go.mod: %w", err)
		}

		modFile, err := modfile.Parse(gomodPath, data, nil)
		if err != nil {
			return "", fmt.Errorf("failed to parse go.mod: %w", err)
		}

		if modFile.Module == nil {
			parent := filepath.Dir(dir)
			if parent == dir {
				return "", fmt.Errorf("go.mod does not contain module declaration")
			}
			dir = parent
			continue
		}

		// Check if module path matches app.module
		// app.module can be a subdirectory of the module path
		// e.g., module = "github.com/happy-sdk/banctl"
		// app.module = "github.com/happy-sdk/banctl/cmd/banctl"
		modulePath := modFile.Module.Mod.Path
		if modulePath == appModule || strings.HasPrefix(appModule, modulePath+"/") {
			// Found matching module root
			return dir, nil
		}

		// Module doesn't match, continue searching up
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod file found with module path matching app.module (%s)", appModule)
		}
		dir = parent
	}
}

// getAppModulePrefix converts the app.module setting to reverse DNS notation for filtering translation keys.
// It returns the module root identifier (not the full app.module path including subdirectories).
// Example: app.module = "github.com/happy-sdk/banctl/cmd/banctl"
//
//	returns "com.github.happy-sdk.banctl" (module root identifier)
func getAppModulePrefix(sess *session.Context) (string, error) {
	appModule := sess.Opts().Get("app.module").String()
	if appModule == "" {
		return "", fmt.Errorf("app.module is not set")
	}

	// Find the module root path (not the subdirectory)
	moduleRoot, err := findModuleRoot(sess)
	if err != nil {
		return "", fmt.Errorf("failed to find module root: %w", err)
	}

	// Read go.mod to get the actual module path
	gomodPath := filepath.Join(moduleRoot, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	modFile, err := modfile.Parse(gomodPath, data, nil)
	if err != nil {
		return "", fmt.Errorf("failed to parse go.mod: %w", err)
	}

	if modFile.Module == nil {
		return "", fmt.Errorf("go.mod does not contain module declaration")
	}

	// Extract the module path from go.mod (not the app.module subdirectory)
	modulePath := modFile.Module.Mod.Path

	// Convert module path to reverse DNS identifier format using address package
	// The "dummy" hostname is used since we only need the reverse DNS conversion
	addr, err := address.FromModule("dummy", modulePath)
	if err != nil {
		return "", fmt.Errorf("failed to convert module to address: %w", err)
	}

	return addr.ReverseDNS(), nil
}

// getDependencyIdentifiers reads go.mod and returns a map of all dependency module identifiers
// in reverse DNS notation. This map is used to distinguish dependency translation keys
// from application translation keys.
func getDependencyIdentifiers(sess *session.Context) (map[string]bool, error) {
	moduleRoot, err := findModuleRoot(sess)
	if err != nil {
		return nil, fmt.Errorf("failed to find module root: %w", err)
	}

	gomodPath := filepath.Join(moduleRoot, "go.mod")
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}

	modFile, err := modfile.Parse(gomodPath, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	deps := make(map[string]bool)

	// Get the app's own module path to exclude it from dependencies
	appModulePath := ""
	if modFile.Module != nil {
		appModulePath = modFile.Module.Mod.Path
	}

	// Convert each dependency module path to reverse DNS identifier
	if modFile.Require != nil {
		for _, req := range modFile.Require {
			// Skip the app's own module (it's not a dependency of itself)
			if req.Mod.Path == appModulePath {
				continue
			}

			// Convert module path to reverse DNS identifier format
			addr, err := address.FromModule("dummy", req.Mod.Path)
			if err != nil {
				// Skip dependencies that cannot be converted (shouldn't happen normally)
				continue
			}
			identifier := addr.ReverseDNS()
			if identifier != "" {
				deps[identifier] = true
			}
		}
	}

	return deps, nil
}

// isDependencyKey determines if a translation key belongs to a dependency package
// (not the application itself). Returns true if the key's root identifier matches
// any dependency module identifier.
func isDependencyKey(sess *session.Context, key string) (bool, error) {
	deps, err := getDependencyIdentifiers(sess)
	if err != nil {
		return false, err
	}

	// Check if key starts with any dependency identifier
	for depID := range deps {
		if strings.HasPrefix(key, depID+".") {
			return true, nil
		}
	}

	// Also check if the key itself is a dependency identifier
	if deps[key] {
		return true, nil
	}

	// Extract root key and check if it's a dependency
	parts := strings.Split(key, ".")
	if len(parts) >= 2 {
		// Check progressively longer prefixes
		for i := 2; i <= len(parts); i++ {
			prefix := strings.Join(parts[:i], ".")
			if deps[prefix] {
				return true, nil
			}
		}
	}

	return false, nil
}

// getAppSupportedLanguages retrieves the app's configured supported languages from settings.
// Returns nil if not configured, which causes the command to use all registered i18n languages.
// The setting value is a StringSlice which uses Unit Separator (\x1f) as delimiter.
func getAppSupportedLanguages(sess *session.Context) []language.Tag {
	supportedSetting := sess.Settings().Get("app.i18n.supported")
	if !supportedSetting.IsSet() {
		return nil
	}

	// Get the value as a variable to access Fields() method
	varVal := supportedSetting.Value()
	if varVal.Empty() {
		return nil
	}

	// Use Fields() method which correctly handles \x1f separator for KindSlice
	// See pkg/vars/value.go for implementation details
	supportedStrings := varVal.Fields()

	if len(supportedStrings) == 0 {
		return nil
	}

	// Parse each language code string into a language.Tag
	supportedLangs := make([]language.Tag, 0, len(supportedStrings))
	for _, codeStr := range supportedStrings {
		if codeStr == "" {
			continue
		}
		lang, err := language.Parse(codeStr)
		if err != nil {
			// Skip invalid language codes silently
			continue
		}
		supportedLangs = append(supportedLangs, lang)
	}

	return supportedLangs
}

// isDependencyKeyForEntry determines if a translation key belongs to a dependency
// by checking if it matches any identifier in the dependency map. This is an
// optimized version that accepts a pre-computed dependency map to avoid repeated lookups.
func isDependencyKeyForEntry(key string, deps map[string]bool) (bool, error) {
	// Check if key starts with any dependency identifier prefix
	for depID := range deps {
		if strings.HasPrefix(key, depID+".") {
			return true, nil
		}
	}

	// Check if the key itself is a dependency root identifier
	if deps[key] {
		return true, nil
	}

	// Check progressively longer prefixes of the key to find matching dependency root
	parts := strings.Split(key, ".")
	if len(parts) >= 2 {
		for i := 2; i <= len(parts); i++ {
			prefix := strings.Join(parts[:i], ".")
			if deps[prefix] {
				return true, nil
			}
		}
	}

	return false, nil
}
