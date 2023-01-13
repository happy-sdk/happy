package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/pkg/varflag"
	"github.com/mkungla/happy/pkg/vars"
	"golang.org/x/exp/slog"
)

// A simple hello application
//
// Usage:
// - go run ./examples/kitchensink/
//
// Increase verbosity
// - go run ./examples/kitchensink/ --verbose
// - go run ./examples/kitchensink/ --debug
// - go run ./examples/kitchensink/ --system-debug
//
// Hello command
// - go run ./examples/kitchensink/ hello --name me --repeat 10
// - go run ./examples/kitchensink/ hello -n me -r 10
// - go run ./examples/kitchensink/ hello -h
//
// Help
// - go run ./examples/kitchensink/ -h
func main() {
	app := happy.New(
		happy.Option("app.name", "Happy Kitchensink"),
		happy.Option("app.description", "Application that demonstrates the happy SDK features."),
		happy.Option("app.copyright.by", "Happy Authors"),
		happy.Option("app.copyright.since", 2022),
		happy.Option("app.license", "Apache-2.0 license"),
		happy.Option("app.throttle.ticks", time.Second/240),
		happy.Option("app.host.addr", "happy://localhost/kitchensink"),
		happy.Option("app.version", "v0.1.0"),
		happy.Option("app.settings.persistent", false),
		happy.Option("log.level", happy.LevelWarn),
		happy.Option("log.source", true),
		happy.Option("log.colors", true),
		happy.Option("log.stdlog", true),
		happy.Option("log.secrets", "password,apiKey"),
	)

	// Add commands,
	app.AddCommand(helloCommand())

	// register simple background service
	app.RegisterService(simpleBackgroundService())

	app.WithAddons(
		// example addon
		rendererAddon(),
	)

	app.Before(func(sess *happy.Session, args happy.Args) error {
		sess.Log().Info("preparing kitchensink")
		// dummy timout to stop app after 30 seconds
		go func() {
			timout, cancel := context.WithTimeout(sess, time.Second*30)
			defer func() { sess.Log().Info("clearing timout"); cancel() }()
			<-timout.Done()
			sess.Destroy(timout.Err())
		}()

		// load services we want to use in app
		loader := happy.NewServiceLoader(
			sess,
			"background",
			// as many service you need at once
		)
		<-loader.Load()
		return loader.Err()
	})

	app.Do(func(sess *happy.Session, args happy.Args) error {
		sess.Log().Out("Ctrl+C to exit or wait 30 seconds")
		time.Sleep(time.Second * 2) // dummy delay to start our renderer
		// you can start and stop services with loader as "background" inBefore is used
		// or start it from any place by dispatching event.

		sess.Dispatch(happy.StartServicesEvent(
			"happy://localhost/kitchensink/service/renderer",
		))
		fmt.Println("")
		// This sleep will ALWAYS block 10 seconds
		// even when you Ctrl-C. That case graceful shutdown will start
		// after reaching <-sess.Done() and falling trough since session is already destroyed.
		task := sess.Log().Task("sleep 10 seconds before stopping render service")
		time.Sleep(time.Second * 10)
		sess.Dispatch(happy.StopServicesEvent(
			"happy://localhost/kitchensink/service/renderer",
		))
		fmt.Println("")
		sess.Log().Ok("asked render service to stop", task.LogAttr())
		<-sess.Done()

		api, err := happy.GetAPI[*RendererAPI](sess, "renderer")
		if err != nil {
			return err
		}
		sess.Log().Ok("rendered total", slog.Int("frames", api.TotalFrames()))
		return nil
	})

	app.AfterSuccess(func(sess *happy.Session) error {
		sess.Log().Ok("execution completed successfully")
		return nil
	})

	app.AfterFailure(func(sess *happy.Session, err error) error {
		sess.Log().Error("execution failed with error", err)
		return nil
	})

	app.AfterAlways(func(sess *happy.Session) error {
		sess.Log().Notice("execution completed do something always")
		return nil
	})

	app.Main()
}

func helloCommand() *happy.Command {
	cmd := happy.NewCommand(
		"hello",
	)

	// cmd.AddSubCommand(someSubCommand)

	nameflag, _ := varflag.New("name", "anonymous", "print hello <name>", "n")
	repeatflag, _ := varflag.Int("repeat", 1, "repeat message n times", "r")
	cmd.AddFlag(nameflag)
	cmd.AddFlag(repeatflag)

	// app.Before(func(sess *happy.Session, args happy.Args) error
	cmd.Do(func(sess *happy.Session, args happy.Args) error {
		r := args.Flag("repeat").Var().Int()
		for i := 0; i < r; i++ {
			sess.Log().Out("hello", slog.String("name", args.Flag("name").String()))
		}
		return nil
	})
	// app.AfterSuccess(func(sess *happy.Session) error
	// app.AfterFailure(func(sess *happy.Session, err error) error
	// app.AfterAlways(func(sess *happy.Session) error
	return cmd
}

