package main

import (
	"context"
	"fmt"
	"time"

	"log/slog"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/varflag"
	"github.com/happy-sdk/vars"
)

func main() {
	app := happy.New(
		happy.Settings{
			Name:           "Happy Kitchensink",
			Description:    "Application that demonstrates the happy SDK features.",
			Slug:           "kitchensink",
			License:        "Apache-2.0 license",
			CopyrightBy:    "Happy Authors",
			CopyrightSince: 2022,
			Logger: logging.Settings{
				Level:   logging.LevelWarn,
				Source:  false,
				Secrets: "password,apiKey",
			},
			ThrottleTicks: settings.Duration(time.Second / 60),
		},
	)

	app.Help(`Example:
  - go run ./examples/kitchensink/

  Increase verbosity
  - go run ./examples/kitchensink/ --verbose
  - go run ./examples/kitchensink/ --debug
  - go run ./examples/kitchensink/ --system-debug

  Hello command
  - go run ./examples/kitchensink/ hello --name me --repeat 10
  - go run ./examples/kitchensink/ hello -n me -r 10
  - go run ./examples/kitchensink/ hello -h

  Help
  - go run ./examples/kitchensink/ -h`)
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
		sess.Log().Msg("Ctrl+C to exit or wait 30 seconds")
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
		// task := sess.Log().Task("sleep", "sleep 10 seconds before stopping render service")
		time.Sleep(time.Second * 10)
		sess.Dispatch(happy.StopServicesEvent(
			"happy://localhost/kitchensink/service/renderer",
		))
		fmt.Println("")
		// sess.Log().Ok("asked render service to stop", task)
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
			sess.Log().Msg("hello", slog.String("name", args.Flag("name").String()))
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
		sess.Log().Msg("kitchen:", ev.Payload().Get("message"))
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
		sess.Log().Msg("recived event: ", attrs...)
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

func rendererAddon() *happy.Addon {
	addon := happy.NewAddon(
		"renderer",
		nil,
		happy.Option("usage", "Sample renderer addon"),
		happy.Option("description", `
      This addon is just show possibilities of Addon system.
    `),
	)

	api := &RendererAPI{}

	addon.API = api
	addon.OnRegister(func(sess *happy.Session, opts *happy.Options) error {
		sess.Log().Notice("renderer addon registered")
		return nil
	})
	addon.ProvidesService(api.renderer())

	return addon
}

// RendererAddon is example addon providing a service
type RendererAPI struct {
	info  happy.AddonInfo
	last  *Frame
	total int
}

func (*RendererAPI) Get(key string) vars.Variable  { return vars.EmptyVariable }
func (*RendererAPI) Set(key string, val any) error { return nil }

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

func (api *RendererAPI) renderer() *happy.Service {
	svc := happy.NewService(
		"renderer",
	)
	// create next frame
	svc.OnTick(func(sess *happy.Session, ts time.Time, delta time.Duration) error {
		// create dummy frame
		return api.process("happy kitchen", ts, delta)
	})
	// render and postprocess the frame
	svc.OnTock(func(sess *happy.Session, delta time.Duration, tps int) error {
		api.last.FPS = tps
		api.last.ProcessDelta = delta
		return api.render()
	})
	svc.OnStop(func(sess *happy.Session) error {
		fmt.Print("\n")
		sess.Log().Notice("renderer service stopped")
		return nil
	})
	return svc
}

func (api *RendererAPI) process(msg string, ts time.Time, delta time.Duration) error {
	frame := &Frame{
		Message:    msg,
		FrameDelta: delta,
		Timestamp:  ts,
	}
	api.last = frame
	return nil
}

func (api *RendererAPI) render() error {
	frame := *api.last
	if api.total == 0 {
		fmt.Print("\n")
	}
	fmt.Printf(
		"\rframe: FPS [%-4d] - frame-delta [%-15s] - process-delta [%-15s]",
		frame.FPS, frame.FrameDelta, frame.ProcessDelta)
	api.total++
	return nil
}
