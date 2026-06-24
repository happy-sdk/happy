// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package humanize

import (
	"math"
	"testing"
	"time"
)

func TestDuration(t *testing.T) {
	tests := []struct {
		name  string
		d     time.Duration
		short bool
		want  string
	}{
		{"zero", 0, false, "less than a millisecond"},
		{"zero short", 0, true, "now"},
		{"sub-millisecond", 500 * time.Microsecond, false, "less than a millisecond"},
		{"one millisecond", time.Millisecond, false, "1 millisecond"},
		{"one millisecond short", time.Millisecond, true, "1ms"},
		{"plural milliseconds", 5 * time.Millisecond, false, "5 milliseconds"},
		{"one second", time.Second, false, "1 second"},
		{"plural seconds", 30 * time.Second, false, "30 seconds"},
		{"one minute", time.Minute, false, "1 minute"},
		{"minute and seconds", 90 * time.Second, false, "1 minute 30 seconds"},
		{"one hour", time.Hour, false, "1 hour"},
		{"hour and minutes", 90 * time.Minute, false, "1 hour 30 minutes"},
		{"hour minutes short", 90 * time.Minute, true, "1h 30m"},
		{"one day", 24 * time.Hour, false, "1 day"},
		{"day and hours", 25 * time.Hour, false, "1 day 1 hour"},
		{"days hours minutes", 25*time.Hour + 30*time.Minute, false, "1 day 1 hour 30 minutes"},
		{"days drop seconds/ms", 25*time.Hour + 30*time.Second, false, "1 day 1 hour"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Duration(test.d, test.short)
			if got != test.want {
				t.Errorf("Duration(%v, %v) = %q, want %q", test.d, test.short, got, test.want)
			}
		})
	}
}

// TestDurationNegative is a regression test: negative durations were
// unconditionally caught by the `d < time.Millisecond` check (true for any
// negative value) and silently collapsed to "less than a millisecond"/
// "now", discarding the actual magnitude entirely.
func TestDurationNegative(t *testing.T) {
	tests := []struct {
		name  string
		d     time.Duration
		short bool
		want  string
	}{
		{"negative minutes", -90 * time.Minute, false, "-1 hour 30 minutes"},
		{"negative minutes short", -90 * time.Minute, true, "-1h 30m"},
		{"negative sub-millisecond", -500 * time.Microsecond, false, "-less than a millisecond"},
		{"negative one second", -time.Second, false, "-1 second"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Duration(test.d, test.short)
			if got != test.want {
				t.Errorf("Duration(%v, %v) = %q, want %q", test.d, test.short, got, test.want)
			}
		})
	}
}

// TestDurationMinInt64DoesNotHang is a regression test: negating
// math.MinInt64 overflows back to math.MinInt64 itself, which would
// recurse forever in a naive `-d` negation without an explicit guard.
func TestDurationMinInt64DoesNotHang(t *testing.T) {
	done := make(chan string, 1)
	go func() {
		done <- Duration(time.Duration(math.MinInt64), false)
	}()
	select {
	case got := <-done:
		if got == "" {
			t.Error("expected a non-empty result")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Duration(math.MinInt64, false) did not return: infinite recursion")
	}
}
