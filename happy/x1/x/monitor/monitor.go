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
	"sync"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
)

var ErrMonitor = happyx.NewError("monitor error")

type Monitor struct {
	mu     sync.Mutex
	status *Status
}

func New(opts ...happy.OptionSetFunc) *Monitor {
	m := &Monitor{
		status: &Status{},
	}
	return m
}

func (m *Monitor) Start(sess happy.Session) happy.Error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.status.start(); err != nil {
		return ErrMonitor.Wrap(err)
	}

	m.status.started = time.Now()
	return nil
}

func (m *Monitor) Stop() happy.Error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.stopped = time.Now()
	return nil
}

func (m *Monitor) Status() happy.ApplicationStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

// happy.EventListener interface
func (m *Monitor) OnAnyEvent(cb happy.ActionWithEventFunc) {

}

func (m *Monitor) OnEvent(scope, key string, cb happy.ActionWithEventFunc) {

}

func (m *Monitor) RegisterAddon(ainfo happy.AddonInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.addons = append(m.status.addons, ainfo)
}

func (m *Monitor) SetServiceStatus(url, key string, val any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.setServiceStatus(url, key, val)
}
