// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"fmt"
	"strings"
	"time"
)

func (a *Application) help() error {
	fmt.Println(a.session.Get("app.name").String() + " " + a.session.Get("app.version").String() + "\n")
	if crby := a.session.Get("app.copyright.by").String(); crby != "" {
		since := a.session.Get("app.copyright.since").Int()
		year := time.Now().In(a.timeLocation).Year()
		yearstr := fmt.Sprint(year)
		if since > 0 && since < year {
			yearstr = fmt.Sprintf("%d - %d", since, year)
		}
		fmt.Printf("  Copyright © %s %s\n", crby, yearstr)
	}

	if lic := a.session.Get("app.license").String(); lic != "" {
		fmt.Printf("  %s\n", lic)
	}

	if desc := a.session.Get("app.description").String(); desc != "" {
		fmt.Printf("  %s\n", desc)
	}
	fmt.Println("")

	settree := a.rootCmd.flags.GetActiveSets()
	name := settree[len(settree)-1].Name()
	if name == "/" {
		if a.helpMsg != "" {
			fmt.Println(a.helpMsg + "\n")
		}

		if err := a.printHelp(a.session); err != nil {
			return err
		}
	} else {
		if err := printCommandHelp(a.session, a.activeCmd); err != nil {
			return err
		}
	}
	return nil
}

func (a *Application) printHelp(sess *Session) error {
	name := a.rootCmd.Name()

	fmt.Printf("USAGE:\n")
	fmt.Printf("  %s [global-flags]\n", name)
	fmt.Printf("  %s command\n", name)
	fmt.Printf("  %s command [command-flags] [arguments]\n", name)
	fmt.Printf("  %s [global-flags] command [command-flags] [arguments]\n", name)
	fmt.Printf("  %s [global-flags] command ...subcommand [command-flags] [arguments]\n\n", name)

	fmt.Printf("COMMANDS:\n")

	var (
		primaryCommands     []*Command
		commandsCategorized map[string][]*Command
	)
	for _, cmd := range a.activeCmd.getSubCommands() {
		cat := cmd.category
		if cat == "" {
			primaryCommands = append(primaryCommands, cmd)
		} else {
			if commandsCategorized == nil {
				commandsCategorized = make(map[string][]*Command)
			}
			commandsCategorized[cat] = append(commandsCategorized[cat], cmd)
		}
	}

	if len(primaryCommands) > 0 {
		for _, cmd := range primaryCommands {
			fmt.Printf("  %s %s\n", cmd.Name(), cmd.Usage())
		}
	}
	if len(commandsCategorized) > 0 {
		for cat, cmds := range commandsCategorized {
			fmt.Printf("  %s\n", strings.ToUpper(cat))
			for _, cmd := range cmds {
				fmt.Printf("  %-20s %s\n", cmd.Name(), cmd.Usage())
			}
		}
	}

	fmt.Printf("\nGLOBAL FLAGS:\n")
	for _, flag := range a.rootCmd.flags.Flags() {
		if !flag.Hidden() {
			fmt.Printf("  %-20s %s\n", flag.Flag(), flag.Usage())
		}
	}

	fmt.Println()
	return nil
}

func printCommandHelp(sess *Session, cmd *Command) error {
	fmt.Printf("COMMAND: %s\n\n", cmd.Name())
	if usage := cmd.Usage(); usage != "" {
		fmt.Println(usage)
	}
	if desc := cmd.Description(); desc != "" {
		fmt.Println(desc)
	}

	fmt.Printf("USAGE:\n")

	var parents string
	for _, c := range cmd.Parents() {
		parents += c + " "
	}

	fmt.Printf("  %s %s\n", parents, cmd.Name())
	fmt.Printf("  %s [global-flags] %s\n", parents, cmd.Name())
	fmt.Printf("  %s %s [command-flags] [arguments]\n", parents, cmd.Name())
	fmt.Printf("  %s [global-flags] %s [command-flags] [arguments]\n", parents, cmd.Name())
	fmt.Printf("  %s [global-flags] %s ...subcommand [command-flags] [arguments]\n\n", parents, cmd.Name())

	scmds := cmd.getSubCommands()
	if len(scmds) > 0 {
		fmt.Println("Subcommands")
		for _, subcmd := range scmds {
			fmt.Printf("  %-20s %s\n", subcmd.Name(), subcmd.Usage())
		}
	}

	flags := cmd.getFlags()

	if flags.Len() > 0 {
		fmt.Println("Accepts following flags:")
		for _, flag := range flags.Flags() {
			if !flag.Hidden() {
				fmt.Printf("  %-20s %s\n", flag.Flag(), flag.Usage())
			}
		}
	}

	fmt.Println("")
	return nil
}
