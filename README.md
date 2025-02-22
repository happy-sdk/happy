![Happy Logo](assets/images/happy.svg)

# Happy Prototyping Framework and SDK

Package happy is a powerful tool for developers looking to bring their ideas to life through rapid prototyping. With its comprehensive set of resources and modular design, it's easy to create working prototypes or MVPs with minimal technical knowledge or infrastructure planning. Plus, its flexible design allows it to seamlessly integrate into projects with components written in different programming languages. So why wait? Let Happy help you achieve your goals and bring a smile to your face along the way.

:warning: *Happy is very early in development phase and is not intended for production use.*  

![GitHub Release](https://img.shields.io/github/v/release/happy-sdk/happy) [![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy)](https://pkg.go.dev/github.com/happy-sdk/happy) [![Coverage Status](https://coveralls.io/repos/github/happy-sdk/happy/badge.svg?branch=main)](https://coveralls.io/github/happy-sdk/happy?branch=main) ![GitHub License](https://img.shields.io/github/license/happy-sdk/happy)

## Creating application

Happy SDK is designed to simplify your development process without introducing any new or additional developer tools. Your applications built with Happy can be used, built, and tested with the standard Go build tools, such as 'go test', 'go build', and 'go run'. With Happy, you have complete control over your development environment, as it will not add any third-party dependencies to your project.

*Here's a simple example of how you can use Happy:*

```go
// main.go
package main

import (
	"fmt"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/app/session"
)

func main() {
  app := happy.New(happy.Settings{})

  app.Do(func(sess *session.Context, args action.Args) error {
    sess.Log().Println("Hello, world!")
    return nil
  })

  app.Run()
}

```

For more examples, take a look at the [examples](#examples) section and the examples in the ./examples/ directory."

### Application api

More details of api read happy Godoc  
 [![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy/sdk/app)](https://pkg.go.dev/github.com/happy-sdk/happy/sdk/app)

```go

...
app.AddInfo(/* add paragraph to app help */)
app.WithOptions(/* adds allowed runtime option */)
app.SetOptions(/* update default value for any application or addon option */) 
app.Setup(/* optional setup action called only first time app is used */)
app.WithAddon(/* adds addon to app */)
app.WithBrand(/* customize branding of the app */)
app.WithCommands(/* adds command(s) to app */)
app.WithFlags(/* adds flag(s) to app root command */)
app.WithLogger(/* uses provided logger */)
app.WithMigrations(/* use migration manager */)
app.WithServices(/* adds service(s) to app */)

...

...
// Application root command actions
app.BeforeAlways(/* called always before any command is invoked*/)
app.Before(/* called before root command is invoked*/)
app.Do(/* root command Do function */)
app.AfterSuccess(/* called when root cmd or sub command returns without errors */)
app.AfterFailure(/* called when root cmd or sub command returns with errors */)
app.AfterAlways(/* called always when root cmd or sub command returns */)
app.Tick(/* called in specific interfal while root command is blocking */)
app.Tock(/* called after every tick*/)
...
```

### Commands

`command.Command` provides a universal API for attaching sub-commands directly to the application or providing them from an Addon.  
 [![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy/sdk/cli/command)](https://pkg.go.dev/github.com/happy-sdk/happy/sdk/cli/command)


```go
import "github.com/happy-sdk/happy/sdk/cli/command"
...
cmd := command.New(command.Config{
  Name: "my-command"
})

cmd.Do(/* Main function for the command */)
// Optional:
cmd.Before(/* Called after app.Before and before cmd.Do */)
cmd.AfterSuccess(/* Called when cmd.Do returns without errors */)
cmd.AfterFailure(/* Called when cmd.Do returns with errors */)
cmd.AfterAlways(/* Called always when cmd.Do returns */)

cmd.Usage(/* add attitional usage lines to help menu */)
cmd.AddInfo(/* add long description paragraph for command */)
cmd.WithSubCommands(/* Add a sub-command to the command */)
cmd.WithFlags(/* add flag(s) to  command*/)
...
```

### Services

The `services.Service` API provides a flexible way to add runtime-controllable background services to your application.  
 [![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy/sdk/services)](https://pkg.go.dev/github.com/happy-sdk/happy/sdk/services)


```go
import (
  "github.com/happy-sdk/happy/sdk/services"
  "github.com/happy-sdk/happy/sdk/services/service"
)
...
svc := services.New(service.Config{
  Name: "my-service",
})

svc.OnRegister(/* Called when the app starts. */)
svc.OnStart(/* Called when the service is requested to start. */)
svc.OnStop(/* Called when the service is requested to stop. */)
svc.OnEvent(/* Called when a specific event is received. */)
svc.OnAnyEvent(/* Called when any event is received. */)
svc.Cron(/* Scheduled cron jobs to run when the service is running. */)
svc.Tick(/* Called every tick when the service is running. */)
svc.Tock(/* Called after every tick when the service is running. */)

app.WithServices(svc)
...
```

## Addons

Addons provide a simple way to bundle commands and services into a single Go package, allowing for easy sharing between projects.  
 [![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy/sdk/addon)](https://pkg.go.dev/github.com/happy-sdk/happy/sdk/addon)

```go
// main.go
package main

import (
  "github.com/happy-sdk/happy"
  "helloworld"
)

func main() {
  app := happy.New(happy.Settings{})
  app.WithAddons(helloworld.Addon())
  app.Main()
}

```

```go
// helloworld/addon.go
package helloworld

import (
  "github.com/happy-sdk/happy/sdk/addon"
  "github.com/happy-sdk/happy/sdk/app/session"
  "github.com/happy-sdk/happy/sdk/custom"
)

type HelloWorldAPI struct {
  custom.API
}

func Addon() *happy.Addon {
  addon := addon.New(addon.Config{
    Name: "Releaser",
  }, 
    addon.Option("my-opt", "default-val", "custom option", false, nil),
  )


  // Optional: Register commands provided by the addon
  addon.ProvideCommands(/* provide command(s) */)

  // Optional: Register services provided by the addon
  addon.ProvideServices(/* provide service(s) */)

  // Optional: Make a custom API accessible across the application 
  addon.ProvideAPI(&HelloWorldAPI{}) 

  // Register all events that the addon may emit ()
  addon.Emits(/* events what addon emits */)

  // Optional callback to be called when the addon is registered
  addon.OnRegister(func(sess session.Register) error {
    sess.Log().Notice("hello-world addon registered")
    return nil
  })

  return addon
}
```

## Credits

[![GitHub contributors](https://img.shields.io/github/contributors/happy-sdk/happy?style=flat-square)](https://github.com/happy-sdk/happy/graphs/contributors)

<sub>**Happy banner design.**</sub>  
<sup>Happy banner was designed by Egon Elbre <a href="https://egonelbre.com/" target="_blank">egonelbre.com</a></sup>
