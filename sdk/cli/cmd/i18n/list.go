// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

type keyStatus struct {
	Key             string
	TranslatedLangs []string
	MissingLangs    []string
	HasTranslation  bool
	Status          string // "ok" or "missing" (for single language mode)
}

func i18nList() *command.Command {
	cmd := command.New("list",
		command.Config{
			Description: settings.String(i18np + ".list.description"),
			Immediate:   true,
		})

	cmd.WithFlags(
		varflag.StringFunc("lang", "", i18np+".list.flag_lang", "l"),
		varflag.BoolFunc("missing", false, i18np+".list.flag_missing", "m"),
		varflag.BoolFunc("json", false, i18np+".list.flag_json"),
		// By default, list only application translations. Use --with-deps to include dependencies.
		varflag.BoolFunc("with-deps", false, i18np+".list.flag_with_deps"),
	)

	cmd.Do(func(sess *session.Context, args action.Args) error {
		langFlag := args.Flag("lang").String()
		missingOnly := args.Flag("missing").Var().Bool()
		jsonOutput := args.Flag("json").Var().Bool()
		includeDeps := args.Flag("with-deps").Var().Bool()

		allEntries := i18n.GetAllTranslations()

		// Get dependency identifiers from go.mod to distinguish app vs dependency keys
		deps, err := getDependencyIdentifiers(sess)
		if err != nil {
			return fmt.Errorf("failed to get dependency identifiers: %w", err)
		}

		// Separate translation entries into app and dependency categories
		appEntries := make([]i18n.TranslationEntry, 0)
		depEntries := make([]i18n.TranslationEntry, 0)

		for _, entry := range allEntries {
			isDep, err := isDependencyKeyForEntry(entry.Key, deps)
			if err != nil {
				return fmt.Errorf("failed to check if key is dependency: %w", err)
			}
			if isDep {
				depEntries = append(depEntries, entry)
			} else {
				appEntries = append(appEntries, entry)
			}
		}

		// By default, operate only on application entries.
		// When --with-deps is set, dependency entries are handled in a separate section.
		allEntries = appEntries

		// Determine which languages to check based on app configuration
		appSupportedLangs := getAppSupportedLanguages(sess)
		fallbackLang := i18n.GetFallbackLanguage()

		var translatableLangs []language.Tag
		if len(appSupportedLangs) > 0 {
			// Use app's configured supported languages
			translatableLangs = appSupportedLangs
		} else {
			// Fallback to all registered i18n languages (excluding fallback)
			languages := i18n.GetLanguages()
			translatableLangs = make([]language.Tag, 0)
			for _, lang := range languages {
				if lang != fallbackLang {
					translatableLangs = append(translatableLangs, lang)
				}
			}
		}

		// Parse target language if --lang flag is provided
		var targetLang language.Tag
		if langFlag != "" {
			var err error
			targetLang, err = language.Parse(langFlag)
			if err != nil {
				return fmt.Errorf("invalid language: %w", err)
			}
		}

		// Process entries to determine translation status for each key
		keyStatuses := processEntriesForDisplay(allEntries, translatableLangs, targetLang, fallbackLang, missingOnly)

	// When --missing is used and there are no missing keys, show a friendly message
	if missingOnly && len(keyStatuses) == 0 && !includeDeps {
		if targetLang != language.Und {
			fmt.Printf("All application translations are complete for %s.\n", targetLang.String())
		} else {
			fmt.Println("All application translations are complete for all configured languages.")
		}
		return nil
	}

		// JSON output
		if jsonOutput {
			type jsonOutput struct {
				Keys []struct {
					Key             string   `json:"key"`
					TranslatedLangs []string `json:"translated_langs,omitempty"`
					MissingLangs    []string `json:"missing_langs,omitempty"`
					Status          string   `json:"status,omitempty"`
					HasTranslation  bool     `json:"has_translation"`
				} `json:"keys"`
			}

			output := jsonOutput{
				Keys: make([]struct {
					Key             string   `json:"key"`
					TranslatedLangs []string `json:"translated_langs,omitempty"`
					MissingLangs    []string `json:"missing_langs,omitempty"`
					Status          string   `json:"status,omitempty"`
					HasTranslation  bool     `json:"has_translation"`
				}, len(keyStatuses)),
			}

			for i, ks := range keyStatuses {
				output.Keys[i].Key = ks.Key
				output.Keys[i].HasTranslation = ks.HasTranslation
				if len(ks.TranslatedLangs) > 0 {
					output.Keys[i].TranslatedLangs = ks.TranslatedLangs
				}
				if len(ks.MissingLangs) > 0 {
					output.Keys[i].MissingLangs = ks.MissingLangs
				}
				if targetLang != language.Und {
					output.Keys[i].Status = ks.Status
				}
			}

			jsonData, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonData))
			return nil
		}

		// Generate table output based on flags
		if !includeDeps {
			// Default: show only application translations
			if targetLang != language.Und {
				// Single language mode: show key and status
				if missingOnly {
					// Only missing keys: show just key column
					table := textfmt.NewTable(
						textfmt.TableTitle(fmt.Sprintf("Missing Translation Keys for %s", targetLang.String())),
						textfmt.TableWithHeader(),
					)
					table.AddRow("Key")
					batch := textfmt.NewTableBatchOp()
					for _, ks := range keyStatuses {
						batch.AddRow(ks.Key)
					}
					table.Batch(batch)
					fmt.Println(table.String())
				} else {
					// All keys: show key and status
					table := textfmt.NewTable(
						textfmt.TableTitle(fmt.Sprintf("Translation Keys for %s", targetLang.String())),
						textfmt.TableWithHeader(),
					)
					table.AddRow("Key", "Status")
					batch := textfmt.NewTableBatchOp()
					for _, ks := range keyStatuses {
						batch.AddRow(ks.Key, ks.Status)
					}
					table.Batch(batch)
					fmt.Println(table.String())
				}
			} else {
				// All languages mode: show key, translated, missing
				table := textfmt.NewTable(
					textfmt.TableTitle("All Translation Keys"),
					textfmt.TableWithHeader(),
				)
				table.AddRow("Key", "Translated", "Missing")
				batch := textfmt.NewTableBatchOp()
				for _, ks := range keyStatuses {
					translatedStr := strings.Join(ks.TranslatedLangs, ", ")
					if translatedStr == "" {
						translatedStr = "-"
					}
					missingStr := strings.Join(ks.MissingLangs, ", ")
					if missingStr == "" {
						missingStr = "-"
					}
					batch.AddRow(ks.Key, translatedStr, missingStr)
				}
				table.Batch(batch)
				fmt.Println(table.String())
			}
		} else {
			// Show both app and dependency entries in separate sections
			// Process app entries
			appKeyStatuses := processEntriesForDisplay(appEntries, translatableLangs, targetLang, fallbackLang, missingOnly)
			// Process dependency entries without missingOnly filter to get all statuses
			// We need to check all entries to determine which have missing translations
			allDepKeyStatuses := processEntriesForDisplay(depEntries, translatableLangs, targetLang, fallbackLang, false)
			
			// Filter dependency entries to only include those with missing translations
			// This ensures the dependency section only shows keys that need attention
			depKeyStatuses := make([]keyStatus, 0)
			for _, ks := range allDepKeyStatuses {
				if len(ks.MissingLangs) > 0 {
					depKeyStatuses = append(depKeyStatuses, ks)
				}
			}
			
			// Create main table
			mainTable := textfmt.NewTable(
				textfmt.TableTitle("All Translation Keys"),
			)
			
			// App translations section
			if len(appKeyStatuses) > 0 {
				appTable := textfmt.NewTable(
					textfmt.TableTitle("Application Translations"),
					textfmt.TableWithHeader(),
				)
				if targetLang != language.Und {
					if missingOnly {
						appTable.AddRow("Key")
						batch := textfmt.NewTableBatchOp()
						for _, ks := range appKeyStatuses {
							batch.AddRow(ks.Key)
						}
						appTable.Batch(batch)
					} else {
						appTable.AddRow("Key", "Status")
						batch := textfmt.NewTableBatchOp()
						for _, ks := range appKeyStatuses {
							batch.AddRow(ks.Key, ks.Status)
						}
						appTable.Batch(batch)
					}
				} else {
					appTable.AddRow("Key", "Translated", "Missing")
					batch := textfmt.NewTableBatchOp()
					for _, ks := range appKeyStatuses {
						translatedStr := strings.Join(ks.TranslatedLangs, ", ")
						if translatedStr == "" {
							translatedStr = "-"
						}
						missingStr := strings.Join(ks.MissingLangs, ", ")
						if missingStr == "" {
							missingStr = "-"
						}
						batch.AddRow(ks.Key, translatedStr, missingStr)
					}
					appTable.Batch(batch)
				}
				mainTable.Append(appTable)
			}
			
			// Dependency translations section - only show if there are missing translations
			if len(depKeyStatuses) > 0 {
				depTable := textfmt.NewTable(
					textfmt.TableTitle("Dependency Translations"),
					textfmt.TableWithHeader(),
				)
				if targetLang != language.Und {
					if missingOnly {
						depTable.AddRow("Key")
						batch := textfmt.NewTableBatchOp()
						for _, ks := range depKeyStatuses {
							batch.AddRow(ks.Key)
						}
						depTable.Batch(batch)
					} else {
						depTable.AddRow("Key", "Status")
						batch := textfmt.NewTableBatchOp()
						for _, ks := range depKeyStatuses {
							batch.AddRow(ks.Key, ks.Status)
						}
						depTable.Batch(batch)
					}
				} else {
					depTable.AddRow("Key", "Translated", "Missing")
					batch := textfmt.NewTableBatchOp()
					for _, ks := range depKeyStatuses {
						translatedStr := strings.Join(ks.TranslatedLangs, ", ")
						if translatedStr == "" {
							translatedStr = "-"
						}
						missingStr := strings.Join(ks.MissingLangs, ", ")
						if missingStr == "" {
							missingStr = "-"
						}
						batch.AddRow(ks.Key, translatedStr, missingStr)
					}
					depTable.Batch(batch)
				}
				mainTable.Append(depTable)
			}
			
			fmt.Println(mainTable.String())
		}

		return nil
	})

	return cmd
}

