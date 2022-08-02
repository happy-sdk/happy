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

// Package settings provides user session settings for happy applications.
package settings

import (
	"errors"

	"github.com/mkungla/vars/v6"
)

type Map struct {
	// Prop `settings:"default-val"`
	data     *vars.Collection
	defaults map[string]vars.Variable
	// path     string
}

func New() *Map {
	return &Map{
		defaults: make(map[string]vars.Variable),
		data:     new(vars.Collection),
	}
}

// Get retrieves the value of the variable named by the key.
// It returns the value, which will be empty string if the
// variable is not set or value was empty or default value
// if default value was provided.
// Do make default value persistent use .Default instead.
func (m *Map) Getd(key string, defval ...any) (val vars.Value) {
	return m.data.Get(key, defval...)
}

func (m *Map) Get(key string) (val vars.Value) {
	return m.data.Get(key)
}

// Set updates key value pair in settings collection.
// If key does not exist then sets key with given value.
func (m *Map) Set(key string, val any) error {
	m.data.Set(key, val)
	return nil
}

// Set updates key value pair in settings collection.
// If key does not exist then sets key with given value.
func (m *Map) Store(key string, val any) error {
	m.data.Store(key, val)
	return nil
}

// Has result indicates whether this setting has been set.
func (m *Map) Has(key string) bool {
	return m.data.Has(key)
}

// Delete setting by given key
func (m *Map) Delete(key string) {
	m.data.Delete(key)
}

// Default sets default value for key.
func (m *Map) Default(key string, val any) {
	if _, ok := m.defaults[key]; ok {
		return
	}

	v := vars.New(key, val)
	m.defaults[key] = v

	if !m.data.Has(key) {
		_ = m.Set(key, val)
	}
}

func (m *Map) Range(f func(key string, value vars.Value) bool) {
	m.data.Range(f)
}

func (m *Map) LoadFile() error {
	return errors.New("settings.LoadFile not impl")
}
func (m *Map) SaveFile() error {
	return errors.New("settings.SaveFile not impl")
}
