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

package service

import (
	"fmt"
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/pkg/vars"
	"time"
)

type Event struct {
	key     string
	ts      time.Time
	payload happy.Variables
}

func (ev Event) Key() string {
	return ev.key
}

func (ev Event) Scope() string {
	return "session"
}

func (ev Event) Err() happy.Error {
	return nil
}

func (ev Event) Payload() happy.Variables {
	return ev.payload
}

func (ev Event) Time() time.Time {
	return ev.ts
}

func NewRequireServicesEvent(urls ...happy.URL) happy.Event {
	svcs := vars.AsMap[happy.Variables, happy.Variable, happy.Value](new(vars.Map))
	for i, url := range urls {
		svcs.Store(fmt.Sprintf("service.%d", i), url)
	}
	return Event{
		key:     "require.services",
		payload: svcs,
		ts:      time.Now(),
	}
}
