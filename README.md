![Happy Logo](assets/images/happy.svg)

# Happy Prototyping Framework and SDK

Happy SDK is an open-source Go framework designed to make building applications simple and fast. Its clear, modular structure and addon system promote code reuse, enabling anyone—coders or non-coders—to create prototypes or full projects efficiently. It integrates smoothly with projects using multiple programming languages, helping teams bring ideas to life with ease with built-in monorepo support.

:warning: *Until v1.0.0, API changes may break compatibility, so pin your Happy version and update cautiously.*  

![GitHub Release](https://img.shields.io/github/v/release/happy-sdk/happy) [![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy)](https://pkg.go.dev/github.com/happy-sdk/happy) [![Coverage Status](https://coveralls.io/repos/github/happy-sdk/happy/badge.svg?branch=main)](https://coveralls.io/github/happy-sdk/happy?branch=main) ![GitHub License](https://img.shields.io/github/license/happy-sdk/happy)

## Creating application

Happy SDK is designed to simplify your development process without introducing any new or additional developer tools. Your applications built with Happy can be used, built, and tested with the standard Go build tools, such as 'go test', 'go build', and 'go run'. With Happy, you have complete control over your development environment, as it will not add any third-party dependencies to your project.

*Here's a minimal example of how you can use Happy:*

```go
// main.go
package main

import (
 "fmt"

 "github.com/happy-sdk/happy"
 "github.com/happy-sdk/happy/sdk/action"
 "github.com/happy-sdk/happy/sdk/session"
)

func main() {
 app := happy.New(nil)

 app.Do(func(sess *session.Context, args action.Args) error {
  sess.Log().Info("Hello, world! ")
  return nil
 })

 app.Run()
}
// go run . 
// OUT: 
// info  00:00:00.000 Hello, world!
```

*Here's a example enabling builtin global flags including (help,version:*

```go
// main.go
package main

import (
 "fmt"

 "github.com/happy-sdk/happy"
 "github.com/happy-sdk/happy/sdk/action"
 "github.com/happy-sdk/happy/sdk/session"
)

func main() {
 app := happy.New(&happy.Settings{
  // Engine: happy.EngineSettings{},

  CLI: happy.CliSettings{
   WithGlobalFlags: true,
  },

  // Profiles: happy.ProfileSettings{},
  // DateTime: happy.DateTimeSettings{},
  Instance: happy.InstanceSettings{},

  // We enable global flags which has --verbose flag to set info level
  // so we set default something higher than info
  Logging: happy.LoggingSettings{
   Level: logging.LevelSuccess,
  },

  // Services: happy.ServicesSettings{},
  // Stats:    happy.StatsSettings{},
  // Devel:    happy.DevelSettings{},
  // I18n:     happy.I18nSettings{},
 })

 app.Do(func(sess *session.Context, args action.Args) error {
  sess.Log().Info("Hello, world! ")
  return nil
 })

 app.Run()
}
// go run . -h
// OUT: 
//  Happy Prototype - v0.0.1-devel+git.<hash>.<timestamp>
//  Copyright © {year} Anonymous
//  License: NOASSERTION
//  
//  This application is built using the Happy-SDK to provide enhanced functionality and features.
//
//  yourcmd [flags]
//
// GLOBAL FLAGS:
//
//  --debug              enable debug log level - default: "false"
//  --help         -h    display help or help for the command. [...command --help] - default: "false"
//  --show-exec    -x    the -x flag prints all the cli commands as they are executed. - default: "false"
//  --system-debug       enable system debug log level (very verbose) - default: "false"
//  --verbose      -v    set log level info - default: "false"
//  --version            print application version - default: "false"
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
  "github.com/happy-sdk/happy/sdk/session"
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
