// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package services

import (
	"testing"
	"time"

	"github.com/happy-sdk/happy/sdk/services/service"
	"github.com/happy-sdk/happy/sdk/session"
)

// TestBuilderMethodsRejectOverride is a regression test: OnRegister,
// OnStart, OnStop, Tick, Tock, and Cron all silently allowed overwriting a
// previously-set callback with no error, unlike the analogous
// command.Command.Do, which errors on double-registration. A second call
// to any of these now records a pending error (surfaced via
// Container.Register, the same mechanism used for the service's other
// initialization errors) instead of silently discarding the first
// callback.
func TestBuilderMethodsRejectOverride(t *testing.T) {
	t.Run("OnRegister", func(t *testing.T) {
		s := New(service.Config{})
		s.OnRegister(func(sess *session.Context) error { return nil })
		before := len(s.errs)
		s.OnRegister(func(sess *session.Context) error { return nil })
		if len(s.errs) <= before {
			t.Error("expected an error appended for overriding OnRegister")
		}
	})

	t.Run("OnStart", func(t *testing.T) {
		s := New(service.Config{})
		s.OnStart(func(sess *session.Context) error { return nil })
		before := len(s.errs)
		s.OnStart(func(sess *session.Context) error { return nil })
		if len(s.errs) <= before {
			t.Error("expected an error appended for overriding OnStart")
		}
	})

	t.Run("OnStop", func(t *testing.T) {
		s := New(service.Config{})
		s.OnStop(func(sess *session.Context, prevErr error) error { return nil })
		before := len(s.errs)
		s.OnStop(func(sess *session.Context, prevErr error) error { return nil })
		if len(s.errs) <= before {
			t.Error("expected an error appended for overriding OnStop")
		}
	})

	t.Run("Tick", func(t *testing.T) {
		s := New(service.Config{})
		s.Tick(func(sess *session.Context, ts time.Time, delta time.Duration) error { return nil })
		before := len(s.errs)
		s.Tick(func(sess *session.Context, ts time.Time, delta time.Duration) error { return nil })
		if len(s.errs) <= before {
			t.Error("expected an error appended for overriding Tick")
		}
	})

	t.Run("Tock", func(t *testing.T) {
		s := New(service.Config{})
		s.Tock(func(sess *session.Context, delta time.Duration, tps int) error { return nil })
		before := len(s.errs)
		s.Tock(func(sess *session.Context, delta time.Duration, tps int) error { return nil })
		if len(s.errs) <= before {
			t.Error("expected an error appended for overriding Tock")
		}
	})

	t.Run("Cron", func(t *testing.T) {
		s := New(service.Config{})
		s.Cron(func(schedule CronScheduler) {})
		before := len(s.errs)
		s.Cron(func(schedule CronScheduler) {})
		if len(s.errs) <= before {
			t.Error("expected an error appended for overriding Cron")
		}
	})
}

func TestBuilderMethodsAllowFirstSet(t *testing.T) {
	s := New(service.Config{})
	s.OnStart(func(sess *session.Context) error { return nil })
	if len(s.errs) != 0 {
		t.Errorf("expected no errors for a single OnStart call, got %v", s.errs)
	}
}
