// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fileserver

import (
	"sync"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/cli"
	"github.com/mkungla/happy/x/happyx"
	"github.com/mkungla/happy/x/sdk"
	"github.com/mkungla/happy/x/service"
	// "io/fs"
	// "net/http"
	// "strings"
)

var ErrFileserver = happyx.NewError("fileserver error")

type FileServerAddon struct {
	happy.Addon
	fs       happy.FS
	cmd      happy.Command
	services []happy.Service

	mu sync.Mutex
	// all below must be protected with mutex
	serviceURL happy.URL
}

func New(option ...happy.OptionSetFunc) (*FileServerAddon, happy.Error) {
	a, err := sdk.NewAddon(
		"fileserver",
		"File Server",
		"0.1.0",
		// options
		happyx.Option("directory", "/"),
		happyx.Option("addr", "localhost:8000"),
		happyx.Option("url.prefix", "/"),
		happyx.Option("service.scope", "/fileserver/services"),

		// wrapper command
		happyx.Option("cmd.name", "fileserver"),
		happyx.Option("cmd.usage.decription", "fileserver addon"),
		happyx.Option("cmd.category", "servers"),
		happyx.Option("cmd.description", "Command wraps all FileServer Addon commands."),
	)
	if err != nil {
		return nil, err
	}

	addon := &FileServerAddon{
		Addon: a,
	}

	for _, opt := range option {
		if err := opt(addon); err != nil {
			return nil, sdk.ErrAddon.Wrap(err)
		}
	}

	if err := addon.createCmds(); err != nil {
		return nil, err
	}

	if err := addon.createServices(); err != nil {
		return nil, err
	}

	return addon, nil
}

func (a *FileServerAddon) Commands() []happy.Command {
	return []happy.Command{
		a.cmd,
	}
}

func (a *FileServerAddon) Services() []happy.Service {
	return a.services
}

func (a *FileServerAddon) FS(fs happy.FS) {
	a.fs = fs
}

func (a *FileServerAddon) createCmds() happy.Error {
	name, err := a.GetOption("cmd.name")
	if err != nil {
		return err
	}

	cmd, err := cli.NewCommand(
		name.String(),
		a.GetOptionSetFunc("cmd.category", "category"),
		a.GetOptionSetFunc("cmd.usage.decription", "usage.decription"),
		a.GetOptionSetFunc("cmd.description", "description"),
	)
	if err != nil {
		return err
	}

	serveCmd, err := a.createServeCommand()
	if err != nil {
		return err
	}
	cmd.AddSubCommand(serveCmd)

	// serveCmd.Do(func(sess happy.Session, args []happy.Value, assets happy.FS) error {

	// host, _ := a.opts.LoadOrDefault("host", "localhost")
	// port, _ := a.opts.LoadOrDefault("port", 8080)
	// directory, _ := a.opts.LoadOrDefault("directory", "/")
	// p, _ := a.opts.LoadOrDefault("path", "/")

	// dir := strings.TrimPrefix(directory.String(), "/")

	// if a.fs == nil {
	// 	return ErrFileserver.WithText("filesstem no configured")
	// }
	// sess.Log().Infof("serving FS directory: %s", dir)
	// sess.Log().Infof("starting fileserver http://%s:%s%s", host, port, p)

	// sub, err := fs.Sub(a.fs, dir)
	// if err != nil {
	// 	return ErrFileserver.WithTextf("could not read fileserver root at %s (%s)", dir, err)
	// }

	// mux := http.NewServeMux()

	// // file serving handler
	// filehadler := http.StripPrefix(p.String(), http.FileServer(http.FS(sub)))
	// msg405 := []byte("405 " + http.StatusText(http.StatusMethodNotAllowed))

	// mux.Handle(p.String(), GzipHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	if r.Method != "" && r.Method != "HEAD" && r.Method != "GET" {
	// 		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// 		w.WriteHeader(http.StatusMethodNotAllowed)
	// 		w.Write(msg405)
	// 		return
	// 	}

	// 	// w.Header().Set("Cache-Control", "public, max-age=180")
	// 	filehadler.ServeHTTP(w, r)

	// 	for k, h := range w.Header() {
	// 		sess.Log().Infof("header: %s - %s %s = %s", r.URL.Path, r.Method, k, h)
	// 	}
	// })))

	// go func() {
	// 	if err := http.ListenAndServe(host.String()+":"+port.String(), mux); err != nil {
	// 		sess.Destroy(ErrFileserver.Wrap(err))
	// 	}
	// }()
	// <-sess.Done()
	// 	return nil
	// })

	// cmd.AddSubCommand(serveCmd)

	a.cmd = cmd
	return nil
}

func (a *FileServerAddon) createServeCommand() (happy.Command, happy.Error) {
	serveCmd, err := cli.NewCommand("serve")
	if err != nil {
		return nil, err
	}

	serveCmd.Before(func(sess happy.Session, flags happy.Flags, assets happy.FS, status happy.ApplicationStatus, apis []happy.API) error {
		// set dynamic service url
		a.mu.Lock()
		serviceURL := a.serviceURL
		a.mu.Unlock()

		// wait for fileserver service
		loader := sess.RequireServices(status, serviceURL.String())
		<-loader.Loaded()

		if loader.Err() != nil {
			sess.Log().Error(loader.Err())
			return ErrFileserver
		}
		return nil
	})

	serveCmd.Do(func(sess happy.Session, flags happy.Flags, assets happy.FS, status happy.ApplicationStatus, apis []happy.API) error {
		return nil
	})

	return serveCmd, nil
}

func (a *FileServerAddon) createServices() happy.Error {
	svcScope, err := a.GetOption("service.scope")
	if err != nil {
		return sdk.ErrAddon.Wrap(err)
	}

	svc, err := service.New(
		"fileserver",
		"File Server Service",
		svcScope.String()+"/fileserver",
		// options
		a.GetOptionSetFunc("directory", "directory"),
		a.GetOptionSetFunc("addr", "addr"),
		a.GetOptionSetFunc("url.prefix", "url.prefix"),
	)
	if err != nil {
		return sdk.ErrAddon.Wrap(err)
	}

	svc.OnInitialize(func(sess happy.Session) error {
		// Set registered service URL so that
		// using serve cmd we know what service to call.
		a.mu.Lock()
		a.serviceURL = svc.URL()
		a.mu.Unlock()
		sess.Log().Debug("fileserver.OnInitialize")
		return nil
	})

	svc.OnStart(func(sess happy.Session, args happy.Variables) error {
		sess.Log().Debug("fileserver.OnStart")
		return nil
	})

	svc.OnStop(func(sess happy.Session) error {
		sess.Log().Debug("fileserver.OnStop")
		return nil
	})

	svc.OnTick(func(sess happy.Session, ts time.Time, delta time.Duration) error {
		sess.Log().Experimental("fileserver.OnTick")
		return nil
	})

	svc.OnTock(func(sess happy.Session, ts time.Time, delta time.Duration) error {
		sess.Log().Experimental("fileserver.OnTock")
		return nil
	})

	a.services = append(a.services, svc)
	return nil
}
