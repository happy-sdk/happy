// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

func i18nTranslate() *command.Command {
	cmd := command.New("translate",
		command.Config{
			Description: settings.String(i18np + ".translate.description"),
			Immediate:   true,
		})

	cmd.WithFlags(
		varflag.StringFunc("lang", "", i18np+".translate.flag_lang", "l"),
		// By default, prompt only for application translations. Use --with-deps to include dependency keys.
		varflag.BoolFunc("with-deps", false, i18np+".translate.flag_with_deps"),
	)

	cmd.Do(func(sess *session.Context, args action.Args) error {
		langFlag := args.Flag("lang").String()
		includeDeps := args.Flag("with-deps").Var().Bool()
		positionalArgs := args.Args()

		// Non-interactive mode: both i18n_key and value provided with --lang flag
		// Usage: translate --lang=et com.github.happy-sdk.happy.sdk.cli.flags.debug "Luba debug-taseme logimine"
		if len(positionalArgs) >= 2 && langFlag != "" {
			i18nKey := positionalArgs[0].String()
			value := positionalArgs[1].String()
			lang, err := language.Parse(langFlag)
			if err != nil {
				return fmt.Errorf("invalid language: %w", err)
			}
			return updateTranslation(sess, lang, i18nKey, value)
		}

		// Interactive mode: no arguments provided, loop through all missing translations
		if len(positionalArgs) == 0 {
			return interactiveMode(sess, includeDeps)
		}

		// Single key mode: translate specific key for missing languages
		if len(positionalArgs) == 1 {
			i18nKey := positionalArgs[0].String()
			if langFlag != "" {
				// Translate specific key for specific language (interactive prompt)
				lang, err := language.Parse(langFlag)
				if err != nil {
					return fmt.Errorf("invalid language: %w", err)
				}
				return translateSpecificKeyForLang(sess, lang, i18nKey)
			}
			// Translate specific key for all missing languages (interactive prompts)
			return translateSpecificKeyInteractive(sess, i18nKey)
		}

		return fmt.Errorf("invalid arguments: expected 0, 1, or 2 positional arguments")
	})

	return cmd
}

func interactiveMode(sess *session.Context, includeDeps bool) error {
	fmt.Println("Starting interactive translation for all missing keys...")
	// Determine languages based on app configuration, falling back to all i18n languages.
	appSupportedLangs := getAppSupportedLanguages(sess)
	fallbackLang := i18n.GetFallbackLanguage()
	var languages []language.Tag
	if len(appSupportedLangs) > 0 {
		languages = appSupportedLangs
	} else {
		all := i18n.GetLanguages()
		for _, lang := range all {
			if lang == fallbackLang {
				continue
			}
			languages = append(languages, lang)
		}
	}

	for _, lang := range languages {
		if lang == fallbackLang {
			continue
		}
		report := i18n.GetTranslationReport(lang)

		// Filter missing entries based on whether they belong to the app or dependencies.
		deps, err := getDependencyIdentifiers(sess)
		if err != nil {
			return fmt.Errorf("failed to get dependency identifiers: %w", err)
		}

		missingEntries := make([]i18n.TranslationEntry, 0, len(report.MissingEntries))
		for _, entry := range report.MissingEntries {
			isDep, err := isDependencyKeyForEntry(entry.Key, deps)
			if err != nil {
				return err
			}
			if !includeDeps && isDep {
				continue
			}
			if includeDeps || !isDep {
				missingEntries = append(missingEntries, entry)
			}
		}

		if len(missingEntries) > 0 {
			fmt.Printf("\n--- Missing translations for %s (%d entries) ---\n", lang.String(), len(missingEntries))
			for _, entry := range missingEntries {
				fmt.Printf("Key: %q\n", entry.Key)
				fmt.Printf("Fallback (%s): %q\n", fallbackLang.String(), entry.Fallback)
				fmt.Printf("Enter translation for %q in %s (leave empty to skip): ", entry.Key, lang.String())
				input, err := readLine()
				if err != nil {
					return err
				}
				if input != "" {
					if err := updateTranslation(sess, lang, entry.Key, input); err != nil {
						return err
					}
				}
			}
		}
	}
	fmt.Println("\nInteractive translation session completed.")
	return nil
}

