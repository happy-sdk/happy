// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/branding"
	"github.com/happy-sdk/happy/pkg/i18n"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/tui/ansicolor"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/addon"
	"github.com/happy-sdk/happy/sdk/app/engine"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/cli/help"
	"github.com/happy-sdk/happy/sdk/events"
	"github.com/happy-sdk/happy/sdk/instance"
	"github.com/happy-sdk/happy/sdk/internal"
	"github.com/happy-sdk/happy/sdk/services"
	"github.com/happy-sdk/happy/sdk/session"
)

var (
	Error          = errors.New("runtime error")
	ErrExitSuccess = errors.New("exit success")
)

const i18np = "com.github.happy-sdk.happy.sdk.app.internal.application"

type Runtime struct {
	startedAt time.Time
	sess      *session.Context
	cmd       *command.Cmd
	inst      *instance.Instance
	brand     *branding.Brand

	exitFuncs []func(sess *session.Context, code int) error
	exitCh    chan ShutDown

	setupAction  action.Action
	beforeAlways action.WithArgs
	tickAction   action.Tick
	tockAction   action.Tock

	sessionReadyEvent events.Event
	evch              chan events.Event
	engine            *engine.Engine

	tmplogger logging.Logger
	execlvl   logging.Level

	initStartedAt time.Time
	initTook      time.Duration

	svcs []*services.Service

	addonm *addon.Manager
}

func (rt *Runtime) WidthBeforeAlways(a action.WithArgs) error {
	if rt.beforeAlways != nil {
		return fmt.Errorf("before always action already set")
	}
	rt.beforeAlways = a
	return nil
}

func (rt *Runtime) WithExitFunc(exitFunc func(sess *session.Context, code int) error) {
	rt.exitFuncs = append(rt.exitFuncs, exitFunc)
}

func (rt *Runtime) SetLogger(l logging.Logger) {
	rt.tmplogger = l
}

func (rt *Runtime) SetMain(cmd *command.Cmd) {
	rt.cmd = cmd
}

func (rt *Runtime) SetSession(sess *session.Context) {
	rt.sess = sess
	rt.tmplogger = nil
}

func (rt *Runtime) SetAddonManager(addonm *addon.Manager) {
	rt.addonm = addonm
}

func (rt *Runtime) SetBrand(b *branding.Brand) {
	rt.brand = b
}

func (rt *Runtime) SetSessionReady(ch chan events.Event, e events.Event) {
	rt.sessionReadyEvent = e
	rt.evch = ch
}

func (rt *Runtime) SetMainTick(a action.Tick) {
	rt.tickAction = a
}

func (rt *Runtime) SetMainTock(a action.Tock) {
	rt.tockAction = a
}

func (rt *Runtime) SetSetup(setup action.Action) {
	rt.setupAction = setup
}

func (rt *Runtime) InitStats(startedAt time.Time, took time.Duration) {
	rt.initStartedAt = startedAt
	rt.initTook = took
}

func (rt *Runtime) AddServices(svcs []*services.Service) {
	rt.svcs = append(rt.svcs, svcs...)
}