func simpleBackgroundService() *happy.Service {
	svc := happy.NewService(
		"background",
	)
	svc.OnInitialize(func(sess *happy.Session) error {
		sess.Log().Ok("initialize background service")
		return nil
	})

	svc.OnStart(func(sess *happy.Session) error {
		sess.Log().Ok("started background service")
		return nil
	})

	svc.OnStop(func(sess *happy.Session) error {
		sess.Log().Ok("stopped background service")
		return nil
	})

	// See renderer example for tick tock example
	// svc.OnTick(func(sess *happy.Session, ts time.Time, delta time.Duration) error {})
	// svc.OnTock(func(sess *happy.Session, delta time.Duration, tps int) error {})

	svc.OnEvent("kitchen", "message", func(sess *happy.Session, ev happy.Event) error {
		sess.Log().Out("kitchen:", ev.Payload().Get("message"))
		return nil
	})

	svc.OnAnyEvent(func(sess *happy.Session, ev happy.Event) error {
		var attrs []any
		attrs = append(attrs, slog.Group(
			"event",
			slog.String("scope", ev.Scope()),
			slog.String("key", ev.Key()),
		))
		if ev.Payload() != nil {
			for _, entry := range ev.Payload().All() {
				attrs = append(attrs, entry)
			}
		}
		sess.Log().Out("recived event: ", attrs...)
		return nil
	})

	svc.Cron(func(schedule happy.CronScheduler) {
		schedule.Job("@every 5s", func(sess *happy.Session) error {
			sess.Log().Notice("cron: running every 5 seconds", slog.Time("now", time.Now()))
			return nil
		})
		schedule.Job("@every 15s", func(sess *happy.Session) error {
			sess.Log().Notice("cron: running every 15 seconds", slog.Time("now", time.Now()))
			return nil
		})
	})
	return svc
}

// RendererAddon is example addon providing a service
type RendererAddon struct {
	info happy.AddonInfo
	last *Frame
	api  *RendererAPI
}

func rendererAddon() *RendererAddon {
	addon := &RendererAddon{
		api: &RendererAPI{},
	}
	return addon
}

func (addon *RendererAddon) Register(sess *happy.Session) (happy.AddonInfo, error) {
	addon.info.Name = "renderer"
	addon.info.Description = "renderer addon"
	addon.info.Version = sess.Get("app.version").String()
	return addon.info, nil
}

// Required for happy.Addon interface
func (addon *RendererAddon) Commands() []*happy.Command { return nil }
func (addon *RendererAddon) Services() []*happy.Service {
	return []*happy.Service{
		addon.renderer(),
	}
}
func (addon *RendererAddon) API() happy.API        { return addon.api }
func (addon *RendererAddon) Events() []happy.Event { return nil }

type RendererAPI struct {
	total int
}

func (*RendererAPI) Get(key string) vars.Variable { return vars.EmptyVariable }

func (api *RendererAPI) TotalFrames() int {
	return api.total
}

// Custom addon implementation
type Frame struct {
	FPS          int
	FrameDelta   time.Duration
	ProcessDelta time.Duration
	Message      string
	Timestamp    time.Time
}

func (addon *RendererAddon) renderer() *happy.Service {
	svc := happy.NewService(
		"renderer",
	)
	// create next frame
	svc.OnTick(func(sess *happy.Session, ts time.Time, delta time.Duration) error {
		// create dummy frame
		return addon.process("happy kitchen", ts, delta)
	})
	// render and postprocess the frame
	svc.OnTock(func(sess *happy.Session, delta time.Duration, tps int) error {
		addon.last.FPS = tps
		addon.last.ProcessDelta = delta
		return addon.render()
	})
	svc.OnStop(func(sess *happy.Session) error {
		fmt.Print("\n")
		sess.Log().Notice("renderer service stopped")
		return nil
	})
	return svc
}

func (addon *RendererAddon) process(msg string, ts time.Time, delta time.Duration) error {
	frame := &Frame{
		Message:    msg,
		FrameDelta: delta,
		Timestamp:  ts,
	}
	addon.last = frame
	return nil
}

func (addon *RendererAddon) render() error {
	frame := *addon.last
	if addon.api.total == 0 {
		fmt.Print("\n")
	}
	fmt.Printf(
		"\rframe: FPS [%-4d] - frame-delta [%-15s] - process-delta [%-15s]",
		frame.FPS, frame.FrameDelta, frame.ProcessDelta)
	addon.api.total++
	return nil
}
