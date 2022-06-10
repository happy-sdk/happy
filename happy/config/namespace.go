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
	"bytes"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"unicode"
)

// Create creates app name from given name
// e.g. github.com/mkungla/happy
// will become
// com.gihub.mkungla.happy
func NamespaceFromModulePath(modulepath string) string {
	sl := strings.Split(modulepath, "/")
	if len(sl) == 1 {
		return ensureNamespacePart(modulepath)
	}
	var rev []string
	var rmdomain bool
	if strings.Contains(sl[0], ".") {
		rmdomain = true
		domainparts := sort.StringSlice(strings.Split(sl[0], "."))
		sort.Sort(domainparts)
		rev = append(rev, ensureNamespacePart(strings.Join(domainparts, ".")))
	}
	p := len(sl)
	for i := 0; i < p; i++ {
		if rmdomain && i == 0 {
			continue
		}
		rev = append(rev, ensureNamespacePart(sl[i]))
	}

	return strings.Join(rev, ".")
}

// NamespaceFromCurrentModule returns namespace of current module.
func NamespaceFromCurrentModule() string {
	var mpath string
	if info, available := debug.ReadBuildInfo(); available && len(info.Main.Path) > 0 {
		mpath = info.Main.Path
	} else {
		pc, _, _, _ := runtime.Caller(0)
		ps := strings.Split(runtime.FuncForPC(pc).Name(), ".")
		pl := len(ps)
		if ps[pl-2][0] == '(' {
			mpath = strings.Join(ps[0:pl-2], ".")
		} else {
			mpath = strings.Join(ps[0:pl-1], ".")
		}
	}
	return NamespaceFromModulePath(mpath)
}

// Valid returns true if s is string which is valid app namespace.
func ValidNamespace(s string) bool {
	re := regexp.MustCompile("^[a-z][a-z0-9-./]*[a-z0-9]$")
	return re.MatchString(s)
}

func ensureNamespacePart(in string) string {
	var b bytes.Buffer
	for _, c := range in {
		isAlnum := unicode.Is(alnum, c)
		isSpace := unicode.IsSpace(c)
		isLower := unicode.IsLower(c)
		if isSpace || (!isAlnum && c != '.') {
			continue
		}
		if !isLower {
			c = unicode.ToLower(c)
		}
		b.WriteRune(c)
	}
	return b.String()
}