func (rt *Runtime) boot() (err error) {
	// Boot the application
	bootedAt := time.Now()

	defer func() {
		if r := recover(); r != nil {
			rt.recover(r, "panic at application boot")
		}
	}()
	// Run setup action?
	if rt.sess.Get("app.firstrun").Bool() && rt.setupAction != nil {
		if err := rt.setupAction(rt.sess); err != nil {
			return fmt.Errorf("failed to setup application: %w", err)
		}
		rt.setupAction = nil
	}

	// Run immediate command?
	if rt.cmd.IsImmediate() {
		internal.Log(rt.sess.Log(), "skip application boot for immediate command")
		if err := rt.executeBeforeActions(); err != nil {
			return err
		}
		rt.sess.Dispatch(rt.sessionReadyEvent)
		return nil
	}

	rt.sess.Log().LogDepth(1, logging.LevelDebug, i18n.PTD(i18np, "booting_application", "booting application"))

	// Create a new instance
	if rt.inst, err = instance.New(rt.sess); err != nil {
		return fmt.Errorf("failed to boot instance: %w", err)
	}
	rt.exitFuncs = append(rt.exitFuncs, func(sess *session.Context, code int) error {
		return rt.inst.Dispose()
	})

	// Create and start app engine
	{
		var (
			tickAction action.Tick
			tockAction action.Tock
		)
		if rt.cmd.IsRoot() {
			tickAction = rt.tickAction
			tockAction = rt.tockAction
		}

		rt.engine = engine.New(rt.evch, tickAction, tockAction)

		// register services
		for _, ev := range rt.addonm.Events() {
			if err := rt.engine.RegisterEvent(ev); err != nil {
				return fmt.Errorf("failed to register event: %w", err)
			}
		}
		for _, svc := range rt.svcs {
			if err := rt.engine.RegisterService(rt.sess, svc); err != nil {
				return fmt.Errorf("failed to register service: %w", err)
			}
		}

		// call addon register functions
		if err := rt.addonm.Register(rt.sess); err != nil {
			return fmt.Errorf("failed to register addons: %w", err)
		}

		rt.svcs = nil
		if err := rt.engine.Start(rt.sess); err != nil {
			return fmt.Errorf("%w: failed to start engine: %w", Error, err)
		}
	}

	if err := rt.executeBeforeActions(); err != nil {
		return err
	}

	if err := rt.engine.Stats().Set("init.at", rt.sess.Time(rt.initStartedAt).Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("failed to set app initialized at: %w", err)
	}
	if err := rt.engine.Stats().Set("init.took", rt.initTook.String()); err != nil {
		return fmt.Errorf("failed to set app initialization took: %w", err)
	}

	if err := rt.engine.Stats().Set("boot.at", rt.sess.Time(bootedAt).Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("failed to set app started at: %w", err)
	}
	bootTook := time.Since(bootedAt).String()
	if err := rt.engine.Stats().Set("boot.took", bootTook); err != nil {
		return fmt.Errorf("failed to set app started at: %w", err)
	}

	rt.sess.Dispatch(rt.sessionReadyEvent)
	rt.sessionReadyEvent = nil
	if rt.execlvl == logging.LevelQuiet || rt.execlvl < logging.LevelDebug {
		rt.sess.Log().LogDepth(1, logging.LevelDebug, i18n.PTD(i18np, "application_booted", "application booted"), slog.String("took", bootTook))
	}
	return nil
}

func (rt *Runtime) Start() {
	if err := rt.boot(); err != nil {
		if errors.Is(err, ErrExitSuccess) {
			rt.Exit(0)
			return
		}

		rt.sess.Log().Debug("failed to boot application", slog.String("err", err.Error()))
		rt.sess.Log().Error(err.Error())
		rt.Exit(1)
		return
	}

	rt.startedAt = rt.sess.Time(time.Now())
	if rt.execlvl == logging.LevelQuiet || rt.execlvl < logging.LevelDebug {
		rt.sess.Log().LogDepth(1, logging.LevelDebug, i18n.PTD(i18np, "starting_application", "starting application"), slog.Time("started.at", rt.startedAt))
	}

	if rt.engine != nil {
		if err := rt.engine.Stats().Set("app.started.at", rt.startedAt.Format(time.RFC3339)); err != nil {
			rt.sess.Log().Error("failed to set app started at", slog.String("err", err.Error()))
		}
	}

	<-rt.sess.Ready()

	if err := rt.sess.Err(); err != nil {
		rt.sess.Log().Error("session error", slog.String("err", err.Error()))
		rt.Exit(1)
		return
	}

	err := rt.executeDoAction()
	defer func() {
		if r := recover(); r != nil {
			rt.recover(r, "shutdown failed")
		}
	}()

	if rt.engine != nil {
		if engErr := rt.engine.Stop(rt.sess); engErr != nil {
			rt.sess.Log().Error("failed to stop engine", slog.String("err", engErr.Error()))
		}
	}

	// Only clear the reference since session owns the channel
	rt.evch = nil

	canRecover := rt.sess.CanRecover(err)

	if !canRecover {
		if e := rt.cmd.ExecAfterFailure(rt.sess, err); e != nil {
			rt.sess.Log().Error(e.Error(), slog.String("action", "AfterFailure"))
			rt.Exit(1)
			return
		}
	} else {
		if e := rt.cmd.ExecAfterSuccess(rt.sess); e != nil {
			rt.sess.Log().Error(e.Error(), slog.String("action", "AfterSuccess"))
			rt.Exit(1)
			return
		}
	}

	if canRecover {
		err = nil
	}
	if e := rt.cmd.ExecAfterAlways(rt.sess, err); e != nil {
		rt.sess.Log().Error(e.Error(), slog.String("action", "AfterAlways"))
		rt.Exit(1)
		return
	}
	if rt.execlvl < logging.LevelQuiet {
		rt.sess.Log().SetLevel(rt.execlvl)
	}

	if err != nil {
		rt.Exit(1)
		return
	}
	rt.Exit(0)
}

