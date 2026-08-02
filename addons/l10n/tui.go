// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package l10n

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

// newProgram is an indirection point so tests can substitute a headless
// program (no real controlling terminal, which most CI/sandboxed test
// environments simply don't have) instead of the real tea.NewProgram -
// mirrors lib/taskrunner/runner.go's identical pattern.
var newProgram = func(m tea.Model, opts ...tea.ProgramOption) *tea.Program {
	return tea.NewProgram(m, opts...)
}

// l10nTUI builds the `l10n tui` subcommand: an interactive translation
// editor covering the same ground as report/list/translate, but as one
// continuously-updated Bubble Tea program instead of three one-shot CLI
// invocations. See tui_root.go for the tab layout and tui_dashboard.go/
// tui_browse.go/tui_translate.go for each tab.
func l10nTUI() *command.Command {
	cmd := command.New("tui",
		command.Config{
			Description: settings.String(l10np + ".tui.description"),
			Immediate:   true,
		})

	cmd.WithFlags(
		varflag.StringFunc("lang", "", l10np+".tui.flag_lang", "l"),
		// By default, the editor's worklist covers only application translations.
		// Use --with-deps to include dependency keys, same as report/list/translate.
		varflag.BoolFunc("with-deps", false, l10np+".tui.flag_with_deps"),
	)

	cmd.Do(func(sess *session.Context, args action.Args) error {
		langFlag := args.Flag("lang").String()
		withDeps := args.Flag("with-deps").Var().Bool()

		var lang language.Tag
		if langFlag != "" {
			parsed, err := language.Parse(langFlag)
			if err != nil {
				return fmt.Errorf("invalid language: %w", err)
			}
			lang = parsed
		}

		m := newTUIRootModel(sess, lang, withDeps)
		p := newProgram(m)
		_, err := p.Run()
		return err
	})

	return cmd
}
