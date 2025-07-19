// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

import (
	"fmt"
	"iter"
	"sort"
	"sync"

	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/version"
	"golang.org/x/text/language"
)

type Profile struct {
	mu       sync.RWMutex
	name     string
	lang     language.Tag
	schema   Schema
	loaded   bool
	changed  bool
	settings map[string]Setting
}

func (p *Profile) Name() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.name
}

func (p *Profile) Lang() language.Tag {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lang
}

func (p *Profile) All() iter.Seq[Setting] {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.all()
}

func (p *Profile) all() iter.Seq[Setting] {
	return func(yield func(Setting) bool) {
		var settings []Setting
		for _, setting := range p.settings {
			settings = append(settings, setting)
		}

		sort.Slice(settings, func(i, j int) bool {
			return settings[i].key < settings[j].key
		})

		for _, setting := range settings {
			if !yield(setting) {
				return
			}
		}
	}
}

// Loaded reports true when settings profile is loaded
// and optional user preferences are applied.
func (p *Profile) Loaded() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.loaded
}

func (p *Profile) Version() version.Version {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.schema.version
}
func (p *Profile) PackageSettingsStructName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.schema.pkgSettingsStructName
}

func (p *Profile) Module() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.schema.module
}

func (p *Profile) Get(key string) Setting {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.settings == nil {
		return Setting{}
	}
	s, ok := p.settings[key]
	if ok {
		return s
	}
	return Setting{}
}
func (p *Profile) Has(key string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.settings == nil {
		return false
	}
	_, ok := p.settings[key]
	return ok
}

func (p *Profile) Set(key string, val any) (err error) {
	if !p.Has(key) {
		return fmt.Errorf("setting not found %s", key)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	setting := p.settings[key]

	if setting.mutability == SettingImmutable {
		return fmt.Errorf("setting is immutable %s", key)
	} else if setting.isSet && setting.mutability == SettingOnce {
		return fmt.Errorf("setting can only be set once %s", key)
	}
	setting.vv, err = vars.NewAs(key, val, true, vars.Kind(setting.kind))
	if err != nil {
		return fmt.Errorf("%w: key(%s) %s", ErrProfile, key, err.Error())
	}

	for _, v := range p.schema.settings[key].validators {
		if err := v.fn(setting); err != nil {
			return err
		}
	}

	if prev, ok := p.settings[key]; ok {
		if prev.vv.String() != setting.vv.String() {
			p.changed = true
		}
	} else {
		p.changed = true
	}

	setting.isSet = true

	p.settings[key] = setting
	return nil
}

func (p *Profile) Changed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.changed
}

func (p *Profile) ValidatePreference(key string, val any) (err error) {
	if !p.Has(key) {
		return fmt.Errorf("setting not found %s", key)
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	setting := p.settings[key]

	if !setting.Persistent() {
		return fmt.Errorf("setting is not persistent %s", key)
	}

	if setting.Mutability() == SettingImmutable {
		return fmt.Errorf("setting is immutable %s", key)
	}

	setting.vv, err = vars.NewAs(key, val, true, vars.Kind(setting.kind))
	if err != nil {
		return fmt.Errorf("%w: key(%s) %s", ErrProfile, key, err.Error())
	}

	for _, v := range p.schema.settings[key].validators {
		if err := v.fn(setting); err != nil {
			return err
		}
	}

	return nil
}

// Preferences returns the profile's preferences which can be saved.
func (p *Profile) Preferences() *Preferences {
	p.mu.RLock()
	defer p.mu.RUnlock()

	prefs := NewPreferences(p.schema.version)
	for setting := range p.all() {
		if setting.IsSet() && setting.Persistent() {
			prefs.Set(setting.Key(), setting.Value().String())
		}
	}

	return prefs
}

func (p *Profile) load(prefs *Preferences) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.loaded {
		return fmt.Errorf("%w: already loaded", ErrProfile)
	}
	p.settings = make(map[string]Setting)
	for _, spec := range p.schema.settings {
		setting, err := spec.Setting(p.lang)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrProfile, err.Error())
		}
		p.settings[spec.Key] = setting
	}

	if prefs == nil {
		p.loaded = true
		return nil
	}
	for key, val := range prefs.data {
		lkey := key
		s, ok := p.settings[lkey]
		if !ok && p.schema.migrations != nil {
			if to, has := p.schema.migrations[lkey]; has {
				lkey = to
				s, ok = p.settings[lkey]
			}
		}

		if ok {
			s.vv, err = vars.NewAs(lkey, val, true, vars.Kind(s.kind))
			if err != nil {
				return fmt.Errorf("%w: preferences key(%s) %s", ErrProfile, lkey, err.Error())
			}
			s.isSet = true

			for _, v := range p.schema.settings[lkey].validators {
				if err := v.fn(s); err != nil {
					return err
				}
			}
			p.settings[lkey] = s
		} else {
			// return fmt.Errorf("%w: preferences provided key(%s) not found", ErrProfile, lkey)
			prefs.data[lkey] = ""
		}
	}
	p.loaded = true
	return nil
}