// processEntriesForDisplay processes translation entries and returns key statuses
func processEntriesForDisplay(entries []i18n.TranslationEntry, translatableLangs []language.Tag, targetLang language.Tag, fallbackLang language.Tag, missingOnly bool) []keyStatus {
	keyStatuses := make([]keyStatus, 0, len(entries))

	for _, entry := range entries {
		missingLangs := make([]string, 0)
		translatedLangs := make([]string, 0)
		allTranslated := true

		// Check translation status for each language
		// When targetLang is set, only check that language
		// When targetLang is not set, check all languages
		languagesToCheck := translatableLangs
		if targetLang != language.Und {
			languagesToCheck = []language.Tag{targetLang}
		}

		for _, lang := range languagesToCheck {
			// Check if translation exists for this specific language
			hasLangTranslation := false
			if lang == fallbackLang {
				// For fallback language, check if Fallback value is set
				hasLangTranslation = entry.Fallback != ""
			} else {
				// For other languages, check Translations map
				_, hasLangTranslation = entry.Translations[lang]
			}
			
			if hasLangTranslation {
				translatedLangs = append(translatedLangs, lang.String())
			} else {
				missingLangs = append(missingLangs, lang.String())
				allTranslated = false
			}
		}

		// Filter by missing-only flag
		if missingOnly && allTranslated {
			continue
		}

		// Status for single language mode
		status := "ok"
		if !allTranslated {
			status = "missing"
		}

		keyStatuses = append(keyStatuses, keyStatus{
			Key:             entry.Key,
			TranslatedLangs: translatedLangs,
			MissingLangs:    missingLangs,
			HasTranslation:  allTranslated,
			Status:          status,
		})
	}

	// Sort by key
	sort.Slice(keyStatuses, func(i, j int) bool {
		return keyStatuses[i].Key < keyStatuses[j].Key
	})

	return keyStatuses
}

