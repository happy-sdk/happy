// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mkungla/happy"
)

func Help(ctx happy.Session, rootCmd, activeCmd happy.Command) bool {
	if rootCmd.Flag("help").Present() {
		Banner(ctx)
		settree := rootCmd.Flags().GetActiveSets()
		name := settree[len(settree)-1].Name()

		if name == "/" {
			help := helpGlobal{
				Commands: rootCmd.SubCommands(),
				Flags:    rootCmd.Flags().Flags(),
			}
			info := info{
				Slug:        ctx.Get("app.slug").String(),
				Description: ctx.Get("app.description").String(),
			}
			if err := help.Print(info); err != nil {
				ctx.Log().Error(err)
			}
		} else {
			helpCmd := helpCommand{}
			if err := helpCmd.Print(ctx, activeCmd); err != nil {
				ctx.Log().Error(err)
			}
		}
		return true
	}
	return false
}

func HelpCommand(ctx happy.Session, cmd happy.Command) {
	helpCmd := helpCommand{}
	helpCmd.Print(ctx, cmd)
}

type info struct {
	Slug        string
	Description string
}

// HelpGlobal used to show help for application.
type helpGlobal struct {
	cliTmplParser
	Info                info
	Commands            []happy.Command
	Flags               []happy.Flag
	PrimaryCommands     []happy.Command
	CommandsCategorized map[string][]happy.Command
}

var (
	helpGlobalTmpl = `{{if .Info.Description}}{{ .Info.Description }}{{end}}
 USAGE:
  {{ .Info.Slug }} command
  {{ .Info.Slug }} command [command-flags] [arguments]
  {{ .Info.Slug }} [global-flags] command [command-flags] [arguments]
  {{ .Info.Slug }} [global-flags] command ...subcommand [command-flags] [arguments]

 COMMANDS:{{ if .PrimaryCommands }}
 {{ range $cmd := .PrimaryCommands }}
 {{ $cmd.Slug.String | funcCmdName }}{{ $cmd.UsageDescription }}{{ end }}{{ end }}
{{ if .CommandsCategorized }}{{ range $cat, $cmds := .CommandsCategorized }}
 {{ $cat | funcCmdCategory }}{{ range $cmd := $cmds }}
 {{$cmd.Slug.String | funcCmdName }}{{ $cmd.UsageDescription }}{{ end }}
 {{ end }}{{ end }}

 GLOBAL FLAGS:{{ if .Flags }}{{ range $flag := .Flags }}{{ if not .Hidden }}
 {{funcFlagName $flag.Flag $flag.UsageAliases }} {{ $flag.Usage }}{{ end }}{{ end }}{{ end }}
`

	helpCommandTmpl = ` COMMAND: {{.Command.Slug }}
  {{ if gt (len .Command.UsageDescription) 0 }}{{funcTextBold .Command.UsageDescription}}
  {{ end }}
 USAGE:
  {{ funcTextBold .Usage }}
{{ if .Command.HasSubcommands }}
 {{ print "Subcommands" | funcCmdCategory }}
{{ range $cmd := .Command.SubCommands }}
{{ $cmd.Slug.String | funcCmdName }}{{ $cmd.UsageDescription }}{{ end }}
{{ end }}
{{ if gt .Command.Flags.Len 0 }} Accepts following flags:
{{ range $flag := .Flags }}{{ if not .Hidden }}
 {{funcFlagName $flag.Flag $flag.UsageAliases }} {{ $flag.Usage }}{{ end }}{{ end }}{{ end }}
{{ if gt (len .Command.Description) 0 }}
{{.Command.Description}}{{ end }}`
)

// Print application help.
func (h *helpGlobal) Print(info info) error {
	h.Info = info
	h.SetTemplate(helpGlobalTmpl)

	for _, cmd := range h.Commands {
		cat := cmd.Category()
		if cat == "" {
			h.PrimaryCommands = append(h.PrimaryCommands, cmd)
		} else {
			if h.CommandsCategorized == nil {
				h.CommandsCategorized = make(map[string][]happy.Command)
			}
			h.CommandsCategorized[cat] = append(h.CommandsCategorized[cat], cmd)
		}
	}
	err := h.ParseTmpl("help-global-tmpl", h, time.Duration(0))
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Fprintln(os.Stdout, h.String())
	return nil
}

// HelpCommand is used to display help for command.
type helpCommand struct {
	cliTmplParser
	Command happy.Command
	Usage   string
	Flags   []happy.Flag
}

// Print command help.
func (h *helpCommand) Print(ctx happy.Session, cmd happy.Command) error {
	if cmd == nil {
		return ErrCommand.WithText("attept to show help without providing command")
	}
	h.Command = cmd
	h.SetTemplate(helpCommandTmpl)
	usage := []string{ctx.Get("app.slug").String()}
	usage = append(usage, cmd.Parents()[1:]...)
	usage = append(usage, cmd.Slug().String())
	if h.Command.Flags().Len() > 0 {
		usage = append(usage, "[flags]")
	}

	if h.Command.HasSubcommands() {
		usage = append(usage, "[subcommands]")
	}

	if h.Command.Flags().AcceptsArgs() {
		usage = append(usage, "[args]")
	}
	h.Usage = strings.Join(usage, " ")
	h.Flags = append(h.Flags, h.Command.Flags().Flags()...)
	if h.Command.Parent() != nil {
		h.Flags = append(h.Flags, h.Command.Parent().Flags().Flags()...)
	}
	err := h.ParseTmpl("help-global-tmpl", h, time.Duration(0))
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Fprintln(os.Stdout, h.String())
	return nil
}