func translateSpecificKeyInteractive(sess *session.Context, key string) error {
	fmt.Printf("Translating key: %q\n", key)
	// Use fallback language report to get all translation entries (not just missing ones)
	report := i18n.GetTranslationReport(i18n.GetFallbackLanguage())

	var targetEntry *i18n.TranslationEntry
	for _, entry := range report.MissingEntries {
		if entry.Key == key {
			targetEntry = &entry
			break
		}
	}

	if targetEntry == nil {
		// Check if key exists at all
		allEntries := i18n.GetAllTranslations()
		for _, entry := range allEntries {
			if entry.Key == key {
				fmt.Printf("Key %q is already fully translated.\n", key)
				return nil
			}
		}
		return fmt.Errorf("key %q not found", key)
	}

	// Use app configuration for languages where possible
	appSupportedLangs := getAppSupportedLanguages(sess)
	fallbackLang := i18n.GetFallbackLanguage()
	var languages []language.Tag
	if len(appSupportedLangs) > 0 {
		languages = appSupportedLangs
	} else {
		all := i18n.GetLanguages()
		for _, lang := range all {
			if lang == fallbackLang {
				continue
			}
			languages = append(languages, lang)
		}
	}

	for _, lang := range languages {
		if lang == fallbackLang {
			continue // Skip fallback language as it's the source
		}
		if _, ok := targetEntry.Translations[lang]; !ok {
			fmt.Printf("Missing translation for %s (Fallback: %q):\n", lang.String(), targetEntry.Fallback)
			fmt.Printf("Enter translation for %q in %s: ", key, lang.String())
			input, err := readLine()
			if err != nil {
				return err
			}
			if input != "" {
				if err := updateTranslation(sess, lang, key, input); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func translateSpecificKeyForLang(sess *session.Context, lang language.Tag, key string) error {
	allEntries := i18n.GetAllTranslations()
	var targetEntry *i18n.TranslationEntry
	for _, entry := range allEntries {
		if entry.Key == key {
			targetEntry = &entry
			break
		}
	}

	if targetEntry == nil {
		return fmt.Errorf("key %q not found", key)
	}

	if _, ok := targetEntry.Translations[lang]; ok {
		fmt.Printf("Key %q already has translation for %s.\n", key, lang.String())
		return nil
	}

	fmt.Printf("Enter translation for %q in %s (Fallback: %q): ", key, lang.String(), targetEntry.Fallback)
	input, err := readLine()
	if err != nil {
		return err
	}
	if input != "" {
		return updateTranslation(sess, lang, key, input)
	}
	return nil
}

// readLine reads a full line from stdin, allowing empty input to be treated as "skip".
func readLine() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err.Error() != "EOF" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// updateTranslation saves a translation to the appropriate JSON file based on whether
// the key belongs to the application or a dependency. App translations are saved to
// <lang>.json files, while dependency translations are saved to dependencies.json.
func updateTranslation(sess *session.Context, lang language.Tag, key, value string) error {
	// Find the module root directory where translation files are stored
	moduleRoot, err := findModuleRoot(sess)
	if err != nil {
		return fmt.Errorf("failed to find module root: %w", err)
	}

	// Determine if this key belongs to a dependency or the application
	isDep, err := isDependencyKey(sess, key)
	if err != nil {
		return fmt.Errorf("failed to check if key is dependency: %w", err)
	}

	var jsonPath string
	var jsonKey string

	if isDep {
		// Dependency translations are stored in a single dependencies.json file
		// Structure: { "rootKey": { "lang": { "key": "value" } } }
		jsonPath = filepath.Join(moduleRoot, "i18n", "dependencies.json")
		
		// Extract the root key (package identifier) from the full translation key
		// Example: "com.github.happy-sdk.happy.sdk.cli.flags.version"
		//          -> root key: "com.github.happy-sdk.happy.sdk.cli"
		rootKey := extractRootKey(key)
		
		// Construct the nested JSON path: rootKey.lang.keyWithoutRoot
		// This creates the structure: dependencies[rootKey][lang][keyWithoutRoot] = value
		keyWithoutRoot := strings.TrimPrefix(key, rootKey+".")
		jsonKey = rootKey + "." + lang.String() + "." + keyWithoutRoot
	} else {
		// Application translations are stored in language-specific JSON files
		// Structure: { "key": "value" } (flat or nested based on key structure)
		jsonPath = filepath.Join(moduleRoot, "i18n", lang.String()+".json")
		
		// Get the app's module identifier prefix to remove from the key
		prefix, err := getAppModulePrefix(sess)
		if err != nil {
			return fmt.Errorf("failed to get app module prefix: %w", err)
		}
		
		// Remove the module prefix to get the relative key path
		// Example: prefix = "com.github.happy-sdk.banctl"
		//          key = "com.github.happy-sdk.banctl.help.description"
		//          jsonKey = "help.description"
		if !strings.HasPrefix(key, prefix) {
			return fmt.Errorf("key %q does not start with app module prefix %q", key, prefix)
		}
		jsonKey = strings.TrimPrefix(key, prefix+".")
	}

	// Ensure the i18n directory exists in the module root
	i18nDir := filepath.Dir(jsonPath)
	if err := os.MkdirAll(i18nDir, 0755); err != nil {
		return fmt.Errorf("failed to create i18n directory: %w", err)
	}

	// Read existing JSON file or initialize empty map if file doesn't exist
	var translations map[string]any
	if data, err := os.ReadFile(jsonPath); err == nil {
		if err := json.Unmarshal(data, &translations); err != nil {
			return fmt.Errorf("failed to parse JSON file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read JSON file: %w", err)
	} else {
		translations = make(map[string]any)
	}

	// Set the nested value in the translations map using dot-separated key path
	setNestedValue(translations, jsonKey, value)

	// Marshal the updated translations back to JSON with indentation
	updatedData, err := json.MarshalIndent(translations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write the updated JSON back to the file
	if err := os.WriteFile(jsonPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	// Register the translation in the i18n system (allows overwriting existing keys)
	if err := i18n.RegisterTranslation(lang, key, value); err != nil {
		return fmt.Errorf("failed to register translation: %w", err)
	}

	fmt.Printf("Translation saved to %s\n", jsonPath)
	return nil
}

// extractRootKey extracts the root package identifier from a full translation key.
// The root key is typically the first 5-6 dot-separated parts, with special handling
// for "pkg" and "sdk" segments which extend the root key by one part.
// Example: "com.github.happy-sdk.happy.sdk.cli.flags.version"
//          -> "com.github.happy-sdk.happy.sdk.cli"
func extractRootKey(key string) string {
	parts := strings.Split(key, ".")
	if len(parts) < 5 {
		// Key is too short to have a meaningful root, return as-is
		return key
	}
	
	// Default root key is first 5 parts
	rootKey := strings.Join(parts[:5], ".")
	// Extend root key if "pkg" or "sdk" is the 5th part (index 4)
	if len(parts) >= 6 && (parts[4] == "pkg" || parts[4] == "sdk") {
		rootKey = strings.Join(parts[:6], ".")
	}
	
	return rootKey
}

// setNestedValue sets a nested value in a map using a dot-separated key path.
// It creates intermediate nested maps as needed to reach the target key.
// Example: setNestedValue(m, "cmd.help.description", "value")
//          creates m["cmd"]["help"]["description"] = "value"
func setNestedValue(m map[string]any, key string, value string) {
	parts := strings.Split(key, ".")
	current := m

	// Navigate through the nested structure, creating maps as needed
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if next, ok := current[part].(map[string]any); ok {
			// Path already exists, continue navigating
			current = next
		} else {
			// Create nested map for this path segment
			current[part] = make(map[string]any)
			current = current[part].(map[string]any)
		}
	}

	// Set the final value at the target key
	current[parts[len(parts)-1]] = value
}

