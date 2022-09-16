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

package sdk

import (
	"github.com/mkungla/happy"
	// "github.com/mkungla/happy/x/happyx"
)

type AddonSkeleton struct {
	name    string
	version Version
}

func NewAddonSkeleton(name, version string) *AddonSkeleton {
	return &AddonSkeleton{
		name: name,
		version: Version{
			s: version,
		},
	}
}

func (a *AddonSkeleton) Cronjobs() {}

func (a *AddonSkeleton) Name() string           { return a.name }
func (a *AddonSkeleton) Slug() happy.Slug       { return Slug{} }
func (a *AddonSkeleton) Version() happy.Version { return a.version }
func (a *AddonSkeleton) Options() happy.OptionDefaultsSetter {
	return nil
}
func (a *AddonSkeleton) Commands() []happy.Command {
	return nil
}
func (a *AddonSkeleton) Services() []happy.Service {
	return nil
}

type Slug struct {
	s string
}

func (s Slug) Valid() bool {
	return false
}

func (s Slug) String() string {
	return s.s
}

type Version struct {
	s string
}

func (v Version) String() string {
	return v.s
}

type URL struct {
	s string
}

func (u URL) String() string {
	return u.s
}

type AddonFactory struct {
	Addon happy.Addon
}

func (a *AddonFactory) GetAddonCreateFunc() happy.AddonCreateFunc {
	return func() (happy.Addon, happy.Error) {
		return a.Addon, nil
	}
}

type ServiceFactory struct {
	Default happy.Service
}

func (a *ServiceFactory) Service() (happy.Service, happy.Error) {
	return a.Default, nil
}

type ServiceSkeleton struct {
}

func (a *ServiceSkeleton) Slug() happy.Slug { return Slug{} }

func (a *ServiceSkeleton) URL() happy.URL { return URL{} }

func (a *ServiceSkeleton) OnInitialize(happy.ActionFunc) {}

func (a *ServiceSkeleton) OnStart(happy.ActionWithArgsFunc) {}

func (a *ServiceSkeleton) OnStop(happy.ActionFunc) {}

func (a *ServiceSkeleton) OnRequest(happy.ServiceRouter)                    {}
func (a *ServiceSkeleton) OnTick(happy.ActionTickFunc)                      {}
func (a *ServiceSkeleton) OnTock(happy.ActionTickFunc)                      {}
func (a *ServiceSkeleton) OnEvent(key string, cb happy.ActionWithEventFunc) {}
func (a *ServiceSkeleton) OnAnyEvent(happy.ActionWithEventFunc)             {}