func (rt *Runtime) recover(r any, msg string) {
	// Log the panic message
	var errMessage string
	if err, ok := r.(error); ok {
		errMessage = err.Error()
	} else {
		errMessage = fmt.Sprintf("%v", r)
	}

	stack := debug.Stack()

	// Obtain and log the stack trace
	stackTrace := string(stack)

	rt.log(3, logging.LevelBUG, fmt.Sprintf("panic: %s (recovered)", msg),
		slog.String("msg", errMessage),
	)
	rt.log(3, logging.LevelAlways, stackTrace)
	rt.Exit(1)
}

func (rt *Runtime) executeBeforeActions() error {
	defer func() {
		if r := recover(); r != nil {
			rt.recover(r, "before actions failed")
		}
	}()
	if rt.execlvl < logging.LevelQuiet {
		execlvl := rt.sess.Log().Level()
		rt.sess.Log().SetLevel(rt.execlvl)
		rt.execlvl = execlvl
	}

	internal.Log(rt.sess.Log(), "executing before actions")

	if rt.sess.Log().Level() < logging.LevelDebug {
		// Settings table
		settingstbl := textfmt.Table{
			Title:      "Application Settings",
			WithHeader: true,
		}
		settingstbl.AddRow("KEY", "KIND", "IS SET", "MUTABILITY", "VALUE", "DEFAULT")
		for s := range rt.sess.Settings().All() {
			var defval string
			if s.Mutability() != settings.SettingImmutable && s.Default().String() != s.Value().String() {
				defval = s.Default().String()
			}
			settingstbl.AddRow(s.Key(), s.Kind().String(), fmt.Sprint(s.IsSet()), fmt.Sprint(s.Mutability()), s.Value().String(), defval)
		}
		rt.sess.Log().Println(settingstbl.String())

		// Options
		optstbl := textfmt.Table{}
		rt.sess.Opts().Range(func(opt options.Option) bool {
			optstbl.AddRow(opt.Key(), opt.Value().String())
			return true
		})
		rt.sess.Log().Println(optstbl.String())
	}

	if rt.beforeAlways != nil && !rt.cmd.SkipSharedBeforeAction() {
		timer := time.Now()
		internal.Log(rt.sess.Log(), "executing before always")
		args := action.NewArgs(rt.cmd.GetFlagSet())
		if err := rt.beforeAlways(rt.sess, args); err != nil {
			internal.Log(rt.sess.Log(), "failed to execute before always action", slog.String("err", err.Error()))
			return err
		}
		internal.Log(rt.sess.Log(), "before always action took", slog.String("took", time.Since(timer).String()))
	}

	timer := time.Now()
	if err := rt.cmd.ExecBefore(rt.sess); err != nil {
		internal.Log(rt.sess.Log(), "failed to execute before action", slog.String("err", err.Error()))
		if !rt.cmd.Flag("help").Present() {
			return err
		}
	}
	internal.Log(rt.sess.Log(), "before action took", slog.String("took", time.Since(timer).String()))

	if rt.cmd.Flag("help").Present() || rt.cmd.IsWrapper() {
		if err := rt.showHelp(); err != nil {
			return err
		}
		return ErrExitSuccess
	}

	return nil
}

