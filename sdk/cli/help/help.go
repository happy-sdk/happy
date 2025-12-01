// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package help

import (
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/tui/ansicolor"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
)

const i18np = "com.github.happy-sdk.happy.sdk.cli.help"

type Help struct {
	style       Style
	info        *Info
	cmds        map[string][]commandInfo
	flags       []flagInfo
	sharedFlags []flagInfo
	globalFlags []flagInfo
	catdesc     map[string]string
}

type commandInfo struct {
	name        string
	description string
}

type flagInfo struct {
	Flag         string
	UsageAliases string
	Usage        string
}

type Style struct {
	Primary     ansicolor.Style
	Info        ansicolor.Style
	Version     ansicolor.Style
	Credits     ansicolor.Style
	License     ansicolor.Style
	Description ansicolor.Style
	Category    ansicolor.Style
}

func New(info Info, style Style) *Help {
	return &Help{
		style:   style,
		info:    &info,
		cmds:    make(map[string][]commandInfo),
		catdesc: make(map[string]string),
	}
}

func (h *Help) AddCommand(category, name, description string) {
	if category == "" {
		category = "default"
	}
	h.cmds[category] = append(h.cmds[category], commandInfo{
		name:        name,
		description: description,
	})
}

func (h *Help) AddGlobalFlags(flags []varflag.Flag) {
	if flags == nil {
		return
	}
	for _, flag := range flags {
		h.globalFlags = append(h.globalFlags, flagInfo{
			Flag:         flag.Flag(),
			UsageAliases: flag.UsageAliases(),
			Usage:        flag.Usage(),
		})
	}
}

func (h *Help) AddSharedFlags(flags []varflag.Flag) {
	if flags == nil {
		return
	}
	for _, flag := range flags {
		h.sharedFlags = append(h.sharedFlags, flagInfo{
			Flag:         flag.Flag(),
			UsageAliases: flag.UsageAliases(),
			Usage:        flag.Usage(),
		})
	}
}

func (h *Help) AddCommandFlags(flags []varflag.Flag) {
	if flags == nil {
		return
	}
	for _, flag := range flags {
		h.flags = append(h.flags, flagInfo{
			Flag:         flag.Flag(),
			UsageAliases: flag.UsageAliases(),
			Usage:        flag.Usage(),
		})
	}
}
func (h *Help) AddCategoryDescriptions(catdescs map[string]string) {
	maps.Copy(catdescs, h.catdesc)
}

func (h *Help) Print() error {
	if err := h.printBanner(); err != nil {
		return err
	}
	if err := h.printInfo(); err != nil {
		return err
	}

	if err := h.printCommands(); err != nil {
		return err
	}
	if err := h.printCommandFlags(); err != nil {
		return err
	}
	if err := h.printGlobalFlags(); err != nil {
		return err
	}
	fmt.Println("")
	return nil
}

func (h *Help) printCommands() error {
	// commands
	if len(h.cmds) > 0 {
		fmt.Println("")
		fmt.Println(h.style.Primary.String(i18n.T(i18np + ".commands")))
		var categories []string

		var maxNameLength int
		for _, commands := range h.cmds {
			if mnl := getMaxNameLength(commands) + 4; mnl > maxNameLength {
				maxNameLength = mnl
			}
		}

		// Collect all category names (except "default")
		for category := range h.cmds {
			if category != "default" {
				categories = append(categories, category)
			}
		}

		// Sort the categories alphabetically
		sort.Strings(categories)

		// Handle "default" category

		if commands, ok := h.cmds["default"]; ok {

			// Sort commands within the "default" category alphabetically
			sort.Slice(commands, func(i, j int) bool {
				return commands[i].name < commands[j].name
			})
			fmt.Println("")
			for _, cmd := range commands {
				h.printSubcommand(maxNameLength, cmd.name, cmd.description)
			}
		}

		// Print other categories
		for _, category := range categories {
			fmt.Println("")
			// Translate category if it looks like an i18n key (starts with "com.github.")
			categoryDisplay := category
			if strings.HasPrefix(category, "com.github.") {
				categoryDisplay = i18n.T(category)
			}
			fmt.Println(" ", h.style.Category.String(strings.ToUpper(categoryDisplay))+h.getCategoryDesc(category))
			fmt.Println("")
			commands := h.cmds[category]

			// Sort commands within each category alphabetically
			sort.Slice(commands, func(i, j int) bool {
				return commands[i].name < commands[j].name
			})

			for _, cmd := range commands {
				h.printSubcommand(maxNameLength, cmd.name, cmd.description)
			}
		}
	}
	return nil
}

