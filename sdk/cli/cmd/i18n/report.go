// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package i18n

import (
	"fmt"
	"os"
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

func i18nReport() *command.Command {
	cmd := command.New("report",
		command.Config{
			Description: settings.String(i18np + ".report.description"),
			Immediate:   true,
		})

	cmd.WithFlags(
		varflag.Float64Func("threshold", 0.0, i18np+".report.flag_threshold", "t"),
		varflag.StringFunc("lang", "", i18np+".report.flag_lang", "l"),
		// By default, report only on application translations. Use --with-deps to include dependencies.
		varflag.BoolFunc("with-deps", false, i18np+".report.flag_with_deps"),
	)

	cmd.Do(func(sess *session.Context, args action.Args) error {
		threshold := args.Flag("threshold").Var().Float64()
		langFlag := args.Flag("lang").String()
		includeDeps := args.Flag("with-deps").Var().Bool()

		// Determine which languages to report on based on app configuration.
		appSupportedLangs := getAppSupportedLanguages(sess)
		var languages []language.Tag

		if langFlag != "" {
			// Filter to specific language when --lang flag is provided
			targetLang, err := language.Parse(langFlag)
			if err != nil {
				return fmt.Errorf("invalid language: %w", err)
			}

			// If app has configured supported languages, ensure the requested one is among them.
			if len(appSupportedLangs) > 0 {
				found := false
				for _, lang := range appSupportedLangs {
					if lang == targetLang {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("language %s is not supported by this application", langFlag)
				}
			}

			languages = []language.Tag{targetLang}
		} else if len(appSupportedLangs) > 0 {
			// Use only application-supported languages
			languages = appSupportedLangs
		} else {
			// Fallback: use all registered languages from i18n system
			allLanguages := i18n.GetLanguages()
			if len(allLanguages) == 0 {
				return fmt.Errorf("no languages found")
			}
			languages = allLanguages
			// Sort for consistent output ordering
			sort.Slice(languages, func(i, j int) bool {
				return languages[i].String() < languages[j].String()
			})
		}

		// Create main table container for all report sections
		mainTable := textfmt.NewTable(
			textfmt.TableTitle("Translation Status Report"),
		)

		// Create language status table for displaying per-language statistics
		langTable := textfmt.NewTable(
			textfmt.TableTitle("Language Status"),
			textfmt.TableWithHeader(),
		)
		langTable.AddRow("Language", "Total", "Translated", "Missing", "Percentage", "Root Keys")

		// Track if any language falls below the threshold for exit code determination
		belowThreshold := false

		// Get dependency identifiers from go.mod to distinguish app vs dependency keys
		deps, err := getDependencyIdentifiers(sess)
		if err != nil {
			return fmt.Errorf("failed to get dependency identifiers: %w", err)
		}

		// Get fallback language - it should always be considered 100% translated
		fallbackLang := i18n.GetFallbackLanguage()

		if !includeDeps {
			// Default: only show app translations
			// Add rows for each language
			for _, lang := range languages {
				report := i18n.GetTranslationReport(lang)
				// Filter to only app keys (exclude dependency keys)
				filteredReport := filterReportByDependencies(sess, report, deps)

				// Fallback language is always 100% translated (it's the base language)
				if lang == fallbackLang {
					filteredReport.Percentage = 100.0
					filteredReport.Translated = filteredReport.Total
					filteredReport.Missing = 0
				}

				percentageStr := fmt.Sprintf("%.1f%%", filteredReport.Percentage)
				rootKeysCount := len(filteredReport.RootKeys)

				langTable.AddRow(
					lang.String(),
					fmt.Sprintf("%d", filteredReport.Total),
					fmt.Sprintf("%d", filteredReport.Translated),
					fmt.Sprintf("%d", filteredReport.Missing),
					percentageStr,
					fmt.Sprintf("%d", rootKeysCount),
				)

				if threshold > 0 && filteredReport.Percentage < threshold {
					belowThreshold = true
				}
			}

			// Append language table to main table
			mainTable.Append(langTable)

			// Add per-root-key statistics if available
			if len(languages) > 0 {
				firstReport := i18n.GetTranslationReport(languages[0])
				filteredFirstReport := filterReportByDependencies(sess, firstReport, deps)
				
				if len(filteredFirstReport.RootKeys) > 0 {
					perRootTable := textfmt.NewTable(
						textfmt.TableTitle("Per Root Key Statistics"),
						textfmt.TableWithHeader(),
					)
					perRootTable.AddRow("Root Key", "Total", "Translated", "Missing", "Percentage")

					// Use first language's report for root key statistics
					for _, rootKey := range filteredFirstReport.RootKeys {
						stats := filteredFirstReport.PerRootKey[rootKey]
						perRootTable.AddRow(
							rootKey,
							fmt.Sprintf("%d", stats.Total),
							fmt.Sprintf("%d", stats.Translated),
							fmt.Sprintf("%d", stats.Missing),
							fmt.Sprintf("%.1f%%", stats.Percentage),
						)
					}
					mainTable.Append(perRootTable)
				}
			}
		} else {
			// Default behavior: show both app and dependency translations in separate sections
			// Application translations section
			appLangTable := textfmt.NewTable(
				textfmt.TableTitle("Application Translations"),
				textfmt.TableWithHeader(),
			)
			appLangTable.AddRow("Language", "Total", "Translated", "Missing", "Percentage", "Root Keys")
			
			// Dependency translations section (lazily created only if missing translations exist)
			var depLangTable *textfmt.Table

			// Process each language to generate statistics
			for _, lang := range languages {
				report := i18n.GetTranslationReport(lang)
				
				// Separate app translations from dependency translations
				appReport := filterReportByDependencies(sess, report, deps)
				
				// Fallback language is always 100% translated (it's the base language)
				if lang == fallbackLang {
					appReport.Percentage = 100.0
					appReport.Translated = appReport.Total
					appReport.Missing = 0
				}
				
				// Calculate dependency statistics by subtracting app stats from total
				depReport := i18n.TranslationReport{
					Language:       report.Language,
					Total:          report.Total - appReport.Total,
					Translated:     report.Translated - appReport.Translated,
					Missing:        report.Missing - appReport.Missing,
					Percentage:     0.0,
					MissingEntries: make([]i18n.TranslationEntry, 0),
					RootKeys:        make([]string, 0),
					PerRootKey:      make(map[string]i18n.RootKeyStats),
				}
				if depReport.Total > 0 {
					depReport.Percentage = float64(depReport.Translated) / float64(depReport.Total) * 100.0
				}
				// Fallback language dependencies are also 100% (base language)
				if lang == fallbackLang && depReport.Total > 0 {
					depReport.Percentage = 100.0
					depReport.Translated = depReport.Total
					depReport.Missing = 0
				}

				// Create dependency table only if there are missing dependency translations
				// This avoids cluttering the output when dependencies are fully translated
				if depReport.Missing > 0 {
					if depLangTable == nil {
						depLangTable = textfmt.NewTable(
							textfmt.TableTitle("Dependency Translations"),
							textfmt.TableWithHeader(),
						)
						depLangTable.AddRow("Language", "Total", "Translated", "Missing", "Percentage", "Root Keys")
					}
				}

				// Add application translation statistics row
				appPercentageStr := fmt.Sprintf("%.1f%%", appReport.Percentage)
				appRootKeysCount := len(appReport.RootKeys)
				appLangTable.AddRow(
					lang.String(),
					fmt.Sprintf("%d", appReport.Total),
					fmt.Sprintf("%d", appReport.Translated),
					fmt.Sprintf("%d", appReport.Missing),
					appPercentageStr,
					fmt.Sprintf("%d", appRootKeysCount),
				)

				// Add dependency translation statistics row only if table was created
				if depLangTable != nil {
					depPercentageStr := fmt.Sprintf("%.1f%%", depReport.Percentage)
					depRootKeysCount := len(report.RootKeys) - appRootKeysCount
					if depRootKeysCount < 0 {
						depRootKeysCount = 0
					}
					depLangTable.AddRow(
						lang.String(),
						fmt.Sprintf("%d", depReport.Total),
						fmt.Sprintf("%d", depReport.Translated),
						fmt.Sprintf("%d", depReport.Missing),
						depPercentageStr,
						fmt.Sprintf("%d", depRootKeysCount),
					)
				}

				// Check threshold against app translations only
				// The threshold check uses app translations to determine if the project meets the quality bar
				if threshold > 0 && appReport.Percentage < threshold {
					belowThreshold = true
				}
			}

			// Append application translations table to main report
			mainTable.Append(appLangTable)
			
			// Append dependency translations table only if it was created
			// (i.e., only when there are missing dependency translations)
			if depLangTable != nil {
				mainTable.Append(depLangTable)
			}

			// Add per-root-key statistics for application translations
			if len(languages) > 0 {
				firstReport := i18n.GetTranslationReport(languages[0])
				filteredFirstReport := filterReportByDependencies(sess, firstReport, deps)
				
				if len(filteredFirstReport.RootKeys) > 0 {
					perRootTable := textfmt.NewTable(
						textfmt.TableTitle("Per Root Key Statistics (Application)"),
						textfmt.TableWithHeader(),
					)
					perRootTable.AddRow("Root Key", "Total", "Translated", "Missing", "Percentage")

					// Use first language's report for root key statistics breakdown
					for _, rootKey := range filteredFirstReport.RootKeys {
						stats := filteredFirstReport.PerRootKey[rootKey]
						perRootTable.AddRow(
							rootKey,
							fmt.Sprintf("%d", stats.Total),
							fmt.Sprintf("%d", stats.Translated),
							fmt.Sprintf("%d", stats.Missing),
							fmt.Sprintf("%.1f%%", stats.Percentage),
						)
					}
					mainTable.Append(perRootTable)
				}
			}
		}

		// Output the complete report
		fmt.Println(mainTable.String())

		// Exit with code 1 if any language falls below the threshold
		// This enables CI/CD workflows to fail builds when translation coverage is insufficient
		if belowThreshold {
			os.Exit(1)
		}

		return nil
	})

	return cmd
}

// filterReportByDependencies filters a translation report to exclude dependency keys
func filterReportByDependencies(sess *session.Context, report i18n.TranslationReport, deps map[string]bool) i18n.TranslationReport {
	filtered := i18n.TranslationReport{
		Language:       report.Language,
		Total:          0,
		Translated:     0,
		Missing:        0,
		Percentage:     0.0,
		MissingEntries: make([]i18n.TranslationEntry, 0),
		RootKeys:        make([]string, 0),
		PerRootKey:      make(map[string]i18n.RootKeyStats),
	}

	// Get all entries and filter them
	allEntries := i18n.GetAllTranslations()
	appEntries := make([]i18n.TranslationEntry, 0)
	rootKeysSet := make(map[string]bool)
	perRootKeyStats := make(map[string]struct {
		total      int
		translated int
		missing    int
	})

	for _, entry := range allEntries {
		// Check if this is a dependency key
		isDep, err := isDependencyKeyForEntry(entry.Key, deps)
		if err != nil {
			// Skip on error
			continue
		}
		if isDep {
			continue // Skip dependency keys
		}

		appEntries = append(appEntries, entry)
		rootKey := entry.RootKey
		if rootKey == "" {
			// Extract root key from the key itself (first segment before first dot)
			parts := strings.Split(entry.Key, ".")
			if len(parts) > 0 {
				rootKey = parts[0]
			} else {
				rootKey = "unknown"
			}
		}
		rootKeysSet[rootKey] = true

		stats := perRootKeyStats[rootKey]
		stats.total++

		if _, hasTranslation := entry.Translations[report.Language]; hasTranslation {
			filtered.Translated++
			stats.translated++
		} else {
			filtered.Missing++
			filtered.MissingEntries = append(filtered.MissingEntries, entry)
			stats.missing++
		}
		perRootKeyStats[rootKey] = stats
	}

	filtered.Total = len(appEntries)
	if filtered.Total > 0 {
		filtered.Percentage = float64(filtered.Translated) / float64(filtered.Total) * 100.0
	}

	// Build root keys list
	for rootKey := range rootKeysSet {
		filtered.RootKeys = append(filtered.RootKeys, rootKey)
	}
	sort.Strings(filtered.RootKeys)

	// Build per-root-key stats
	for rootKey, stats := range perRootKeyStats {
		rootPercentage := 0.0
		if stats.total > 0 {
			rootPercentage = float64(stats.translated) / float64(stats.total) * 100.0
		}
		filtered.PerRootKey[rootKey] = i18n.RootKeyStats{
			RootKey:    rootKey,
			Total:      stats.total,
			Translated: stats.translated,
			Missing:    stats.missing,
			Percentage: rootPercentage,
		}
	}

	return filtered
}

