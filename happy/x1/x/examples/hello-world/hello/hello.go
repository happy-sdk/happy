package hello

import (
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/addon"
	"github.com/mkungla/happy/x/happyx"
	"github.com/mkungla/happy/x/sdk"
	"github.com/mkungla/happy/x/service"
)

type HelloAddon struct {
	happy.Addon
	cmds []happy.Command
	svcs []happy.Service
	api  API
}

func New() happy.Addon {
	av, _ := happy.ParseVersion("v0.1.0")
	a, _ := addon.New("hello", "say hello", av)

	addon := &HelloAddon{
		Addon: a,
	}

	addon.cmds = []happy.Command{cmdHello()}

	addon.svcs = []happy.Service{serviceHello()}
	// for _, opt := range options {
	// 	_ = opt(addon)
	// }

	return addon
}

func (a *HelloAddon) Commands() []happy.Command {
	return a.cmds
}

func (a *HelloAddon) Services() []happy.Service {
	return a.svcs
}

func (a *HelloAddon) API() happy.API {
	return &a.api
}

type API struct{}

func (a *API) Commands() error {
	return nil
}

func (a *API) Services() error {
	return nil
}

func (a *API) GetHello() string {
	return "HELLO"
}

func cmdHello() happy.Command {
	cmd, _ := sdk.NewCommand(
		"hello",
		happyx.ReadOnlyOption("usage.decription", "Say hello"),
		// happyx.ReadOnlyOption("category", "GENERAL"),
	)
	cmd.Before(func(sess happy.Session, f happy.Flags, assets happy.FS, status happy.ApplicationStatus, apis []happy.API) error {
		sess.Log().Notice("hello.before")
		<-sess.Ready()
		return nil
	})

	cmd.Do(func(sess happy.Session, f happy.Flags, assets happy.FS, status happy.ApplicationStatus, apis []happy.API) error {
		sess.Log().Notice("hello.do")
		return nil
	})

	cmd.AfterFailure(func(sess happy.Session, err happy.Error) error {
		sess.Log().Notice("hello.AfterFailure")
		return nil
	})

	cmd.AfterSuccess(func(sess happy.Session) error {
		sess.Log().Notice("hello.AfterSuccess")
		return nil
	})

	cmd.AfterAlways(func(sess happy.Session, status happy.ApplicationStatus) error {
		sess.Log().Noticef("hello.AfterAlways: elapsed %s", status.Elapsed())
		return nil
	})

	return cmd
}

type HelloService struct {
	happy.Service
}

func serviceHello() happy.Service {
	s, _ := service.New(
		"hello-service",
		"Say hello in background",
		"/hello-service",
	)

	svc := HelloService{
		Service: s,
	}

	svc.OnInitialize(func(sess happy.Session, status happy.ApplicationStatus) error {
		sess.Log().Notice("svc.OnInitialize")
		return nil
	})

	svc.OnStart(func(sess happy.Session, args happy.Variables) error {
		sess.Log().Notice("svc.OnStart")
		return nil
	})

	svc.OnStop(func(sess happy.Session) error {
		sess.Log().Notice("svc.OnStop")
		return nil
	})

	svc.OnTick(func(sess happy.Session, ts time.Time, delta time.Duration) error {
		// sess.Log().Notice("svc.OnTick")
		return nil
	})

	svc.OnTock(func(sess happy.Session, ts time.Time, delta time.Duration) error {
		// sess.Log().Notice("svc.OnTock")
		return nil
	})

	svc.OnEvent("say", "hello.world", func(sess happy.Session, ev happy.Event) error {
		sess.Log().Notice("svc.OnEvent hello")
		return nil
	})

	svc.Cron(func(cs happy.CronScheduler) {
		cs.Job("* * * * * *", func(sess happy.Session) error {
			sess.Log().Notice("Cron")
			return nil
		})
	})
	return svc
}