func (rt *Runtime) showHelp() error {
	theme := rt.brand.ANSI()

	hideDisabled := rt.sess.Get("app.cli.hide_disabled_commands").Bool()

	helpInfo := help.Info{
		Name:           rt.sess.Get("app.name").String(),
		Description:    rt.sess.Get("app.description").String(),
		Version:        rt.sess.Get("app.version").String(),
		CopyrightBy:    rt.sess.Get("app.copyright_by").String(),
		CopyrightSince: rt.sess.Get("app.copyright_since").Int(),
		License:        rt.sess.Get("app.license").String(),
		Address:        rt.sess.Get("app.address").String(),
		Usage:          rt.cmd.Usage(),
		Info:           rt.cmd.Info(),
	}

	if rt.cmd.Disabled() {
		if !hideDisabled {
			err := rt.cmd.Err()
			if err == nil {
				err = fmt.Errorf("command disabled in current context")
			}
			helpInfo.Usage = []string{
				ansicolor.Format(helpInfo.Usage[0], ansicolor.Strike),
				ansicolor.Text(
					err.Error(),
					theme.Warning,
					ansicolor.Color{},
					0,
				),
			}
		} else {
			parent := rt.cmd.Parent()
			var parrentName string
			if parent != nil {
				parrentName = parent.Name()
				if parent.Disabled() {
					parrentName = rt.sess.Get("app.slug").String()
				}
			} else {
				parrentName = rt.sess.Get("app.slug").String()
			}
			rt.sess.Log().Error(varflag.ErrInvalidArguments.Error())
			return varflag.ErrInvalidCommandOrArgs.WithArgs(
				parrentName,
				rt.cmd.Name(),
			)
		}

	}

	desc := rt.cmd.Config().Get("description").String()
	if desc != "" && !hideDisabled {
		helpInfo.Usage = append(
			helpInfo.Usage,
			"",
			desc,
		)
	}

	h := help.New(
		helpInfo,
		help.Style{
			Primary:     ansicolor.Style{FG: theme.Primary, Format: ansicolor.Bold},
			Info:        ansicolor.Style{FG: theme.Secondary},
			Version:     ansicolor.Style{FG: theme.Accent, Format: ansicolor.Faint},
			Credits:     ansicolor.Style{FG: theme.Secondary},
			License:     ansicolor.Style{FG: theme.Accent, Format: ansicolor.Faint},
			Description: ansicolor.Style{FG: theme.Primary},
			Category:    ansicolor.Style{FG: theme.Accent, Format: ansicolor.Bold},
		},
	)

	for _, scmd := range rt.cmd.SubCommands() {
		name := scmd.Name
		desc := scmd.Description
		if scmd.Disabled {
			if hideDisabled {
				continue
			}
			name = ansicolor.Format(name, ansicolor.Strike)
			desc += ansicolor.Text(" (disabled)", theme.Warning, ansicolor.Color{}, 0)
		}
		h.AddCommand(scmd.Category, name, desc)
	}

	h.AddCategoryDescriptions(rt.cmd.Categories())

	if !rt.cmd.IsRoot() && !rt.cmd.Disabled() {
		h.AddCommandFlags(rt.cmd.Flags())
		h.AddSharedFlags(rt.cmd.SharedFlags())
	}

	h.AddGlobalFlags(rt.cmd.GlobalFlags())

	if err := h.Print(); err != nil {
		return fmt.Errorf("%w: failed to print help %w", Error, err)
	}
	return nil
}

