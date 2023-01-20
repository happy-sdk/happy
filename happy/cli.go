// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/mkungla/happy/pkg/hlog"
	"github.com/mkungla/happy/pkg/varflag"
)

type help struct {
	padding uint8
	colored bool
}

func (a *Application) cliCmdHelp(cmd *Command) {
	help := &help{
		padding: 2,
		colored: a.rootCmd.flag("no-color").Var().Bool(),
	}
	help.banner(a)
}

func (h *help) banner(a *Application) {
	h.printColoredln(fmt.Sprintf("  %s", a.session.Get("app.name")))
}

func (h *help) println(line string) {
	fmt.Println(line)
}
func (h *help) printColoredln(line string) {
	if h.colored {
		h.println(line)
		return
	}
	fmt.Println(line)
}

// // DEPRECATED
type view struct {
	banner cliTmplParser
	Info   struct {
		Name           string
		CopyrightBy    string
		CopyrightSince int
		License        string
		Version        string
		Description    string
	}
}

func (a *Application) clihelp() error {
	a.session.Log().Deprecated("clihelp is deprecated")
	view := &view{}
	view.Info.Name = a.session.Get("app.name").String()
	view.Info.Version = a.session.Get("app.version").String()
	view.Info.CopyrightBy = a.session.Get("app.copyright.by").String()
	view.Info.CopyrightSince = a.session.Get("app.copyright.since").Int()
	view.Info.License = a.session.Get("app.license").String()
	view.Info.Description = a.session.Get("app.description").String()

	view.banner.setTemplate(` {{ .Name }}{{ if .CopyrightBy }}
  Copyright © {{ if .CopyrightSince }}{{ .CopyrightSince }} {{ end }}{{ if (gt funcYear  .CopyrightSince) }}- {{ funcYear }} {{ end }}{{ .CopyrightBy }}. All rights reserved.{{end}}{{ if .License }}
  License:      {{ .License }}{{ end }}{{ if .Version }}
  {{ .Version }}{{end}}
  {{ if .Description }}
  {{ .Description }}{{end}}
  `)

	if err := view.printBanner(); err != nil {
		return err
	}

	settree := a.rootCmd.flags.GetActiveSets()
	name := settree[len(settree)-1].Name()
	if name == "/" {
		help := helpGlobal{
			Commands: a.rootCmd.subCommands,
			Flags:    a.rootCmd.flags.Flags(),
		}
		if err := help.print(); err != nil {
			return err
		}
	} else {
		helpCmd := helpCommand{}
		if err := helpCmd.print(a.session, a.activeCmd); err != nil {
			return err
		}
	}
	return nil
}

func (h *view) printBanner() error {
	if err := h.banner.parseTmpl("header-tmpl", h.Info, 0); err != nil {
		return err
	}
	fmt.Fprintln(
		os.Stdout,
		hlog.Colorize(
			h.banner.buffer.String(),
			hlog.FgYellow,
			0,
			0,
		),
	)
	return nil
}

// TmplParser enables to parse templates for cli apps.
type cliTmplParser struct {
	tmpl   string
	buffer bytes.Buffer
	t      *template.Template
}

// SetTemplate sets template to be parsed.
func (t *cliTmplParser) setTemplate(tmpl string) {
	t.tmpl = tmpl
}

// ParseTmpl parses template for cli application
// arg name is template name, arg info is common passed to template
// and elapsed is time duration used by specific type of templates and can usually set to "0".
func (t *cliTmplParser) parseTmpl(name string, h interface{}, elapsed time.Duration) error {
	t.t = template.New(name)
	t.t.Funcs(template.FuncMap{
		"funcTextBold":    t.textBold,
		"funcCmdCategory": t.cmdCategory,
		"funcCmdName":     t.cmdName,
		"funcFlagName":    t.flagName,
		"funcDate":        t.dateOnly,
		"funcYear":        t.year,
		"funcElapsed":     func() string { return elapsed.String() },
	})
	tmpl, err := t.t.Parse(t.tmpl)
	if err != nil {
		return err
	}
	err = tmpl.Execute(&t.buffer, h)
	if err != nil {
		return err
	}
	return nil
}

func (t *cliTmplParser) cmdCategory(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s)
}

func (t *cliTmplParser) cmdName(s string) string {
	if s == "" {
		return s
	}
	return fmt.Sprintf("\033[1m %-20s\033[0m", s)
}

func (t *cliTmplParser) flagName(s string, a string) string {
	if s == "" {
		return s
	}
	if len(a) > 0 {
		s += ", " + a
	}
	return fmt.Sprintf("%-25s", s)
}

func (t *cliTmplParser) textBold(s string) string {
	if s == "" {
		return s
	}
	return fmt.Sprintf("\033[1m%s\033[0m", s)
}

func (t *cliTmplParser) dateOnly(ts time.Time) string {
	y, m, d := ts.Date()
	return fmt.Sprintf("%.2d-%.2d-%d", d, m, y)
}

