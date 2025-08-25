// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors
package cron

import "time"

// ScheduleSpec specifies a duty cycle (to the second granularity), based on a
// traditional crontab specification. It is computed initially and stored as bit sets.
type ScheduleSpec struct {
	Second, Minute, Hour, Dom, Month, Dow uint64

	// Override location for this schedule.
	Location *time.Location

	disabled bool
}

// bounds provides a range of acceptable values (plus a map of name to value).
type bounds struct {
	min, max uint
	names    map[string]uint
}

// The bounds for each field.
var (
	seconds = bounds{0, 59, nil}
	minutes = bounds{0, 59, nil}
	hours   = bounds{0, 23, nil}
	dom     = bounds{1, 31, nil}
	months  = bounds{1, 12, map[string]uint{
		"jan": 1,
		"feb": 2,
		"mar": 3,
		"apr": 4,
		"may": 5,
		"jun": 6,
		"jul": 7,
		"aug": 8,
		"sep": 9,
		"oct": 10,
		"nov": 11,
		"dec": 12,
	}}
	dow = bounds{0, 6, map[string]uint{
		"sun": 0,
		"mon": 1,
		"tue": 2,
		"wed": 3,
		"thu": 4,
		"fri": 5,
		"sat": 6,
	}}
)

const (
	// Set the top bit if a star was included in the expression.
	starBit = 1 << 63
)

// Next returns the next time this schedule is activated, greater than the given
// time.  If no time can be found to satisfy the schedule, return the zero time.
func (s *ScheduleSpec) Next(t time.Time) time.Time {
	if s.disabled {
		return time.Time{}
	}
	// General approach
	//
	// For Month, Day, Hour, Minute, Second:
	// Check if the time value matches.  If yes, continue to the next field.
	// If the field doesn't match the schedule, then increment the field until it matches.
	// While incrementing the field, a wrap-around brings it back to the beginning
	// of the field list (since it is necessary to re-verify previous field
	// values)

	// Convert the given time into the schedule's timezone, if one is specified.
	// Save the original timezone so we can convert back after we find a time.
	// Note that schedules without a time zone specified (time.Local) are treated
	// as local to the time provided.
	origLocation := t.Location()
	loc := s.Location
	if loc == time.Local {
		loc = t.Location()
	}
	if s.Location != time.Local {
		t = t.In(s.Location)
	}

	// Start at the earliest possible time (the upcoming second).
	t = t.Add(1*time.Second - time.Duration(t.Nanosecond())*time.Nanosecond)

	// This flag indicates whether a field has been incremented.
	added := false

	// If no time is found within five years, return zero.
	yearLimit := t.Year() + 5

WRAP:
	if t.Year() > yearLimit {
		return time.Time{}
	}

	// Find the first applicable month.
	// If it's this month, then do nothing.
	for 1<<uint(t.Month())&s.Month == 0 {
		// If we have to add a month, reset the other parts to 0.
		if !added {
			added = true
			// Otherwise, set the date at the beginning (since the current time is irrelevant).
			t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
		}
		t = t.AddDate(0, 1, 0)

		// Wrapped around.
		if t.Month() == time.January {
			goto WRAP
		}
	}

	// Now get a day in that month.
	//
	// NOTE: This causes issues for daylight savings regimes where midnight does
	// not exist.  For example: Sao Paulo has DST that transforms midnight on
	// 11/3 into 1am. Handle that by noticing when the Hour ends up != 0.
	for !dayMatches(s, t) {
		if !added {
			added = true
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		}
		t = t.AddDate(0, 0, 1)
		// Notice if the hour is no longer midnight due to DST.
		// Add an hour if it's 23, subtract an hour if it's 1.
		if t.Hour() != 0 {
			if t.Hour() > 12 {
				t = t.Add(time.Duration(24-t.Hour()) * time.Hour)
			} else {
				t = t.Add(time.Duration(-t.Hour()) * time.Hour)
			}
		}

		if t.Day() == 1 {
			goto WRAP
		}
	}

	for 1<<uint(t.Hour())&s.Hour == 0 {
		if !added {
			added = true
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, loc)
		}
		t = t.Add(1 * time.Hour)

		if t.Hour() == 0 {
			goto WRAP
		}
	}

	for 1<<uint(t.Minute())&s.Minute == 0 {
		if !added {
			added = true
			t = t.Truncate(time.Minute)
		}
		t = t.Add(1 * time.Minute)

		if t.Minute() == 0 {
			goto WRAP
		}
	}

	for 1<<uint(t.Second())&s.Second == 0 {
		if !added {
			added = true
			t = t.Truncate(time.Second)
		}
		t = t.Add(1 * time.Second)

		if t.Second() == 0 {
			goto WRAP
		}
	}

	return t.In(origLocation)
}

// Interval returns the shortest duration between consecutive times the schedule would trigger.
// If the schedule is irregular (e.g., specific days of the month or weekdays), it returns the
// smallest possible interval based on the finest granularity specified.
func (s *ScheduleSpec) Interval() time.Duration {
	// Helper function to count set bits in a uint64
	countSetBits := func(n uint64) int {
		count := 0
		for n != 0 {
			count += int(n & 1)
			n >>= 1
		}
		return count
	}

	// Check if the starBit is set (indicating a wildcard)
	hasStar := func(field uint64) bool {
		return field&starBit != 0
	}

	// If seconds are specified (not all set or not a wildcard), return 1s as the finest granularity
	if !hasStar(s.Second) && countSetBits(s.Second) < 60 {
		return time.Second
	}

	// If minutes are specified, return 1m
	if !hasStar(s.Minute) && countSetBits(s.Minute) < 60 {
		return time.Minute
	}

	// If hours are specified, return 1h
	if !hasStar(s.Hour) && countSetBits(s.Hour) < 24 {
		return time.Hour
	}

	// If days of month or days of week are specified, return 24h
	if (!hasStar(s.Dom) && countSetBits(s.Dom) < 31) || (!hasStar(s.Dow) && countSetBits(s.Dow) < 7) {
		return 24 * time.Hour
	}

	// If months are specified, return a nominal 1-month duration (30 days for approximation)
	if !hasStar(s.Month) && countSetBits(s.Month) < 12 {
		return 30 * 24 * time.Hour
	}

	// Default case: if all fields are wildcards or fully set, assume the finest granularity (1s)
	return time.Second
}

func (s *ScheduleSpec) Disabled() bool {
	return s.disabled
}

// dayMatches returns true if the schedule's day-of-week and day-of-month
// restrictions are satisfied by the given time.
func dayMatches(s *ScheduleSpec, t time.Time) bool {
	var (
		domMatch = 1<<uint(t.Day())&s.Dom > 0
		dowMatch = 1<<uint(t.Weekday())&s.Dow > 0
	)
	if s.Dom&starBit > 0 || s.Dow&starBit > 0 {
		return domMatch && dowMatch
	}
	return domMatch || dowMatch
}

type Expression string

// MarshalSetting converts the Expression setting to a byte slice for storage or transmission.
func (e *Expression) MarshalSetting() ([]byte, error) {
	// Simply cast the String to a byte slice.
	return []byte(*e), nil
}

// UnmarshalSetting updates the Expression setting from a byte slice, typically read from storage or received in a message.
func (e *Expression) UnmarshalSetting(data []byte) error {
	*e = Expression(data)
	return nil
}

// String returns Expression string
func (e Expression) String() string {
	return string(e)
}