func (rt *Runtime) executeDoAction() error {
	defer func() {
		if r := recover(); r != nil {
			rt.recover(r, fmt.Sprintf("command failed: %s", rt.cmd.Name()))
		}
	}()
	doTimer := time.Now()
	internal.Log(rt.sess.Log(), "executing command", slog.String("args", strings.Join(os.Args, " ")))
	src, err := rt.cmd.ExecDo(rt.sess)
	if err != nil {
		if errors.Is(err, command.ErrNotImplemented) {
			rt.sess.Log().NotImplemented(err.Error())
		} else {
			var source []slog.Attr
			if src != "" {
				source = append(source, slog.String(slog.SourceKey, src))
			}
			rt.sess.Log().Error(err.Error(), source...)
		}

	}

	internal.Log(rt.sess.Log(), "command took", slog.String("took", time.Since(doTimer).String()))
	return err
}

type ShutDown struct{}

// ExitCh return blocking channel that will reveive a signal when the runtime exits
func (rt *Runtime) ExitCh() <-chan ShutDown {
	if testing.Testing() && rt.exitCh == nil {
		rt.exitCh = make(chan ShutDown, 1)
	}
	return rt.exitCh
}

func (rt *Runtime) SetExecLogLevel(lvl logging.Level) {
	rt.execlvl = lvl
}

func (rt *Runtime) Exit(code int) {
	rt.log(0, internal.LogLevelHappy, "shutting down", slog.Int("exit.code", code))

	for _, fn := range rt.exitFuncs {
		if err := fn(rt.sess, code); err != nil {
			rt.log(0, logging.LevelError, "exit func", slog.String("err", err.Error()))
			code = 1
		}
	}

	if rt.engine != nil {
		if err := rt.engine.Stop(rt.sess); err != nil {
			rt.sess.Log().Error("failed to stop engine", slog.String("err", err.Error()))
		}
	}

	if rt.sess != nil {
		if rt.sess.Get("app.stats.enabled").Bool() && rt.sess.Log().Level() <= logging.LevelDebug {
			if rt.engine != nil {
				rt.sess.Log().Println(rt.engine.Stats().State().String())
			}
		}
		rt.sess.Destroy(nil)
		if err := rt.sess.Err(); err != nil && !errors.Is(err, session.ErrExitSuccess) {
			rt.log(0, logging.LevelError, "session", slog.String("err", err.Error()))
			code = 1
		}
		// Final session cleanup, close channels
		rt.sess.Dispatch(internal.TerminateSessionEvent)
	}

	if rt.exitCh != nil {
		rt.exitCh <- struct{}{}
	}

	if !rt.startedAt.IsZero() {
		rt.log(1, logging.LevelDebug, i18n.PTD(i18np, "shutdown_complete", "shutdown complete"), slog.String("uptime", time.Since(rt.startedAt).String()), slog.Int("exit.code", code))
	} else {
		rt.log(1, logging.LevelDebug, i18n.PTD(i18np, "shutdown_complete", "shutdown complete"), slog.Int("exit.code", code))
	}

	rt.disposeLogger()

	// If we are not testing, exit the main process
	if !testing.Testing() {
		os.Exit(code)
	}
}

func (rt *Runtime) disposeLogger() {
	if rt.sess != nil {
		logger := rt.sess.Log()
		if logger != nil {
			if err := logger.Dispose(); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}
		}
	}
}

func (rt *Runtime) log(depth int, lvl logging.Level, msg string, attrs ...slog.Attr) {
	// try to log with session logger
	if rt.sess != nil {
		rt.sess.Log().LogDepth(depth+1, lvl, msg, attrs...)
		return
	}
	if rt.tmplogger != nil {
		rt.tmplogger.LogDepth(depth+1, lvl, msg, attrs...)
		return
	}

	// log with slog
	slog.LogAttrs(context.Background(), slog.Level(lvl), msg, attrs...)
}
