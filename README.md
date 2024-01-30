![Happy Logo](assets/images/happy.svg)

# Happy Prototyping Framework and SDK

Package happy is a powerful tool for developers looking to bring their ideas to life through rapid prototyping. With its comprehensive set of resources and modular design, it's easy to create working prototypes or MVPs with minimal technical knowledge or infrastructure planning. Plus, its flexible design allows it to seamlessly integrate into projects with components written in different programming languages. So why wait? Let Happy help you achieve your goals and bring a smile to your face along the way.

:warning: *Happy is very early in development phase and is not intended for production use.*  

[![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy)](https://pkg.go.dev/github.com/happy-sdk/happy) [![Coverage Status](https://coveralls.io/repos/github/happy-sdk/happy/badge.svg?branch=main)](https://coveralls.io/github/happy-sdk/happy?branch=main)

## Creating application

Happy SDK is designed to simplify your development process without introducing any new or additional developer tools. Your applications built with Happy can be used, built, and tested with the standard Go build tools, such as 'go test', 'go build', and 'go run'. With Happy, you have complete control over your development environment, as it will not add any third-party dependencies to your project.

*Here's a simple example of how you can use Happy:*

```go
// main.go
package main

import (
  "errors"
  "github.com/happy-sdk/happy"
)

func main() {
  app := happy.New()
  app.Do(func(sess *happy.Session, args happy.Args) error {
    sess.Log().Println("Hello, world!")
    return errors.New("This is just a basic example.")
  })
  app.Main()
}
```

For more examples, take a look at the [examples](#examples) section and the examples in the ./examples/ directory."

### Application api

More details of api read happy Godoc 

```go
...
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
app.Cron(/* optional cron jobs registered for application */)
app.RegisterService(/* register standalone service to your app. */)
app.AddCommand(/* add sub command to your app. */)
app.AddFlag(/* add global flag to your app. */)
app.Setting(/* add additional, custom user settings to your app */)
...
```

### Commands

`happy.Command` provides a universal API for attaching sub-commands directly to the application or providing them from an Addon.


```go
...
cmd := happy.NewCommand(
  "my-command",
  happy.Option("usage", "My sub-command"),
)

cmd.Do(/* Main function for the command */)

// Optional:
cmd.Before(/* Called after app.Before and before cmd.Do */)
cmd.AfterSuccess(/* Called when cmd.Do returns without errors */)
cmd.AfterFailure(/* Called when cmd.Do returns with errors */)
cmd.AfterAlways(/* Called always when cmd.Do returns */)
cmd.AddSubCommand(/* Add a sub-command to the command */)
cmd.AddFlag(/* Add a flag for the command */)

app.AddCommand(cmd)
...
```

### Services

The `happy.Service` API provides a flexible way to add runtime-controllable background services to your application.

```go
...
svc := happy.NewService(
  "my-service",
  happy.Option("usage", "my custom service"),
)

svc.OnInitialize(/* Called when the app starts. */)
svc.OnStart(/* Called when the service is requested to start. */)
svc.OnStop(/* Called when the service is requested to stop. */)
svc.OnEvent(/* Called when a specific event is received. */)
svc.OnAnyEvent(/* Called when any event is received. */)
svc.Cron(/* Scheduled cron jobs to run when the service is running. */)
svc.OnTick(/* Called every tick when the service is running. */)
svc.OnTock(/* Called after every tick when the service is running. */)

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

func Addon() *happy.Addon {
  addon := happy.NewAddon(
    "hello-world",
    happy.Option("description", "example addon"),
  )

  // Optional: Set a custom setting
  addon.Setting("greet.msg", "any value", "setting description", /* validation func */)

  // Optional: Register commands provided by the addon
  addon.ProvidesCommand(...)

  // Optional: Register services provided by the addon
  addon.ProvidesService(...)

  // Optional: Make a custom API accessible across the application 
  addon.API = &HelloWorldAPI{}

  // Register all events that the addon may emit
  addon.Emits("event scope", "event key" , "event description", /* example payload */)

  // Optional callback to be called when the addon is registered
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

[![GitHub contributors](https://img.shields.io/github/contributors/mkungla/happy?style=flat-square)](https://github.com/happy-sdk/happy/graphs/contributors)

<sub>**Happy banner design.**</sub>  
<sup>Happy banner was designed by Egon Elbre <a href="https://egonelbre.com/" target="_blank">egonelbre.com</a></sup>
