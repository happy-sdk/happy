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
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
)

type Monitor struct {
}

func New(opts ...happy.OptionWriteFunc) *Monitor {
	return &Monitor{}
}

func (m *Monitor) Start() happy.Error {
	return happyx.Errorf("Monitor.Start: %w", happyx.ErrNotImplemented)
}

func (m *Monitor) Stop() happy.Error {
	return happyx.Errorf("Monitor.Stop: %w", happyx.ErrNotImplemented)
}

func (m *Monitor) Stats() happy.ApplicationStats { return nil }

// happy.EventListener interface
func (m *Monitor) OnAnyEvent(cb happy.ActionWithEventFunc)          {}
func (m *Monitor) OnEvent(key string, cb happy.ActionWithEventFunc) {}
