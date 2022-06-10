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
	"strings"
	"testing"
)

func TestCreateCamelCaseSlug(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"some-str", "SomeStr"},
		{"some-str ", "SomeStr"},
		{" some-str", "SomeStr"},
		{" some-str ", "SomeStr"},
		{"some str", "SomeStr"},
		{"SoMe STr", "SomeStr"},
		{"@SoMe!STr", "SomeStr"},
	}
	for _, tt := range tests {
		if got := CreateCamelCaseSlug(tt.in); got != strings.TrimSpace(tt.want) {
			t.Errorf("ToCamelCaseAlnum(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestValidSlug(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"n", true},
		{"name-space", true},
		{"name_space", true},
		{"2", false},
		{"NameSpace", true},
		{"NameSpacE", true},
		{"NameSpace2", true},
		{"2NameSpace", false},
		{"name space", false},
		{"name_space ", false},
		{" name_space", false},
		{"name_space_", false},
		{"_name_space", false},
		{"name_space-", false},
		{"-name_space", false},
		{"CamelCase ", false},
		{"~abc", false},
		{"a@bc", false},
	}
	for _, tt := range tests {
		if got := ValidSlug(tt.in); got != tt.want {
			t.Errorf("ValidSlug(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestCreateSlug(t *testing.T) {
	if slug, expected := CreateSlug("test->àèâ<-test"), "test-aea-test"; slug != expected {
		t.Fatal("Return string is not slugified as expected", expected, slug)
	}
}

func TestLowerOption(t *testing.T) {
	if slug, expected := CreateSlug("Test->àèâ<-Test"), "test-aea-test"; slug != expected {
		t.Error("Return string is not slugified as expected", expected, slug)
	}
}
