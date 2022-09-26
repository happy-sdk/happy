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

package monitor

import (
	"fmt"
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/pkg/vars"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

type Status struct {
	mu      sync.Mutex
	started time.Time
	stopped time.Time
	evs     int
	addons  []happy.AddonInfo
	deps    []happy.DependencyInfo

	debug happy.Variables

	svcs map[string]*happy.ServiceStatus
}

func (s *Status) start() error {
	s.debug = vars.AsMap[happy.Variables, happy.Variable, happy.Value](new(vars.Map))

	s.svcs = make(map[string]*happy.ServiceStatus)

	bi, ok := debug.ReadBuildInfo()
	if ok {
		for _, dep := range bi.Deps {
			d := happy.DependencyInfo{
				Path:    dep.Path,
				Version: dep.Version,
				Sum:     dep.Sum,
			}
			s.deps = append(s.deps, d)
		}
		s.debug.Store("go.version", bi.GoVersion)
		s.debug.Store("go.path", bi.Path)
		// The module containing the main package
		s.debug.Store("go.module.path", bi.Main.Path)
		s.debug.Store("go.module.version", bi.Main.Version)
		s.debug.Store("go.module.sum", bi.Main.Sum)
		if bi.Main.Replace != nil {
			s.debug.Store("go.module.replace.path", bi.Main.Replace.Path)
			s.debug.Store("go.module.replace.version", bi.Main.Replace.Version)
			s.debug.Store("go.module.replace.sum", bi.Main.Replace.Sum)
		} else {
			s.debug.Store("go.module.replace", nil)
		}

		if bi.Settings != nil {
			for _, setting := range bi.Settings {
				s.debug.Store(fmt.Sprintf("go.module.settings.%s", setting.Key), setting.Value)
			}
		} else {
			s.debug.Store("go.module.settings", nil)
		}
	}

	s.debug.Store("core.go.arch", runtime.GOARCH)
	s.debug.Store("core.go.os", runtime.GOOS)
	// https://pkg.go.dev/runtime#pkg-variables
	s.debug.Store("core.go.mem.profile.rate", runtime.MemProfileRate)
	// Compiler is the name of the compiler toolchain that built the running binary. Known toolchains are:
	s.debug.Store("core.go.compiler", runtime.Compiler)

	e, err := os.Executable()
	if err != nil {
		return err
	}
	s.debug.Store("os.executable", e)
	// effective group id of the caller.
	s.debug.Store("os.egid", os.Getegid())
	s.debug.Store("os.euid", os.Geteuid())
	gs, err := os.Getgroups()
	if err != nil {
		return err
	}
	for _, g := range gs {
		s.debug.Store(fmt.Sprintf("os.groups.%d", g), true)
	}
	s.debug.Store("os.pagesize", os.Getpagesize())
	s.debug.Store("os.pid", os.Getpid())
	s.debug.Store("os.ppid", os.Getppid())
	s.debug.Store("os.user.uid", os.Getuid())
	s.debug.Store("os.user.gid", os.Getgid())
	return nil
}

func (s *Status) setServiceStatus(url, key string, val any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.svcs[url]; !ok {
		s.svcs[url] = &happy.ServiceStatus{
			URL: url,
		}
	}
	v, err := vars.NewVariable(key, val, true)
	if err != nil {
		return err
	}
	switch key {
	case "registered":
		s.svcs[url].Registered = v.Bool()
	case "running":
		s.svcs[url].Running = v.Bool()
	case "failed":
		s.svcs[url].Failed = v.Bool()
	case "err":
		s.svcs[url].Err = v.String()
	case "started.at":
		if v, ok := val.(time.Time); ok {
			s.svcs[url].StartedAt = v
		}
	case "stopped.at":
		if v, ok := val.(time.Time); ok {
			s.svcs[url].StoppedAt = v
		}
	}
	return nil
}

func (s *Status) Started() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

func (s *Status) Elapsed() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped.IsZero() {
		return time.Since(s.started)
	}
	return s.stopped.Sub(s.started)
}

func (s *Status) TotalEvents() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.evs
}

func (s *Status) Addons() []happy.AddonInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addons
}

func (s *Status) Dependencies() []happy.DependencyInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deps
}

func (s *Status) DebugInfo() happy.Variables {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.debug
}

func (s *Status) Services() (svcs []happy.ServiceStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, svc := range s.svcs {
		svcs = append(svcs, *svc)
	}
	return
}

func (s *Status) GetServiceStatus(url happy.URL) (happy.ServiceStatus, happy.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if svss, ok := s.svcs[url.String()]; ok {
		return *svss, nil
	}
	return happy.ServiceStatus{}, ErrMonitor.WithTextf("service not registered: %s", url)
}
