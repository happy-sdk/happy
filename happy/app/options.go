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
	"strings"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/config"
	"github.com/mkungla/happy/internal"
)

func (a *Application) applyOptions(options ...happy.Option) bool {
	conf := config.New()

	for _, opt := range options {
		var key internal.OptionKey

		a.handleInitErr(opt(&key))

		var opts happy.OptionSetter
		if strings.HasPrefix(string(key), "app.") {
			opts = &conf
			a.handleInitErr(applyOptionToOptions(opt, a.session))
		}
		if strings.HasPrefix(string(key), "settings.") {
			opts = a.session.Settings()
		}

		a.handleInitErr(applyOptionToOptions(opt, opts))
	}

	a.config = conf
	return a.errors.Len() == 0
}

func applyOptionToOptions(opt happy.Option, opts happy.OptionSetter) error {
	return opt(opts)
}
