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

//go:build (linux && !android) || freebsd || windows || openbsd || darwin || !js

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mkungla/happy"
)

func (a *Application) addCommand(c happy.Command) {
	if c == nil {
		return
	}

	if a.commands == nil {
		a.commands = make(map[string]happy.Command)
	}

	// Can only check command name here since nothing stops you to add possible
	// shadow flags after this command was added.
	if _, exists := a.commands[c.String()]; exists {
		a.logger.Errorf("command (%s) is already in use, can not add command", c.String())
		return
	}

	a.commands[c.String()] = c
}

func (a *Application) initLocalhost() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		a.addAppErr(err)
		return
	}
	a.addAppErr(a.session.Set("host.path.home", homeDir))

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		a.addAppErr(err)
		return
	}
	cachepath := filepath.Join(cacheDir, a.Config().Slug)
	if err := os.MkdirAll(cachepath, 0700); err != nil {
		a.addAppErr(err)
		return
	}
	a.addAppErr(a.session.Set("host.path.cache", cachepath))

	configDir, err := os.UserConfigDir()
	if err != nil {
		a.addAppErr(err)
		return
	}
	configpath := filepath.Join(configDir, a.Config().Slug)
	if err := os.MkdirAll(configpath, 0700); err != nil {
		a.addAppErr(err)
		return
	}
	a.addAppErr(a.session.Set("host.path.config", configpath))

	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d", a.Config().Slug, time.Now().UnixMilli()))
	if err = os.Mkdir(tempDir, 0700); err != nil {
		a.addAppErr(err)
		return
	}
	a.addAppErr(a.session.Set("host.path.tmp", tempDir))
}

func (a *Application) removeDir(logId, dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		a.Log().SystemDebugf("removing %s dir: %s", logId, dir)
		return os.RemoveAll(dir)
	}
	return fmt.Errorf("removing %s dir: %w - %q", logId, err, dir)
}
