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
)

type Configurator struct {
	logger  happy.Logger
	session happy.Session
	monitor happy.ApplicationMonitor
	assets  happy.FS
	engine  happy.Engine
}

func New(opts ...happy.OptionWriteFunc) *Configurator {
	return &Configurator{}
}

// API
func (c *Configurator) UseLogger(happy.Logger)                 {}
func (c *Configurator) GetLogger() (happy.Logger, happy.Error) { return c.logger, nil }

func (c *Configurator) UseSession(happy.Session) {}
func (c *Configurator) GetSession() (happy.Session, happy.Error) {
	return c.session, nil
}

func (c *Configurator) UseMonitor(happy.ApplicationMonitor) {}
func (c *Configurator) GetMonitor() (happy.ApplicationMonitor, happy.Error) {
	return c.monitor, nil
}

func (c *Configurator) UseAssets(happy.FS)                 {}
func (c *Configurator) GetAssets() (happy.FS, happy.Error) { return c.assets, nil }

func (c *Configurator) UseEngine(happy.Engine)                 {}
func (c *Configurator) GetEngine() (happy.Engine, happy.Error) { return c.engine, nil }

// happy.OptionDefaultsWriter interface
func (c *Configurator) SetOptionDefault(happy.Variable) happy.Error {
	return happyx.Errorf("Configurator.SetOptionDefault: %w", happyx.ErrNotImplemented)
}
func (c *Configurator) SetOptionDefaultKeyValue(key string, val any) happy.Error {
	return happyx.Errorf("Configurator.SetOptionDefaultKeyValue: %w", happyx.ErrNotImplemented)
}
func (c *Configurator) SetOptionsDefaultFuncs(vfuncs ...happy.VariableParseFunc) happy.Error {
	return happyx.Errorf("Configurator.SetOptionsDefaultFuncs: %w", happyx.ErrNotImplemented)
}
