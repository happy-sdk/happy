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

package config

import (
	"errors"
	"regexp"

	"github.com/mkungla/vars/v6"
)

var (
	ErrInvalidSettingsKey   = errors.New("invalid setting key")
	ErrInvalidSettingsGroup = errors.New("invalid setting group")
	ErrSettingKeyNotUnique  = errors.New("setting key must be unique")
)

type (
	Settings struct {
		General SettingsGroup
		Groups  []SettingsGroup
	}

	SettingsGroup struct {
		Key      string
		Title    string
		Desc     string
		Settings []Setting
	}

	Setting struct {
		// once        sync.Once
		FieldType   string
		Once        func() error
		Key         string
		Label       string
		Description string
		Value       vars.Value
		Hidden      bool
		Helper      string
		Secure      bool
		Validate    func(val vars.Value) error
		Default     func() vars.Value
	}
)

func ValidSettingKey(key string) bool {
	re := regexp.MustCompile("^[a-z][a-z0-9.]*[a-z0-9]$")
	return re.MatchString(key)
}

func DefaultSettings() Settings {
	return Settings{}
}

func (s *Settings) Add(settings ...Setting) {
	s.General.Add(settings...)
}

func (g *SettingsGroup) Add(settings ...Setting) {
	g.Settings = append(g.Settings, settings...)
}
