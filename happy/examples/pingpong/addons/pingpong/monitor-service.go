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
	"errors"
	"sync"
	"time"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/addon"
	"github.com/mkungla/vars/v6"
)

type MonitorService struct {
	mu sync.Mutex

	total       uint64
	totalEvents uint64
	enabled     bool
	firstEvTS   time.Time
	lastEvTS    time.Time
}

func (m *MonitorService) Name() string                       { return "Monitor" }
func (m *MonitorService) Slug() string                       { return "monitor" }
func (m *MonitorService) Description() string                { return "monitor service" }
func (m *MonitorService) Version() happy.Version             { return addon.Version{} }
func (m *MonitorService) Initialize(ctx happy.Session) error { return nil }
func (m *MonitorService) Start(ctx happy.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !ctx.Has("pingpong.exit.after") {
		return errors.New("monitor: failed to read pingpong.max value")
	}
	m.total = ctx.Get("pingpong.exit.after").Uint64()
	ctx.Log().Outf("monitor: starting ping pong... with %d iterations", m.total)
	m.enabled = true

	return nil
}

func (m *MonitorService) Tick(ctx happy.Session, ts time.Time, delta time.Duration) error { return nil }

func (m *MonitorService) OnEvent(ctx happy.Session, ev happy.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ev.Key != "ping" && ev.Key != "pong" {
		return
	}

	if !m.enabled {
		ctx.Log().Errorf("monitor: recieved %s event while monitor was not ready", ev.Key)
		return
	}

	if m.totalEvents == 0 {
		m.firstEvTS = time.Now().UTC()
	} else {
		m.lastEvTS = time.Now().UTC()
	}
	m.totalEvents++
	ctx.Log().Outf("monitor: %s %s -> %s", ev.Key, ev.Payload.Get("src").String(), ev.Payload.Get("dest").String())
	if m.totalEvents > m.total {
		ctx.Destroy(nil)
	}
}

func (m *MonitorService) Stop(ctx happy.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled {
		return errors.New("monitor service was never started")
	}

	ctx.Log().Outf("monitor: recorded %d events", m.totalEvents)
	if m.totalEvents == 0 {
		return nil
	}
	ctx.Log().Outf("monitor: event queue took %s", m.lastEvTS.Sub(m.firstEvTS))
	return nil
}

func (m *MonitorService) Call(fn string, args ...vars.Variable) (any, error) {
	return nil, nil
}
