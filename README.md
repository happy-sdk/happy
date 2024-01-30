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
  "github.com/happy-sdk/happy/sdk/logging"
)

func ExampleNew() {
  app := happy.New(happy.Settings{})
  app.Do(func(sess *happy.Session, args happy.Args) error {
    sess.Log().Println("Hello, world!")
    return nil
  })
  app.Run()
}

```

For more examples, take a look at the [examples](#examples) section and the examples in the ./examples/ directory."

### Application api

More details of api read happy Godoc 

```go
...
app.WithAddon(/* adds addon to app */)
app.WithMigrations(/* use migration manager */)
app.WithService(/* adds service to app */)
app.WithCommand(/* adds command to app */)
app.WithFlag(/* adds flag to app root command*/)
app.WithLogger(/* uses provided logger */)
app.WithOptions(/* adds allowed runtime option */)
...

...
// All the following are optional 
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

`happy.Command` provides a universal API for attaching sub-commands directly to the application or providing them from an Addon.


```go
...
cmd := happy.NewCommand(
  "my-command",
  happy.Option("usage", "My sub-command"),
  happy.Option("argn.min", 1),
  happy.Option("argn.max", 10),
)

cmd.Do(/* Main function for the command */)

// Optional:
cmd.AddInfo(/* add long description paragraph for command*/)
cmd.AddFlag(/* add flag to  command*/)

cmd.Before(/* Called after app.Before and before cmd.Do */)
cmd.AfterSuccess(/* Called when cmd.Do returns without errors */)
cmd.AfterFailure(/* Called when cmd.Do returns with errors */)
cmd.AfterAlways(/* Called always when cmd.Do returns */)
cmd.AddSubCommand(/* Add a sub-command to the command */)
cmd.AddFlag(/* Add a flag for the command */)

cmd.AddSubCommand(/* add subcommand to command */)
...
```

### Services

The `happy.Service` API provides a flexible way to add runtime-controllable background services to your application.

```go
...
svc := happy.NewService("my-service")

svc.OnInitialize(/* Called when the app starts. */)
svc.OnStart(/* Called when the service is requested to start. */)
svc.OnStop(/* Called when the service is requested to stop. */)
svc.OnEvent(/* Called when a specific event is received. */)
svc.OnAnyEvent(/* Called when any event is received. */)
svc.Cron(/* Scheduled cron jobs to run when the service is running. */)
svc.Tick(/* Called every tick when the service is running. */)
svc.Tock(/* Called after every tick when the service is running. */)

app.RegisterService(svc)
...
```

## Addons

Addons provide a simple way to bundle commands and services into a single Go package, allowing for easy sharing between projects.

```go
// main.go
package main

import (
  "github.com/happy-sdk/happy"
  "helloworld"
)

func main() {
  app := happy.New()
  app.WithAddons(helloworld.Addon())
  app.Main()
}

```

```go
// helloworld/addon.go
package helloworld

import "github.com/happy-sdk/happy"

type HelloWorldAPI struct {
  happy.API
}

func Addon() *happy.Addon {
  addon := happy.NewAddon(
    "hello-world",
    happy.Option("description", "example addon"),
  )

  // Optional: Set a custom setting
  addon.Setting("greet.msg", "any value", "setting description", /* validation func */)

  // Optional: Register commands provided by the addon
  addon.ProvidesCommand(/* provide command */)

  // Optional: Register services provided by the addon
  addon.ProvidesService(/* provide service */)

  // Optional: Make a custom API accessible across the application 
  addon.ProvidesAPI(&HelloWorldAPI{}) 

  // Register all events that the addon may emit ()
  addon.Emits("event scope", "event key" , "event description", /* example payload */)
  addon.EmitsEvent(/* if you already have event */)

  addon.Option("key", "value", "addon specific runtime option", /* optional validator*/)
  
  // Optional callback to be called when the addon is registered
  addon.OnRegister(func(sess *happy.Session, opts *happy.Options) error {
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
