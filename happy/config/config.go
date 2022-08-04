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

// Package config enables you to configure happy application instance.
package config

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/exp/slices"

	"github.com/mkungla/vars/v6"
)

const (
	// NamespaceMustCompile against following expression.
	SlugRe = "^[a-zA-Z][a-zA-Z0-9_.-]*[a-zA-Z0-9]$"
)

type (
	Config struct {
		Title          string
		Slug           string
		Namespace      string
		Description    string
		CopyrightSince int
		CopyrightBy    string
		License        string
	}

	// Replacement structure.
	replacement struct {
		re *regexp.Regexp
		ch string
	}
)

var (
	ErrInvalidSlug      = errors.New("invalid slug")
	ErrInvalidNamespace = errors.New("invalid namspace")
	ErrSetConfigOpt     = errors.New("can not set config option at runtime")

	Keys = []string{
		"app.title",
		"app.slug",
		"app.namespace",
		"app.description",
		"app.copyright.by",
		"app.copyright.since",
		"app.license",
	}

	// regexps and replacements.
	rExps = []replacement{ //nolint:gochecknoglobals
		{re: regexp.MustCompile(`[\xC0-\xC6]`), ch: "A"},
		{re: regexp.MustCompile(`[\xE0-\xE6]`), ch: "a"},
		{re: regexp.MustCompile(`[\xC8-\xCB]`), ch: "E"},
		{re: regexp.MustCompile(`[\xE8-\xEB]`), ch: "e"},
		{re: regexp.MustCompile(`[\xCC-\xCF]`), ch: "I"},
		{re: regexp.MustCompile(`[\xEC-\xEF]`), ch: "i"},
		{re: regexp.MustCompile(`[\xD2-\xD6]`), ch: "O"},
		{re: regexp.MustCompile(`[\xF2-\xF6]`), ch: "o"},
		{re: regexp.MustCompile(`[\xD9-\xDC]`), ch: "U"},
		{re: regexp.MustCompile(`[\xF9-\xFC]`), ch: "u"},
		{re: regexp.MustCompile(`[\xC7-\xE7]`), ch: "c"},
		{re: regexp.MustCompile(`[\xD1]`), ch: "N"},
		{re: regexp.MustCompile(`[\xF1]`), ch: "n"},
	}
	spacereg       = regexp.MustCompile(`\s+`)
	noncharreg     = regexp.MustCompile(`[^A-Za-z0-9-]`)
	minusrepeatreg = regexp.MustCompile(`\-{2,}`)
	alnum          = &unicode.RangeTable{ //nolint:gochecknoglobals
		R16: []unicode.Range16{
			{'0', '9', 1},
			{'A', 'Z', 1},
			{'a', 'z', 1},
		},
	}
)

func New() Config {
	c := Config{}
	c.Namespace = NamespaceFromCurrentModule()
	nparts := strings.Split(c.Namespace, ".")
	slug := nparts[len(nparts)-1]
	c.Slug = slug
	c.Title = CreateCamelCaseSlug(slug)
	return c
}

func (c Config) Get(key string) vars.Value {
	return vars.NewValue("")
}

func (c Config) Set(key string, val any) error {
	return ErrSetConfigOpt
}
func (c Config) Store(key string, val any) error {
	return ErrSetConfigOpt
}

func (c Config) Has(key string) bool {
	return slices.Contains(Keys, key)
}
