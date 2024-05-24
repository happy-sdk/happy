// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"golang.org/x/text/language"
)

var (
	ErrSchema = errors.New("schema")
)

type Schema struct {
	id         string
	pkg        string
	module     string
	mode       ExecutionMode
	version    string
	settings   map[string]SettingSpec
	migrations map[string]string
}

func (s *Schema) set(key string, spec SettingSpec) error {
	if s.settings == nil {
		s.settings = make(map[string]SettingSpec)
	}
	if err := spec.Validate(); err != nil {
		return err
	}
	s.settings[key] = spec
	return nil
}

func (s *Schema) Profile(name string, pref *Preferences) (*Profile, error) {
	profile := &Profile{
		name:   name,
		schema: *s,
		lang:   language.English,
	}
	if err := profile.load(pref); err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *Schema) setID() {
	// Generate the ID using SHA-256 on the combined package path and execution mode.
	data := s.pkg + "-" + s.module + "-" + s.mode.String()
	hash := sha256.Sum256([]byte(data))
	s.id = fmt.Sprintf("%x", hash)
}
