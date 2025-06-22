// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package service

import (
	"sync"
	"time"

	"github.com/happy-sdk/happy/pkg/networking/address"
)

type Info struct {
	mu        sync.RWMutex
	name      string
	addr      *address.Address
	running   bool
	errs      map[time.Time]error
	startedAt time.Time
	stoppedAt time.Time
}

func NewInfo(name string, addr *address.Address) *Info {
	return &Info{
		name: name,
		addr: addr,
	}
}

func (s *Info) Valid() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.name != "" && s.addr != nil
}

func (s *Info) Running() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Info) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.name
}

func (s *Info) StartedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startedAt
}

func (s *Info) StoppedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stoppedAt
}

func (s *Info) Addr() *address.Address {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}

func (s *Info) Failed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.errs) > 0
}

func (s *Info) Errs() map[time.Time]error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.errs == nil {
		return nil
	}
	errsCopy := make(map[time.Time]error, len(s.errs))
	for k, v := range s.errs {
		errsCopy[k] = v
	}
	return errsCopy
}

func (s *Info) started() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = true
	s.startedAt = time.Now().UTC()
}

func (s *Info) stopped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
	s.stoppedAt = time.Now().UTC()
}

func (s *Info) addErr(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.errs == nil {
		s.errs = make(map[time.Time]error)
	}
	s.errs[time.Now().UTC()] = err
}

func AddError(s *Info, err error) {
	if s == nil {
		return
	}
	s.addErr(err)
}

func MarkStarted(s *Info) {
	if s == nil {
		return
	}
	s.started()
}

func MarkStopped(s *Info) {
	if s == nil {
		return
	}
	s.stopped()
}
