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

package app

import (
	"fmt"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/mkungla/happy/version"
)

// loadVersion sets application version
//
// additional context
// https://github.com/golang/go/issues/37475
func (a *Application) loadModuleInfo() {
	var (
		ver string
		err error
	)
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		ver = fmt.Sprintf("0.0.0-devel+%d", time.Now().UnixMilli())
	} else {
		if bi.Main.Version == "(devel)" {
			moduleVersion := strings.Trim(filepath.Ext(a.config.Namespace), ".")
			major := "v1"
			if len(moduleVersion) > 0 {
				majorint, err := strconv.Atoi(strings.TrimPrefix(moduleVersion, "v"))
				if err == nil {
					major = fmt.Sprintf("v%d", majorint+1)
				}
				a.addAppErr(err)
			}
			ver = fmt.Sprintf("%s.0.0-alpha+%d", major, time.Now().UnixMilli())
		} else {
			ver = bi.Main.Version
		}

		a.session.Set("app.go.version", bi.GoVersion)
		a.session.Set("app.module.path", bi.Path)
		a.session.Set("app.module.sum", bi.Main.Sum)
		a.session.Set("app.module.version", bi.Main.Version)
	}

	a.version, err = version.Parse(ver)
	a.session.Set("app.version", a.version.String())
	a.addAppErr(err)
}