func (h *Help) printCommandFlags() error {
	if len(h.flags) > 0 {
		fmt.Println("")
		fmt.Println(h.style.Primary.String(i18n.T(i18np + ".flags")))
		fmt.Println("")

		// Sort the globalFlags by flag name
		sort.Slice(h.flags, func(i, j int) bool {
			return h.flags[i].Flag < h.flags[j].Flag
		})

		var (
			maxFlagLength  int
			maxAliasLength int
		)
		for _, flag := range h.flags {
			if mfl := len(flag.Flag); mfl > maxFlagLength {
				maxFlagLength = mfl
			}
			if mal := len(flag.UsageAliases); mal > maxAliasLength {
				maxAliasLength = mal
			}
		}

		for _, flag := range h.flags {
			h.printFlag(maxFlagLength, maxAliasLength, flag)
		}
	}

	if len(h.sharedFlags) > 0 {
		fmt.Println("")
		fmt.Println(h.style.Primary.String(i18n.T(i18np + ".shared_flags")))
		fmt.Println("")

		// Sort the globalFlags by flag name
		sort.Slice(h.sharedFlags, func(i, j int) bool {
			return h.sharedFlags[i].Flag < h.sharedFlags[j].Flag
		})

		var (
			maxFlagLength  int
			maxAliasLength int
		)
		for _, flag := range h.sharedFlags {
			if mfl := len(flag.Flag); mfl > maxFlagLength {
				maxFlagLength = mfl
			}
			if mal := len(flag.UsageAliases); mal > maxAliasLength {
				maxAliasLength = mal
			}
		}

		for _, flag := range h.sharedFlags {
			h.printFlag(maxFlagLength, maxAliasLength, flag)
		}
	}

	return nil
}
func (h *Help) printGlobalFlags() error {
	if len(h.globalFlags) > 0 {
		fmt.Println("")
		fmt.Println(h.style.Primary.String(i18n.T(i18np + ".global_flags")))
		fmt.Println("")

		// Sort the globalFlags by flag name
		sort.Slice(h.globalFlags, func(i, j int) bool {
			return h.globalFlags[i].Flag < h.globalFlags[j].Flag
		})

		var (
			maxFlagLength  int
			maxAliasLength int
		)
		for _, flag := range h.globalFlags {
			if mfl := len(flag.Flag); mfl > maxFlagLength {
				maxFlagLength = mfl
			}
			if mal := len(flag.UsageAliases); mal > maxAliasLength {
				maxAliasLength = mal
			}
		}

		for _, flag := range h.globalFlags {
			h.printFlag(maxFlagLength, maxAliasLength, flag)
		}
	}
	return nil
}

func (h *Help) printFlag(maxFlagLength, maxAliasLength int, flag flagInfo) {
	aliases := flag.UsageAliases
	if aliases == "" {
		aliases = strings.Repeat(" ", maxAliasLength)
	}
	fstr := fmt.Sprintf(
		"  %-"+fmt.Sprint(maxFlagLength)+"s %-"+fmt.Sprint(maxAliasLength+2)+"s ",
		flag.Flag,
		aliases,
	)

	prefix := strings.Repeat(" ", maxFlagLength+maxAliasLength+7)
	// Try to translate the usage string - if it's an i18n key, translate it
	// Otherwise use as-is (for backwards compatibility)
	// Extract base usage (before "- default:") for translation
	var usage string
	baseUsage := flag.Usage
	defaultSuffix := ""
	if idx := strings.Index(flag.Usage, " - default:"); idx > 0 {
		baseUsage = flag.Usage[:idx]
		defaultSuffix = flag.Usage[idx:]
	}
	translatedBase := i18n.TD(baseUsage, baseUsage)
	if defaultSuffix != "" {
		// Translate the "default" word in the suffix
		// Pattern: " - default: %q"
		defaultWord := i18n.TD(i18np+".default", "default")
		// Replace "default" with translated version
		translatedSuffix := strings.Replace(defaultSuffix, "default", defaultWord, 1)
		usage = translatedBase + translatedSuffix
	} else {
		usage = translatedBase
	}
	desc := wordWrapWithPrefix(usage, prefix, 80)

	fmt.Println(fstr, desc)
}

func (h *Help) printSubcommand(maxNameLength int, name, description string) {
	re := regexp.MustCompile(`\x1B\[[0-9;]*[a-zA-Z]`)
	cleanNameLength := len(re.ReplaceAllString(name, ""))
	padding := maxNameLength - cleanNameLength + 2
	prefix := strings.Repeat(" ", maxNameLength+2)
	desc := wordWrapWithPrefix(description, prefix, 80)

	str := fmt.Sprintf("  %s%"+fmt.Sprint(padding)+"s  %s", ansicolor.Format(name, ansicolor.Bold), "", desc)
	fmt.Println(str)
}

