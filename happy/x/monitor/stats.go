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

package monitor

import (
	"time"
)

type Stats struct {
	started time.Time
	stopped time.Time
	evs     int
}

func (s *Stats) Started() time.Time {
	return s.started
}

func (s *Stats) Elapsed() time.Duration {
	if s.stopped.IsZero() {
		return time.Since(s.started)
	}
	return s.stopped.Sub(s.started)
}

func (s *Stats) TotalEvents() int {
	return s.evs
}
