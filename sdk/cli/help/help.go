// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package help

import (
	"fmt"
	"sort"
	"strings"

	"github.com/happy-sdk/happy/pkg/cli/ansicolor"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
)

type Help struct {
	fg, bg      ansicolor.Color
	info        *Info
	cmds        map[string][]commandInfo
	globalFlags []flagInfo
	flags       []flagInfo
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

func New(info Info) *Help {
	return &Help{
		fg:   ansicolor.FgRGB(255, 237, 86),
		info: &info,
		cmds: make(map[string][]commandInfo),
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

func (h *Help) AddGlobalFlags(f varflag.Flags) {
	if f == nil {
		return
	}
	for _, flag := range f.Flags() {
		h.globalFlags = append(h.globalFlags, flagInfo{
			Flag:         flag.Flag(),
			UsageAliases: flag.UsageAliases(),
			Usage:        flag.Usage(),
		})
	}
}

func (h *Help) AddCommandFlags(f varflag.Flags) {
	if f == nil {
		return
	}
	for _, flag := range f.Flags() {
		h.flags = append(h.flags, flagInfo{
			Flag:         flag.Flag(),
			UsageAliases: flag.UsageAliases(),
			Usage:        flag.Usage(),
		})
	}
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
		fmt.Println("COMMANDS:")
		fmt.Println("")
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

			for _, cmd := range commands {
				h.printSubcommand(maxNameLength, cmd.name, cmd.description)
			}
		}

		// Print other categories
		for _, category := range categories {
			fmt.Println("")
			fmt.Println(" ", ansicolor.Text(strings.ToUpper(category), h.fg, h.bg, 0))
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
		fmt.Println("FLAGS:")
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
	return nil
}
func (h *Help) printGlobalFlags() error {
	if len(h.globalFlags) > 0 {
		fmt.Println("")
		fmt.Println("GLOBAL FLAGS:")
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
		"  %-"+fmt.Sprint(maxFlagLength+12)+"s %-"+fmt.Sprint(maxAliasLength+2)+"s ",
		ansicolor.Text(flag.Flag, ansicolor.FgWhite, 0, 1),
		ansicolor.Text(aliases, ansicolor.FgWhite, 0, 1),
	)

	prefix := strings.Repeat(" ", maxFlagLength+maxAliasLength+6)
	desc := wordWrapWithPrefix(flag.Usage, prefix, 80)

	fmt.Println(fstr, desc)
}

func (h *Help) printSubcommand(maxNameLength int, name, description string) {
	prefix := strings.Repeat(" ", maxNameLength)
	desc := wordWrapWithPrefix(description, prefix, 80)

	str := fmt.Sprintf("  %-"+fmt.Sprint(maxNameLength)+"s  %s", ansicolor.Text(name, ansicolor.FgWhite, 0, 1), desc)

	fmt.Println(str)
}

func (h *Help) printBanner() error {
	name := ansicolor.Text(h.info.Name, h.fg, 0, 1)
	version := ansicolor.Text(h.info.Version, ansicolor.FgWhite, 0, 2)

	fmt.Println(" ", name, "-", version)

	copyr := h.info.copyright()
	if copyr != "" {
		fmt.Println(" ", ansicolor.Text(copyr, h.fg, 0, 0))
	}
	license := h.info.license()
	if license != "" {
		fmt.Println(" ", ansicolor.Text(license, ansicolor.FgWhite, 0, 2))
	}
	description := h.info.description()
	if description != "" {
		fmt.Println(" ", ansicolor.Text(description, ansicolor.FgWhite, 0, 2))
	}
	fmt.Println("")
	for _, usage := range h.info.Usage {
		fmt.Println(" ", ansicolor.Text(usage, ansicolor.FgWhite, 0, 0))
	}
	return nil
}

func (h *Help) printInfo() error {
	if len(h.info.Info) > 0 {
		fmt.Println("")
		for _, info := range h.info.Info {
			fmt.Println(" ", ansicolor.Text(info, ansicolor.FgWhite, 0, 2))
		}
	}
	return nil
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
	return "Copyright © " + i.CopyrightBy
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
