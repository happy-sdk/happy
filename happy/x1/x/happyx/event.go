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

package happyx

import (
	"github.com/mkungla/happy"
	"time"
)

type Event struct {
	scope, key string
	ts         time.Time
	err        happy.Error
	payload    happy.Variables
}

func NewEvent(scope, key string, payload happy.Variables, err error) Event {
	var e happy.Error
	if err != nil {
		e = ErrEvent.Wrap(err)
	}
	return Event{
		scope:   scope,
		key:     key,
		payload: payload,
		err:     e,
		ts:      time.Now(),
	}
}

func (ev Event) Key() string {
	return ev.key
}

func (ev Event) Scope() string {
	return ev.scope
}

func (ev Event) Time() time.Time {
	return ev.ts
}

func (ev Event) Payload() happy.Variables {
	return ev.payload
}

func (ev Event) Err() happy.Error {
	return ev.err
}
