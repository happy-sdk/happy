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

package configurator

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"github.com/mkungla/happy/x/pkg/vars"
)

type Configurator struct {
	logger  happy.Logger
	session happy.Session
	monitor happy.ApplicationMonitor
	assets  happy.FS
	engine  happy.Engine

	config *config
}

func New(opts ...happy.OptionWriteFunc) (*Configurator, happy.Error) {
	c := &Configurator{
		config: new(config),
	}

	for _, opt := range opts {
		if err := opt(c.config); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// UseLogger sets and configures the logger which can be used by application.
func (c *Configurator) UseLogger(logger happy.Logger) {
	m := c.config.m.ExtractWithPrefix("log.")
	m.Range(func(v vars.Variable) bool {
		vv := vars.AsVariable[happy.Variable, happy.Value](v)
		if err := logger.SetOptionDefault(vv); err != nil {
			logger.Emergency(err)
		}
		return true
	})
	c.logger = logger
}

func (c *Configurator) GetLogger() (happy.Logger, happy.Error) { return c.logger, nil }

func (c *Configurator) UseSession(session happy.Session) {
	c.session = session
}
func (c *Configurator) GetSession() (happy.Session, happy.Error) {
	return c.session, nil
}

func (c *Configurator) UseMonitor(monitor happy.ApplicationMonitor) {
	c.monitor = monitor
}
func (c *Configurator) GetMonitor() (happy.ApplicationMonitor, happy.Error) {
	return c.monitor, nil
}

func (c *Configurator) UseAssets(assets happy.FS) {
	c.assets = assets
}
func (c *Configurator) GetAssets() (happy.FS, happy.Error) { return c.assets, nil }

func (c *Configurator) UseEngine(engine happy.Engine) {
	c.engine = engine
}
func (c *Configurator) GetEngine() (happy.Engine, happy.Error) { return c.engine, nil }

// happy.OptionDefaultsWriter interface
func (c *Configurator) SetOptionDefault(happy.Variable) happy.Error {
	return happyx.NotImplementedError("configurator.SetOptionDefault")
}

func (c *Configurator) SetOptionDefaultKeyValue(key string, val any) happy.Error {
	return happyx.NotImplementedError("configurator.SetOptionDefault")
}

func (c *Configurator) SetOptionsDefaultFuncs(vfuncs ...happy.VariableParseFunc) happy.Error {
	return happyx.NotImplementedError("configurator.SetOptionsDefaultFuncs")
}

func (c *Configurator) GetApplicationOptions() happy.Variables {
	m := c.config.m.ExtractWithPrefix("app.")
	if m == nil {
		return nil
	}
	return vars.AsMap[happy.Variables, happy.Variable, happy.Value](m)
}

type config struct {
	m vars.Map
}

func (c *config) Write(p []byte) (int, error) {
	return 0, happyx.Errorf("%w: configurator .Write", happyx.ErrNotImplemented)
}

func (c *config) SetOption(v happy.Variable) happy.Error {
	if v == nil {
		return happyx.NewError("configurator .SetOption got nil argument")
	}
	c.m.Store(v.Key(), v.Value())
	return nil
}

func (c *config) SetOptionKeyValue(key string, val any) happy.Error {
	k, err := vars.ParseKey(key)
	if err != nil {
		return happyx.ErrOption.Wrap(err)
	}
	c.m.Store(k, val)
	return nil
}

func (c *config) SetOptionValue(key string, val happy.Value) happy.Error {
	k, err := vars.ParseKey(key)
	if err != nil {
		return happyx.ErrOption.Wrap(err)
	}
	c.m.Store(k, val)
	return nil
}