func (t *cliTmplParser) year() int {
	return time.Now().Year()
}

// HelpGlobal used to show help for application.
type helpGlobal struct {
	cliTmplParser
	Name                string
	Commands            map[string]*Command
	Flags               []varflag.Flag
	PrimaryCommands     []*Command
	CommandsCategorized map[string][]*Command
}

// Print application help.
func (h *helpGlobal) print() error {
	h.Name = filepath.Base(os.Args[0])
	h.setTemplate(helpGlobalTmpl)

	for _, cmd := range h.Commands {
		cat := cmd.category
		if cat == "" {
			h.PrimaryCommands = append(h.PrimaryCommands, cmd)
		} else {
			if h.CommandsCategorized == nil {
				h.CommandsCategorized = make(map[string][]*Command)
			}
			h.CommandsCategorized[cat] = append(h.CommandsCategorized[cat], cmd)
		}
	}
	err := h.parseTmpl("help-global-tmpl", h, time.Duration(0))
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Fprintln(os.Stdout, h.buffer.String())
	return nil
}

// HelpCommand is used to display help for command.

type helpCommand struct {
	cliTmplParser
	Command *command
	Usage   string
	Flags   []varflag.Flag
}

type command struct {
	Name        string
	Usage       string
	SubCommands []command
	Flags       varflag.Flags
	Description string
}

func (h *helpCommand) print(sess *Session, cmd *Command) error {
	sess.Log().Deprecated("helpCommand.print is deprecated")

	h.Command = &command{
		Name:        cmd.name,
		Usage:       cmd.Usage(),
		Flags:       cmd.flags,
		Description: cmd.desc,
	}
	h.setTemplate(helpCommandTmpl)
	usage := []string{""}
	// usage := []string{filepath.Base(os.Args[0])}
	usage = append(usage, cmd.parents...)
	usage = append(usage, cmd.name)
	if cmd.flags.Len() > 0 {
		usage = append(usage, "[flags]")
	}

	if cmd.subCommands != nil {
		usage = append(usage, "[subcommands]")
		for _, subcmd := range cmd.subCommands {
			subCmd := command{
				Name:        subcmd.name,
				Usage:       subcmd.Usage(),
				Flags:       subcmd.flags,
				Description: subcmd.desc,
			}
			h.Command.SubCommands = append(h.Command.SubCommands, subCmd)
		}
	}

	if cmd.flags.AcceptsArgs() {
		usage = append(usage, "[args]")
	}
	h.Usage = strings.Join(usage, " ")
	h.Flags = append(h.Flags, cmd.flags.Flags()...)
	if cmd.parent != nil {
		h.Flags = append(h.Flags, cmd.parent.flags.Flags()...)
	}
	err := h.parseTmpl("help-global-tmpl", h, time.Duration(0))
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Fprintln(os.Stdout, h.buffer.String())
	return nil
}

var (
	helpGlobalTmpl = `
 USAGE:
  {{ .Name }} command
  {{ .Name }} command [command-flags] [arguments]
  {{ .Name }} [global-flags] command [command-flags] [arguments]
  {{ .Name }} [global-flags] command ...subcommand [command-flags] [arguments]

 COMMANDS:{{ if .PrimaryCommands }}
 {{ range $cmd := .PrimaryCommands }}
 {{ $cmd.Name | funcCmdName }}{{ $cmd.Usage }}{{ end }}{{ end }}
{{ if .CommandsCategorized }}{{ range $cat, $cmds := .CommandsCategorized }}
 {{ $cat | funcCmdCategory }}{{ range $cmd := $cmds }}
 {{$cmd.Name | funcCmdName }}{{ $cmd.Usage }}{{ end }}
 {{ end }}{{ end }}

 GLOBAL FLAGS:{{ if .Flags }}{{ range $flag := .Flags }}{{ if not .Hidden }}
 {{funcFlagName $flag.Flag $flag.UsageAliases }} {{ $flag.Usage }}{{ end }}{{ end }}{{ end }}
`

	helpCommandTmpl = ` COMMAND: {{.Command.Name }}
  {{ if gt (len .Command.Usage) 0 }}{{funcTextBold .Command.Usage}}
  {{ end }}
 USAGE:
  {{ funcTextBold .Usage }}
{{ if .Command.SubCommands }}
 {{ print "Subcommands" | funcCmdCategory }}
{{ range $cmd := .Command.SubCommands }}
{{ $cmd.Name | funcCmdName }}{{ $cmd.Usage }}{{ end }}
{{ end }}
{{ if gt .Command.Flags.Len 0 }} Accepts following flags:
{{ range $flag := .Flags }}{{ if not .Hidden }}
 {{funcFlagName $flag.Flag $flag.UsageAliases }} {{ $flag.Usage }}{{ end }}{{ end }}{{ end }}
{{ if gt (len .Command.Description) 0 }}
{{.Command.Description}}{{ end }}`
)
