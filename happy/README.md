![Happy Logo](assets/images/happy.svg)

# Howijd Prototyping Framework and SDK

Package happy is a powerful tool for developers looking to bring their ideas to life through rapid prototyping. With its comprehensive set of resources and modular design, it's easy to create working prototypes or MVPs with minimal technical knowledge or infrastructure planning. Plus, its flexible design allows it to seamlessly integrate into projects with components written in different programming languages. So why wait? Let Happy help you achieve your goals and bring a smile to your face along the way.

:warning: *Happy is very early in development phase and is not intended for production use.*  

[![PkgGoDev](https://pkg.go.dev/badge/github.com/mkungla/happy)](https://pkg.go.dev/github.com/mkungla/happy)

## Creating application

Happy SDK is designed to simplify your development journey without introducing any new or additional developer tools. Your applications built with Happy can be used, built, and tested with Go standard build tools such as 'go test', 'go build', and 'go run'. Furthermore, Happy will not add any third-party dependencies to your project, giving you complete control over your development environment.

```go
// main.go
package main

import (
	"github.com/mkungla/happy"
)

func main() {
	app := happy.New()
	app.Do(func(sess *happy.Session, args happy.Args) error {
		sess.Log().Println("hello world")
		return errors.New("boring it just says hello world")
	})

  // must be called at the end, depending on archidecture it will
  // block or release main thread.
	app.Main()
}
```

Take a look at [examples](#examples) section and examples in `./examples/` directory. 

### Application api

```go
// All the following are optional 
app.Before(/* called always before any command is invoked*/)
app.Do(/* root command Do function */)
app.AfterSuccess(/* called when root cmd or sub command returns without errors */)
app.AfterFailure(/* called when root cmd or sub command returns with errors */)
app.AfterAlways(/* called always when root cmd or sub command returns */)
app.OnTick(/* called while root command is blocking */)
app.OnTock(/* called while root command is blocking after tick*/)
app.OnInstall(/* optional installation step to call when user first uses your app */)
app.OnMigrate(/* optional migrations step to call when user upgrades/downgrades app */)
app.Cron(/* optional cron jobs registerd fo application */)
app.RegisterService(/* register stand alone service to your app. */)
app.AddCommand(/* add sub command to your app. */)
app.AddFlag(/* add global flag to your app. */)
app.Setting(/* add additional; custom user settings to your app */)

```

### Commands

```go
cmd := happy.NewCommand(
	"my-command",
	happy.Option("usage", "my sub command"),
)

cmd.Do(/* command main function */)
// optional:
cmd.Before(/* called after app.Before and before cmd.Do */)
cmd.AfterSuccess(/* called when cmd.Do return without errors */)
cmd.AfterFailure(/* called when cmd.Do return with errors */)
cmd.AfterAlways(/* called always when cmd.Do return */)
cmd.AddSubCommand(/* add sub command to command */)

cmd.AddFlag(/* add flag for command*/)

app.AddCommand(cmd)
```

### Services

```go
svc := happy.NewService(
	"my-service",
	happy.Option("usage", "my custom service"),
)

cmd.OnInitialize(/* called always when app starts */)
cmd.OnStart(/* called always when service requested to start */)
cmd.OnStop(/* called always when service requested to stop */)
cmd.OnEvent(/* action to call when specific event is recieved */)
cmd.OnAnyEvent(/* action to call when any event is recieved */)
cmd.Cron(/* cron jobs to run based on schedule when service is running */)
cmd.OnTick(/* called on every tick when service is running */)
cmd.OnTock(/* called after every tick when service is running */)

app.RegisterService(svc)
```

## Creating Addon

```go
// main.go
package main

import "github.com/mkungla/happy"

func main() {
	app := happy.New()
	app.WithAddons(helloWorldAddon())
	app.Main()
}
```

```go
// hello-world-addon.go
package main

import "github.com/mkungla/happy"

func helloWorldAddon() *happy.Addon {
  addon := happy.NewAddon(
    "hello-world",
    happy.Option("description", "example addon"),
  )
  // optional:
  addon.Setting("greet.msg", "any value", "setting description", /* validation func */)

  // optional: Register commands what addon provides
  addon.ProvidesCommand(...)

  // optional: Register services what addon provides
  addon.ProvidesService(...)
  
  // optional: Make optional custom API accessible across the application 
  addon.API = &HelloWorldAPI{}

  // Register all events what addon may emit
  addon.Emits("event scope", "event key" , "event description", /* exaple payload */)
  
  // Optional callbac to be called when addon is registered
  addon.OnRegister(func(sess *happy.Session, opts *happy.Options) error {
		sess.Log().Notice("hello-world addon registered")
		return nil
	})
  return addon
}

```

## examples

**hello**

*most minimal usage*

```
go run ./examples/hello/
go run ./examples/hello/ nickname
# increase verbosity
go run ./examples/hello/ --debug
go run ./examples/hello/ --system-debug
# help
go run ./examples/hello/ -h
```

**kitchensink**

*main application when no subcommand is provided*

```
go run ./examples/kitchensink/
# increase verbosity
go run ./examples/kitchensink/ --verbose
go run ./examples/kitchensink/ --debug
go run ./examples/kitchensink/ --system-debug

# main application help
go run ./examples/kitchensink/ -h
```

*hello command with flags*

```
go run ./examples/kitchensink/ hello --name me --repeat 10 
# or shorter
go run ./examples/kitchensink/ hello -n me -r 10 

# help for hello command
go run ./examples/kitchensink/ hello -h
```

## Credits

[![GitHub contributors](https://img.shields.io/github/contributors/mkungla/happy?style=flat-square)](https://github.com/mkungla/happy/graphs/contributors)

<sub>**Happy banner design.**</sub>  
<sup>Happy banner was designed by Egon Elbre <a href="https://egonelbre.com/" target="_blank">egonelbre.com</a></sup>
