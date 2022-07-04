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
	"fmt"
	"sync"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/config"
)

type Manager struct {
	addons   []happy.Addon
	registry sync.Map
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Register(addons ...happy.Addon) error {
	var err error
	for _, addon := range addons {
		if addon == nil {
			err = fmt.Errorf("%w: registering nil addon", err)
			continue
		}
		id := fmt.Sprintf("%p", addon)
		if _, registered := m.registry.Load(id); registered {
			err = fmt.Errorf("%w: addon %q already registered", ErrAddonRegister, addon.Name())
			continue
		}
		if !config.ValidSlug(addon.Slug()) {
			err = fmt.Errorf("%w: invalid addon slug %q", ErrAddonRegister, addon.Slug())
			continue
		}
		m.registry.Store(id, struct{}{})
		m.addons = append(m.addons, addon)
	}
	return err
}

func (m *Manager) Addons() []happy.Addon {
	return m.addons
}
