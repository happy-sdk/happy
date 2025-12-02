// SPDX-License-Identifier: Apache-2.0
//
// Showcase application demonstrating multiple Happy-SDK features:
// - Application settings (name, slug, i18n)
// - Built-in CLI commands (config, i18n) and global flags
// - Custom commands
// - Addons
// - Services
package main

import (
	"fmt"
	"time"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/examples/addons/hello"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/services"
	"github.com/happy-sdk/happy/sdk/services/service"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/text/language"
)

func main() {
	app := happy.New(&happy.Settings{
		Name:        "Showcase",
		Description: "Showcase example using most Happy-SDK features",
		Slug:        "showcase",
		CLI: happy.CliSettings{
			WithConfigCmd:   true,
			WithI18nCmd:     true,
			WithGlobalFlags: true,
			MainMinArgs:     0,
			MainMaxArgs:     2,
		},
		Logging: happy.LoggingSettings{
			Level: logging.LevelInfo,
		},
		Services: happy.ServicesSettings{
			// LoaderTimeout uses string duration in real settings; here we keep defaults.
			// RunCronOnStart: false,
		},
		Stats: happy.StatsSettings{
			// Enabled: false,
		},
	})

	// Root command just prints a simple banner and delegates to subcommands or flags.
	app.Do(func(sess *session.Context, args action.Args) error {
		loader := services.NewLoader(sess, "showcase-demo-service")
		<-loader.Load()
		if err := loader.Err(); err != nil {
			sess.Log().Error("Service loading failed:", "err", err)
		}
		sess.Log().Info("Welcome to the Happy-SDK showcase!")
		sess.Log().Info("Try `--help` to explore commands, or `showcase greet` / `showcase time`.")
		return nil
	})

	// Attach reusable Hello addon shared across examples.
	app.WithAddons(hello.Addon())

	// Attach a demo service.
	app.WithServices(demoService())

	// Add a couple of root-level commands.
	app.WithCommands(
		greetCommand(),
		timeCommand(),
	)

	// Add a custom flag to root command to show varflag usage.
	app.WithFlags(
		varflag.StringFunc("lang", "en", "Preferred language for greeting", "l"),
	)

	app.Run()
}

// greetCommand prints a greeting, demonstrating a simple custom command.
func greetCommand() *command.Command {
	cmd := command.New("greet",
		command.Config{
			Description: "Greet someone (optionally by name)",
			Immediate:   true,
		})

	cmd.WithFlags(
		varflag.StringFunc("name", "World", "Name to greet", "n"),
	)

	cmd.Do(func(sess *session.Context, args action.Args) error {
		name := args.Flag("name").String()
		langStr := sess.Opts().Get("app.main.flag.lang").String()
		if langStr == "" {
			langStr = "en"
		}
		lang, _ := language.Parse(langStr)

		msg := fmt.Sprintf("Hello, %s!", name)
		switch lang.String() {
		case "et":
			msg = fmt.Sprintf("Tere, %s!", name)
		}
		sess.Log().Info(msg)
		return nil
	})

	return cmd
}

// timeCommand demonstrates accessing context and printing current time.
func timeCommand() *command.Command {
	cmd := command.New("time",
		command.Config{
			Description: "Print current time from within a Happy session",
			Immediate:   true,
		})

	cmd.Do(func(sess *session.Context, args action.Args) error {
		now := time.Now().Format(time.RFC3339)
		sess.Log().Info("Current time", "time", now)
		return nil
	})

	return cmd
}

// demoService creates a tiny service that logs a tick on start and stop.
func demoService() *services.Service {
	svc := services.New(service.Config{
		Name: "Showcase Demo Service",
	})

	svc.OnRegister(func(sess *session.Context) error {
		sess.Log().Info("demo service registered")
		return nil
	})

	svc.OnStart(func(sess *session.Context) error {
		sess.Log().Info("demo service started")
		return nil
	})

	svc.OnStop(func(sess *session.Context, err error) error {
		if err != nil {
			sess.Log().Error("demo service stopped with error", "error", err)
			return nil
		}
		sess.Log().Info("demo service stopped")
		return nil
	})

	return svc
}
