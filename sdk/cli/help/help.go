// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package help

import (
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	"github.com/happy-sdk/happy/pkg/cli/ansicolor"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
)

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
		fmt.Println(h.style.Primary.String(" COMMANDS:"))
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
			fmt.Println(" ", h.style.Category.String(strings.ToUpper(category))+h.getCategoryDesc(category))
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
		fmt.Println(h.style.Primary.String(" FLAGS:"))
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
		fmt.Println(h.style.Primary.String(" SHARED FLAGS:"))
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
		fmt.Println(h.style.Primary.String(" GLOBAL FLAGS:"))
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
	desc := wordWrapWithPrefix(flag.Usage, prefix, 80)

	fmt.Println(fstr, desc)
}

func (h *Help) printSubcommand(maxNameLength int, name, description string) {
	prefix := strings.Repeat(" ", maxNameLength+2)
	desc := wordWrapWithPrefix(description, prefix, 80)

	str := fmt.Sprintf("  %-"+fmt.Sprint(maxNameLength+10)+"s  %s", ansicolor.Format(name, ansicolor.Bold), desc)
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
		fmt.Println(" ", ansicolor.Format(usage, ansicolor.Bold))
	}
	return nil
}

func (h *Help) printInfo() error {
	if len(h.info.Info) > 0 {
		fmt.Println("")
		for _, info := range h.info.Info {

			fmt.Println(textfmt.WordWrapWithPrefixes(h.style.Info.String(info), 72, "    ", "  "))
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
	str := "Copyright ©"
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
	return "License: " + i.License
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
	max := 0
	for _, cmd := range commands {
		if len(cmd.name) > max {
			max = len(cmd.name)
		}
	}
	return max
}
