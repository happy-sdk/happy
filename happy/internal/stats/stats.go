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

// Package stats adds minimal stats collector to your application..
package stats

import (
	"context"
	"sync"
	"time"
)

type Stats struct {
	// Updated is used to notify UI about changes in the timer.
	updated chan struct{}
	started time.Time // when application was started

	// mu locks the state such that it can be modified and accessed
	// from multiple goroutines.
	mu     sync.Mutex
	prev   time.Time // corresponds to the last updated time.
	frames int       // frames in current second
	fps    int       // fps in last second
	maxfps int       // max fps on that session
}

func New() *Stats {
	return &Stats{
		started: time.Now(),
		updated: make(chan struct{}),
	}
}

func (s *Stats) Dispose() {
	close(s.updated)
}

func (s *Stats) Elapsed() time.Duration {
	return time.Since(s.started)
}

// Start the timer goroutine and return a cancel func that
// that can be used to stop it.
func (s *Stats) Start() context.CancelFunc {
	// initialize the timer state.
	now := time.Now()
	time.Sleep(time.Duration(999999999 - now.Nanosecond()))
	s.prev = now

	// we use done to signal stopping the goroutine.
	// a context.Context could be also used.
	done := make(chan struct{})
	go s.run(done)
	return func() { close(done) }
}

func (s *Stats) Next() <-chan struct{} {
	return s.updated
}

func (s *Stats) Now() time.Time {
	return s.prev
}

func (s *Stats) FPS() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fps
}

func (s *Stats) MaxFPS() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxfps
}

func (s *Stats) Frame() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frames++
}

// run is the main loop for the timer.
func (s *Stats) run(done chan struct{}) {
	// we use a time.Ticker to update the state,
	tick := time.NewTicker(100 * time.Microsecond)
	defer tick.Stop()

	for {
		select {
		case now := <-tick.C:
			s.update(now)
		case <-done:
			return
		}
	}
}

func (s *Stats) update(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.prev.Second() != now.Second() {
		s.invalidate()
		s.fps = s.frames
		if s.fps > s.maxfps {
			s.maxfps = s.fps
		}
		s.frames = 0
	}
	s.prev = now
}

// invalidate sends a signal to the UI that
// the internal state has changed.
func (s *Stats) invalidate() {
	// we use a non-blocking send, that way the Timer
	// can continue updating internally.
	select {
	case s.updated <- struct{}{}:
	default:
	}
}
