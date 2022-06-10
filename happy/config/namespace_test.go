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

package config_test

import (
	"testing"

	"github.com/mkungla/happy/config"
)

func TestNamespaceFromCurrentModule(t *testing.T) {
	if ns := config.NamespaceFromCurrentModule(); ns != "com.github.mkungla.happy.config" {
		t.Errorf("CurrentModuleNamespace() returned %q, want com.github.mkungla.happy.config", ns)
	}
}

type nstest struct {
	from  string
	want  string
	valid bool
}

func namespaceTests() []nstest {
	return []nstest{
		{"howi", "howi", true},
		{"github.com/mkungla/happy", "com.github.mkungla.happy", true},
		{"happy", "happy", true},
		{"happy ", "happy", false},
		{"happy|sdk", "happysdk", false},
	}
}

func TestNamespaceFromModulePath(t *testing.T) {
	for _, tt := range namespaceTests() {
		if got := config.NamespaceFromModulePath(tt.from); got != tt.want {
			t.Errorf("Create(%q) = %v, want %v", tt.from, got, tt.want)
		}
	}
}

func TestValid(t *testing.T) {
	for _, tt := range namespaceTests() {
		if got := config.ValidNamespace(tt.from); got != tt.valid {
			t.Errorf("Valid(%q) = %t, want %t", tt.from, got, tt.valid)
		}
	}
}
