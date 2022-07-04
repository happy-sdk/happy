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

package service

import (
	"fmt"
	"sync"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/config"
)

type Manager struct {
	services []happy.Service
	registry sync.Map
}

func NewManager() *Manager {
	return &Manager{}
}

func (sm *Manager) Stop() error {
	return nil
}

func (sm *Manager) Len() int {
	return 0
}

func (m *Manager) Register(services ...happy.Service) error {
	var err error
	for _, service := range services {
		id := fmt.Sprintf("%p", service)
		if _, registered := m.registry.Load(id); registered {
			err = fmt.Errorf("%w: service %q already registered", ErrServiceRegister, service.Name())
			continue
		}
		if !config.ValidSlug(service.Slug()) {
			err = fmt.Errorf("%w: invalid service slug %q", ErrServiceRegister, service.Slug())
			continue
		}
		m.registry.Store(id, struct{}{})
		m.services = append(m.services, service)
	}
	return err
}

func (m *Manager) Initialize(ctx happy.Session, log happy.Logger, keepAlive bool) error {
	return nil
}
