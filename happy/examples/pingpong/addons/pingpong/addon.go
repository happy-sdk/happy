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

package pingpong

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/config"
	"github.com/mkungla/happy/version"
)

func New() (happy.Addon, error) {
	v, err := version.Parse("0.1.0")
	if err != nil {
		return nil, err
	}
	return &Addon{
		version: v,
	}, nil
}

type Addon struct {
	version happy.Version
}

func (a Addon) Name() string                                      { return "Ping Pong" }
func (a Addon) Slug() string                                      { return "pingpong" }
func (a Addon) Description() string                               { return "ping pong example" }
func (a Addon) Version() happy.Version                            { return a.version }
func (a Addon) Configured(ctx happy.Session) bool                 { return true }
func (a Addon) DefaultSettings(ctx happy.Session) config.Settings { return config.DefaultSettings() }

func (a Addon) Commands() ([]happy.Command, error) {
	var cmds []happy.Command

	for _, cfn := range []func() (happy.Command, error){
		cmdStart,
	} {
		cmd, err := cfn()
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}

	return cmds, nil
}

func (a Addon) Services() []happy.Service {
	return []happy.Service{
		&MonitorService{},
		NewPeerService(1), // first instance
		NewPeerService(2), // second instance
		NewPeerService(3), // second instance
	}
}