func (h *Help) printBanner() error {
	name := h.style.Primary.String(h.info.Name)
	version := h.style.Version.String(h.info.Version)

	fmt.Println(" ", name, "-", version)

	copyr := h.info.copyright()
	if copyr != "" {
		fmt.Println(" ", h.style.Credits.String(copyr))
	}
	license := h.info.license()
	if license != "" {
		fmt.Println(" ", h.style.License.String(license))
	}
	description := h.info.description()
	if description != "" {
		fmt.Println(" ", h.style.Description.String(description))
	}
	fmt.Println("")
	for _, usage := range h.info.Usage {
		// Try to translate the entire usage string first
		translatedUsage := i18n.TD(usage, usage)

		// If translation didn't change the string, try to extract and translate
		// the i18n key part (typically at the end after the command name)
		if translatedUsage == usage {
			// Split by spaces to find potential i18n key at the end
			parts := strings.Fields(usage)
			if len(parts) > 0 {
				// Check if the last part looks like an i18n key (contains dots)
				lastPart := parts[len(parts)-1]
				if strings.Contains(lastPart, ".") {
					// Try to translate just the last part
					translatedLastPart := i18n.TD(lastPart, lastPart)
					if translatedLastPart != lastPart {
						// Translation found - reconstruct usage with translated part
						parts[len(parts)-1] = translatedLastPart
						translatedUsage = strings.Join(parts, " ")
					}
				}
			}
		}
		fmt.Println(" ", ansicolor.Format(translatedUsage, ansicolor.Bold))
	}
	return nil
}

func (h *Help) printInfo() error {
	if len(h.info.Info) > 0 {
		for _, info := range h.info.Info {
			fmt.Println("")
			// Translate info if it looks like an i18n key (starts with "com.github.")
			translatedInfo := info
			if strings.HasPrefix(info, "com.github.") {
				translatedInfo = i18n.T(info)
			}
			fmt.Println(h.style.Info.String(textfmt.WordWrapWithPrefixes(translatedInfo, 72, "    ", "  ")))
		}
	}
	return nil
}

func (h *Help) getCategoryDesc(category string) string {
	switch category {
	case "default":
		return ""
	default:
		if desc, ok := h.catdesc[strings.ToLower(category)]; ok {
			return " - " + h.style.Description.String(desc)
		}
		return ""
	}
}

type Info struct {
	Name           string
	Description    string
	Version        string
	CopyrightBy    string
	CopyrightSince int
	License        string
	Address        string
	Usage          []string
	Info           []string
}

func (i *Info) copyright() string {
	if i.CopyrightBy == "" {
		return ""
	}
	str := i18n.T(i18np + ".copyright")
	year := time.Now().Year()
	if i.CopyrightSince > 0 && year != i.CopyrightSince {
		str += " " + fmt.Sprint(i.CopyrightSince) + " - " + fmt.Sprint(year)
	} else {
		str += " " + fmt.Sprint(year)
	}
	return str + " " + i.CopyrightBy
}
func (i *Info) license() string {
	if i.License == "" {
		return ""
	}
	return i18n.T(i18np+".license") + i.License
}

func (i *Info) description() string {
	if i.Description == "" {
		return ""
	}
	return "\n  " + wordWrapWithPrefix(i.Description, "  ", 100)
}

func wordWrapWithPrefix(input, prefix string, lineLength int) string {
	var result strings.Builder
	var line strings.Builder
	words := strings.Fields(input)

	firstLine := true

	for _, word := range words {
		if line.Len()+len(word)+1 <= lineLength { // +1 for the space between words
			if line.Len() > 0 {
				line.WriteByte(' ')
			}
			line.WriteString(word)
		} else {
			if !firstLine {
				result.WriteString("\n" + prefix)
			}
			result.WriteString(line.String())
			line.Reset()
			line.WriteString(word)
			firstLine = false
		}
	}

	if line.Len() > 0 {
		if !firstLine {
			result.WriteString("\n" + prefix)
		}
		result.WriteString(line.String())
	}

	return result.String()
}

func getMaxNameLength(commands []commandInfo) int {
	re := regexp.MustCompile(`\x1B\[[0-9;]*[a-zA-Z]`)
	max := 0
	for _, cmd := range commands {
		// Strip ANSI escape codes
		cleanName := re.ReplaceAllString(cmd.name, "")
		// Count only printable runes
		count := 0
		for _, r := range cleanName {
			if unicode.IsPrint(r) {
				count++
			}
		}
		if count > max {
			max = count
		}
	}
	return max
}
