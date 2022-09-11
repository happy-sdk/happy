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

package restapi

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/sdk"
)

type RestAPIAddon struct {
	*sdk.AddonSkeleton
}

func New(options ...happy.OptionWriteFunc) *RestAPIAddon {
	return &RestAPIAddon{
		AddonSkeleton: sdk.NewAddonSkeleton("restapi", "0.0.1"),
	}
}

func (a *RestAPIAddon) Cronjobs() {}

func (a *RestAPIAddon) Options() happy.OptionDefaultsSetter {
	return nil
}
func (a *RestAPIAddon) Commands() []happy.Command {
	return nil
}
func (a *RestAPIAddon) Services() []happy.Service {
	return nil
}
