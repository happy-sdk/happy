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

package addon

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
)

var ErrAddon = happyx.NewError("addon error")

type Addon struct {
	slug    happy.Slug
	name    string
	version happy.Version
	opts    happy.Variables
}

func New(slug, name string, version happy.Version, defaultOptions ...happy.OptionSetFunc) (happy.Addon, happy.Error) {
	s, err := happyx.NewSlug(slug)
	if err != nil {
		return nil, ErrAddon.Wrap(err)
	}

	opts, err := happyx.OptionsToVariables(defaultOptions...)
	if err != nil {
		return nil, ErrAddon.Wrap(err)
	}

	addon := &Addon{
		slug:    s,
		name:    name,
		version: version,
		opts:    opts,
	}
	return addon, nil
}

func (a *Addon) Cronjobs() {}

func (a *Addon) Name() string           { return a.name }
func (a *Addon) Slug() happy.Slug       { return a.slug }
func (a *Addon) Version() happy.Version { return a.version }

func (a *Addon) Commands() []happy.Command {
	return nil
}
func (a *Addon) Services() []happy.Service {
	return nil
}

func (a *Addon) API() happy.API {
	return nil
}

func (a *Addon) SetOption(v happy.Variable) happy.Error {
	return a.SetOptionKeyValue(v.Key(), v.Value())
}

func (a *Addon) SetOptionKeyValue(key string, val any) happy.Error {
	if !a.opts.Has(key) {
		return ErrAddon.WithTextf("unknown option %s for addon %s", key, a.slug)
	}
	if a.opts.Get(key).ReadOnly() {
		return ErrAddon.WithTextf("option %s for addon %s is read only", val, a.slug)
	}
	a.opts.Store(key, val)
	return nil
}

func (a *Addon) SetOptionValue(key string, val happy.Value) happy.Error {
	return a.SetOptionKeyValue(key, val)
}

func (a *Addon) DeleteOption(key string) happy.Error {
	return ErrAddon.WithText("addon option can not be deleted")
}

func (a *Addon) ResetOptions() happy.Error {
	return ErrAddon.WithText("can not reset addon options")
}

func (a *Addon) GetOptions() happy.Variables {
	opts, _ := a.opts.LoadWithPrefix("") // make copy
	return opts
}

func (a *Addon) GetOption(key string) (happy.Variable, happy.Error) {
	if !a.opts.Has(key) {
		return nil, happyx.ErrOption.WithTextf("addon does not have option %s", key)
	}
	return a.opts.Get(key), nil
}

func (a *Addon) GetOptionOrDefault(key string, defval any) (val happy.Variable) {
	opt, _ := a.opts.LoadOrDefault(key, defval)
	return opt
}

func (a *Addon) HasOption(key string) bool {
	return a.opts.Has(key)
}

func (a *Addon) GetOptionSetFunc(srcKey, destKey string) happy.OptionSetFunc {
	return func(opts happy.OptionSetter) happy.Error {
		o, loaded := a.opts.Load(srcKey)
		if !loaded {
			return ErrAddon.WithTextf("addon %s does not have option %s", a.slug, srcKey)
		}
		return opts.SetOptionValue(destKey, o.Value())
	}
}

func (a *Addon) RangeOptions(f func(opt happy.Variable) bool) {
	a.opts.Range(f)
}
