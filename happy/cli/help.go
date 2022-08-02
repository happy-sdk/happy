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
	"github.com/mkungla/happy/config"
	"github.com/mkungla/varflag/v6"
)

func Help(a happy.Application) {
	if a.Flag("help").Present() {
		Banner(a.Session())
		settree := a.Flags().GetActiveSetTree()
		name := settree[len(settree)-1].Name()

		if name == "/" {
			help := helpGlobal{
				Commands: a.Commands(),
				Flags:    a.Flags().Flags(),
			}
			help.Print(a.Config())
		} else {
			helpCmd := helpCommand{}
			helpCmd.Print(a.Session(), a.Command())
		}
	}

	// settree := a.Flags().GetActiveSetTree()
	// a.Session().Log().NotImplemented("help not implemented")
}

func HelpCommand(a happy.Session) {
	fmt.Println("HELP COMMAND")
	// settree := a.Flags().GetActiveSetTree()
	// a.Session().Log().NotImplemented("help not implemented")
}

// HelpGlobal used to show help for application.
type helpGlobal struct {
	cliTmplParser
	Config              config.Config
	Commands            map[string]happy.Command
	Flags               []varflag.Flag
	PrimaryCommands     []happy.Command
	CommandsCategorized map[string][]happy.Command
}

var (
	helpGlobalTmpl = `{{if .Config.Description}}{{ .Config.Description }}{{end}}

 Usage:
  {{ .Config.Slug }} command
  {{ .Config.Slug }} command [command-flags] [arguments]
  {{ .Config.Slug }} [global-flags] command [command-flags] [arguments]
  {{ .Config.Slug }} [global-flags] command ...subcommand [command-flags] [arguments]

 The commands are:{{ if .PrimaryCommands }}
 {{ range $cmdObj := .PrimaryCommands }}
 {{ $cmdObj.String | funcCmdName }}{{ $cmdObj.ShortDesc }}{{ end }}{{ end }}
{{ if .CommandsCategorized }}{{ range $cat, $cmds := .CommandsCategorized }}
 {{ $cat | funcCmdCategory }}
 {{ range $cmdObj := $cmds }}
 {{$cmdObj.String | funcCmdName }}{{ $cmdObj.ShortDesc }}{{ end }}
 {{ end }}{{ end }}

 The global flags are:{{ if .Flags }}
 {{ range $flag := .Flags }}{{ if not .IsHidden }}
 {{funcFlagName $flag.Flag $flag.AliasesString }} {{ $flag.Usage }}{{ end }}{{ end }}{{ end }}
`

	helpCommandTmpl = `{{ if gt (len .Command.ShortDesc) 0 }}{{funcTextBold .Command.ShortDesc}}
  {{ end }}
 Usage:
   {{ if .Command.Usage }}{{ funcTextBold .Command.Usage }}{{else}}{{ funcTextBold .Usage }}{{ end }}
{{ if .Command.HasSubcommands }}
 {{ print "Subcommands" | funcCmdCategory }}
{{ range $cmdObj := .Command.Subcommands }}
{{ $cmdObj.String | funcCmdName }}{{ $cmdObj.ShortDesc }}{{ end }}
{{ end }}
{{ if .Command.AcceptsFlags }} Accepts following flags:
{{ range $flag := .Flags }}{{ if not .IsHidden }}
 {{funcFlagName $flag.Flag $flag.AliasesString }} {{ $flag.Usage }}{{ end }}{{ end }}{{ end }}
{{ if gt (len .Command.ShortDesc) 0 }}
{{.Command.ShortDesc}}{{ end }}
{{ if gt (len .Command.LongDesc) 0 }}
{{.Command.LongDesc}}{{ end }}
`
)

// Print application help.
func (h *helpGlobal) Print(cnf config.Config) error {
	h.Config = cnf
	h.SetTemplate(helpGlobalTmpl)

	for _, cmdObj := range h.Commands {
		if cmdObj.Category() == "" {
			h.PrimaryCommands = append(h.PrimaryCommands, cmdObj)
		} else {
			if h.CommandsCategorized == nil {
				h.CommandsCategorized = make(map[string][]happy.Command)
			}
			h.CommandsCategorized[cmdObj.Category()] = append(h.CommandsCategorized[cmdObj.Category()],
				cmdObj)
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
	Flags   []varflag.Flag
}

// Print command help.
func (h *helpCommand) Print(ctx happy.Session, cmd happy.Command) error {
	h.Command = cmd
	h.SetTemplate(helpCommandTmpl)
	usage := []string{ctx.Get("app.slug").String()}
	usage = append(usage, cmd.Parents()...)
	usage = append(usage, cmd.String())
	if h.Command.AcceptsFlags() {
		usage = append(usage, "[flags]")
	}

	if h.Command.HasSubcommands() {
		usage = append(usage, "[subcommands]")
	}

	if h.Command.AcceptsArgs() {
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
