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

package webserver

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/sdk"
)

type WebServerAddon struct {
	happy.Addon
	fs happy.FS
}

func New(option ...happy.OptionSetFunc) (*WebServerAddon, happy.Error) {
	a, err := sdk.NewAddon(
		"webserver",
		"Web server",
		"0.1.0",
	)
	if err != nil {
		return nil, err
	}
	addon := &WebServerAddon{
		Addon: a,
	}
	for _, opt := range option {
		if err := opt(addon); err != nil {
			return nil, sdk.ErrAddon.Wrap(err)
		}
	}

	return addon, nil
}

func (a *WebServerAddon) Cronjobs() {}

func (a *WebServerAddon) Options() happy.OptionDefaultsSetter {
	return nil
}
func (a *WebServerAddon) Commands() []happy.Command {
	return nil
}
func (a *WebServerAddon) Services() []happy.Service {
	return nil
}

func (a *WebServerAddon) FS(fs happy.FS) {
	a.fs = fs
}
