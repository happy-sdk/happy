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

// Package x provides an experimental API. These Happy subpackages are part
// of a experimental API and are not guaranteed to make it to future releases.
package peeraddr

import (
	"bytes"
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/happyx"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"unicode"
)

const (
	// MustCompile against following expression.
	MustCompile = "^[a-z][a-z0-9-./]*[a-z0-9]$"
	dot         = '.'
)

var alnum = &unicode.RangeTable{ //nolint:gochecknoglobals
	R16: []unicode.Range16{
		{'0', '9', 1},
		{'A', 'Z', 1},
		{'a', 'z', 1},
	},
}

// Create creates app name from given name
// e.g. github.com/digaverse/howi
// will become
// com.gihub.digaverse.howi

func Create(modulepath string) happy.URL {
	// fully qualified ?

	sl := strings.Split(modulepath, "/")
	if len(sl) == 1 {
		u, _ := happyx.ParseURL("happy://" + ensure(modulepath))
		return u
	}

	var rev []string
	var rmdomain bool
	if strings.Contains(sl[0], ".") {
		rmdomain = true
		domainparts := sort.StringSlice(strings.Split(sl[0], "."))
		sort.Sort(domainparts)
		rev = append(rev, ensure(strings.Join(domainparts, ".")))
	}
	p := len(sl)
	for i := 0; i < p; i++ {
		if rmdomain && i == 0 {
			continue
		}
		rev = append(rev, ensure(sl[i]))
	}
	u, _ := happyx.ParseURL("happy://" + strings.Join(rev, "."))
	return u
}

// Current returns MustCompile format of current application
// all non alpha numeric characters removed.
func Current() happy.URL {
	var name string
	if info, available := debug.ReadBuildInfo(); available {
		name = info.Main.Path
	} else {
		pc, _, _, _ := runtime.Caller(0)
		ps := strings.Split(runtime.FuncForPC(pc).Name(), ".")
		pl := len(ps)
		if ps[pl-2][0] == '(' {
			name = strings.Join(ps[0:pl-2], ".")
		} else {
			name = strings.Join(ps[0:pl-1], ".")
		}
	}
	return Create(name)
}

// Valid returns true if s is string which is valid app name.
func Valid(s string) bool {
	re := regexp.MustCompile(MustCompile)
	return re.MatchString(s)
}

func ensure(in string) string {
	var b bytes.Buffer
	for _, c := range in {
		isAlnum := unicode.Is(alnum, c)
		isSpace := unicode.IsSpace(c)
		isLower := unicode.IsLower(c)
		if isSpace || (!isAlnum && c != dot) {
			continue
		}
		if !isLower {
			c = unicode.ToLower(c)
		}
		b.WriteRune(c)
	}
	return b.String()
}
