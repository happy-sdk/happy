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

	Disabled bool
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
	lastDomBit = 1 << 61
	lastDowBit = 1 << 62
	starBit    = 1 << 63
)

// Next returns the next time this schedule is activated, greater than the given
// time.  If no time can be found to satisfy the schedule, return the zero time.
func (s *ScheduleSpec) Next(t time.Time) time.Time {
	if s.Disabled {
		return time.Time{}
	}

	// Convert to schedule's timezone, preserving original timezone for return.
	origLocation := t.Location()
	loc := s.Location
	if loc == time.Local {
		loc = t.Location()
	}
	if s.Location != time.Local {
		t = t.In(s.Location)
	}

	// Start at the next second.
	t = t.Add(1*time.Second - time.Duration(t.Nanosecond())*time.Nanosecond)

	// If no time is found within five years, return zero.
	yearLimit := t.Year() + 5

	// Track if a field has been incremented.
	added := false

WRAP:
	if t.Year() > yearLimit {
		return time.Time{}
	}

	// Handle last-day-of-month case.
	if s.Dom&lastDomBit != 0 {
		year, month, day := t.Date()
		// Calculate last day of the current month.
		lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
		// If current day is past the last day or after midnight on the last day, move to next month's last day.
		if day > lastDay || (day == lastDay && (t.Hour() > 0 || t.Minute() > 0 || t.Second() > 0)) {
			t = time.Date(year, month+1, 1, 0, 0, 0, 0, loc)
			month = t.Month()
			lastDay = time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
			added = true
		}
		// Set to midnight on the last day.
		t = time.Date(year, month, lastDay, 0, 0, 0, 0, loc)
	}

	// Handle last-weekday case.
	if s.Dow&lastDowBit != 0 {
		year, month, day := t.Date()
		// Calculate last day of the current month.
		lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
		// Find the last workday (Monday-Friday) or specific weekday in the month.
		dowBits := s.Dow &^ lastDowBit            // Clear lastDowBit
		isWorkdays := dowBits == getBits(1, 5, 1) // Check if Monday-Friday
		lastDowDay := lastDay
		for {
			d := time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
			weekday := uint(d.Weekday())
			if isWorkdays {
				if weekday >= 1 && weekday <= 5 { // Monday to Friday
					break
				}
			} else {
				// Specific weekday (e.g., 5L for last Friday)
				for i := uint(0); i <= 6; i++ {
					if dowBits&(1<<i) != 0 && weekday == i {
						goto found
					}
				}
			}
			lastDowDay--
			continue
		found:
			break
		}
		// If current day is past the last Dow or after midnight on that day, move to next month.
		if day > lastDowDay || (day == lastDowDay && (t.Hour() > 0 || t.Minute() > 0 || t.Second() > 0)) {
			t = time.Date(year, month+1, 1, 0, 0, 0, 0, loc)
			month = t.Month()
			lastDay = time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
			lastDowDay = lastDay
			for {
				d := time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
				weekday := uint(d.Weekday())
				if isWorkdays {
					if weekday >= 1 && weekday <= 5 {
						break
					}
				} else {
					for i := uint(0); i <= 6; i++ {
						if dowBits&(1<<i) != 0 && weekday == i {
							goto foundNext
						}
					}
				}
				lastDowDay--
				continue
			foundNext:
				break
			}
			added = true
		}
		// Set to midnight on the last Dow.
		t = time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
	}

	// Handle first-weekday case (e.g., @firstweekday).
	isFirstWeekday := s.Dom == getBits(1, 7, 1) && s.Dow == (1<<1) // Monday in days 1–7
	if isFirstWeekday {
		year, month, day := t.Date()
		// Find the first Monday in the current month.
		firstDowDay := findFirstMonday(year, month, loc)
		// If current day is past the first Monday or after midnight on that day, move to next month.
		if day > firstDowDay || (day == firstDowDay && (t.Hour() > 0 || t.Minute() > 0 || t.Second() > 0)) {
			t = time.Date(year, month+1, 1, 0, 0, 0, 0, loc)
			month = t.Month()
			firstDowDay = findFirstMonday(year, month, loc)
			added = true
		}
		// Set to midnight on the first Monday.
		t = time.Date(year, month, firstDowDay, 0, 0, 0, 0, loc)
	}

	// Find the first applicable month.
	for 1<<uint(t.Month())&s.Month == 0 {
		if !added {
			added = true
			t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
		}
		t = t.AddDate(0, 1, 0)
		if t.Month() == time.January {
			if s.Dom&lastDomBit != 0 {
				// Recalculate last day for new month.
				year, month, _ := t.Date()
				lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
				t = time.Date(year, month, lastDay, 0, 0, 0, 0, loc)
			} else if s.Dow&lastDowBit != 0 {
				// Recalculate last Dow for new month.
				year, month, _ := t.Date()
				lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
				dowBits := s.Dow &^ lastDowBit
				isWorkdays := dowBits == getBits(1, 5, 1)
				lastDowDay := lastDay
				for {
					d := time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
					weekday := uint(d.Weekday())
					if isWorkdays {
						if weekday >= 1 && weekday <= 5 {
							break
						}
					} else {
						for i := uint(0); i <= 6; i++ {
							if dowBits&(1<<i) != 0 && weekday == i {
								goto foundWrap
							}
						}
					}
					lastDowDay--
					continue
				foundWrap:
					break
				}
				t = time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
			} else if isFirstWeekday {
				// Recalculate first Monday for new month.
				year, month, _ := t.Date()
				firstDowDay := findFirstMonday(year, month, loc)
				t = time.Date(year, month, firstDowDay, 0, 0, 0, 0, loc)
			}
			goto WRAP
		}
	}

	// Handle non-last-day/non-last-Dow/non-first-weekday day matching.
	if s.Dom&lastDomBit == 0 && s.Dow&lastDowBit == 0 && !isFirstWeekday {
		for !dayMatches(s, t) {
			if !added {
				added = true
				t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
			}
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, loc)
			if t.Hour() != 0 { // Handle DST transitions.
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
	}

	// Find matching hour.
	for 1<<uint(t.Hour())&s.Hour == 0 {
		if !added {
			added = true
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, loc)
		}
		t = t.Add(1 * time.Hour)
		if t.Hour() == 0 {
			if s.Dom&lastDomBit != 0 {
				// Recalculate last day for next day.
				year, month, _ := t.Date()
				lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
				t = time.Date(year, month, lastDay, 0, 0, 0, 0, loc)
			} else if s.Dow&lastDowBit != 0 {
				// Recalculate last Dow for new month.
				year, month, _ := t.Date()
				lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
				dowBits := s.Dow &^ lastDowBit
				isWorkdays := dowBits == getBits(1, 5, 1)
				lastDowDay := lastDay
				for {
					d := time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
					weekday := uint(d.Weekday())
					if isWorkdays {
						if weekday >= 1 && weekday <= 5 {
							break
						}
					} else {
						for i := uint(0); i <= 6; i++ {
							if dowBits&(1<<i) != 0 && weekday == i {
								goto foundHour
							}
						}
					}
					lastDowDay--
					continue
				foundHour:
					break
				}
				t = time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
			} else if isFirstWeekday {
				// Recalculate first Monday for new month.
				year, month, _ := t.Date()
				firstDowDay := findFirstMonday(year, month, loc)
				t = time.Date(year, month, firstDowDay, 0, 0, 0, 0, loc)
			}
			goto WRAP
		}
	}

	// Find matching minute.
	for 1<<uint(t.Minute())&s.Minute == 0 {
		if !added {
			added = true
			t = t.Truncate(time.Minute)
		}
		t = t.Add(1 * time.Minute)
		if t.Minute() == 0 {
			if s.Dom&lastDomBit != 0 {
				// Recalculate last day for next day.
				year, month, _ := t.Date()
				lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
				t = time.Date(year, month, lastDay, 0, 0, 0, 0, loc)
			} else if s.Dow&lastDowBit != 0 {
				// Recalculate last Dow for new month.
				year, month, _ := t.Date()
				lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
				dowBits := s.Dow &^ lastDowBit
				isWorkdays := dowBits == getBits(1, 5, 1)
				lastDowDay := lastDay
				for {
					d := time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
					weekday := uint(d.Weekday())
					if isWorkdays {
						if weekday >= 1 && weekday <= 5 {
							break
						}
					} else {
						for i := uint(0); i <= 6; i++ {
							if dowBits&(1<<i) != 0 && weekday == i {
								goto foundMinute
							}
						}
					}
					lastDowDay--
					continue
				foundMinute:
					break
				}
				t = time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
			} else if isFirstWeekday {
				// Recalculate first Monday for new month.
				year, month, _ := t.Date()
				firstDowDay := findFirstMonday(year, month, loc)
				t = time.Date(year, month, firstDowDay, 0, 0, 0, 0, loc)
			}
			goto WRAP
		}
	}

	// Find matching second.
	for 1<<uint(t.Second())&s.Second == 0 {
		if !added {
			added = true
			t = t.Truncate(time.Second)
		}
		t = t.Add(1 * time.Second)
		if t.Second() == 0 {
			if s.Dom&lastDomBit != 0 {
				// Recalculate last day for new month.
				year, month, _ := t.Date()
				lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
				t = time.Date(year, month, lastDay, 0, 0, 0, 0, loc)
			} else if s.Dow&lastDowBit != 0 {
				// Recalculate last Dow for new month.
				year, month, _ := t.Date()
				lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
				dowBits := s.Dow &^ lastDowBit
				isWorkdays := dowBits == getBits(1, 5, 1)
				lastDowDay := lastDay
				for {
					d := time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
					weekday := uint(d.Weekday())
					if isWorkdays {
						if weekday >= 1 && weekday <= 5 {
							break
						}
					} else {
						for i := uint(0); i <= 6; i++ {
							if dowBits&(1<<i) != 0 && weekday == i {
								goto foundSecond
							}
						}
					}
					lastDowDay--
					continue
				foundSecond:
					break
				}
				t = time.Date(year, month, lastDowDay, 0, 0, 0, 0, loc)
			} else if isFirstWeekday {
				// Recalculate first Monday for new month.
				year, month, _ := t.Date()
				firstDowDay := findFirstMonday(year, month, loc)
				t = time.Date(year, month, firstDowDay, 0, 0, 0, 0, loc)
			}
			goto WRAP
		}
	}

	return t.In(origLocation)
}

// Interval returns the shortest duration between consecutive times the schedule would trigger.
// If the schedule is irregular (e.g., specific days of the month or weekdays), it returns the
// smallest possible interval based on the finest granularity specified.
func (s *ScheduleSpec) Interval() time.Duration {
	if s.Disabled {
		return -1
	}
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

func (s *ScheduleSpec) IsDisabled() bool {
	return s.Disabled
}

func findFirstMonday(year int, month time.Month, loc *time.Location) int {
	firstDowDay := 1
	for {
		d := time.Date(year, month, firstDowDay, 0, 0, 0, 0, loc)
		if d.Weekday() == time.Monday {
			return firstDowDay
		}
		firstDowDay++
		if firstDowDay > 7 {
			break // Should not happen with valid Dom
		}
	}
	return firstDowDay
}

// dayMatches returns true if the schedule's day-of-week and day-of-month
// restrictions are satisfied by the given time.
//
//	func dayMatches(s *ScheduleSpec, t time.Time) bool {
//		return (s.Dom&(1<<uint(t.Day())) != 0 || s.Dom&starBit != 0) &&
//			(s.Dow&(1<<uint(t.Weekday())) != 0 || s.Dow&starBit != 0)
//	}
func dayMatches(s *ScheduleSpec, t time.Time) bool {
	domMatch := 1<<uint(t.Day())&s.Dom > 0
	dowMatch := 1<<uint(t.Weekday())&s.Dow > 0
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

func annual(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,
		Minute:   1 << minutes.min,
		Hour:     1 << hours.min,
		Dom:      1 << dom.min,
		Month:    1 << months.min,
		Dow:      all(dow),
		Location: loc,
	}
}

func quarterly(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,
		Minute:   1 << minutes.min,
		Hour:     1 << hours.min,
		Dom:      1 << dom.min,
		Month:    (1 << 1) | (1 << 4) | (1 << 7) | (1 << 10), // Jan, Apr, Jul, Oct
		Dow:      all(dow),
		Location: loc,
	}
}

func monthly(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,
		Minute:   1 << minutes.min,
		Hour:     1 << hours.min,
		Dom:      1 << dom.min,
		Month:    all(months),
		Dow:      all(dow),
		Location: loc,
	}
}

func lastday(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,
		Minute:   1 << minutes.min,
		Hour:     1 << hours.min,
		Dom:      getBits(28, 31, 1) | lastDomBit, // Possible last days, flagged as 'L'
		Month:    all(months),
		Dow:      all(dow),
		Location: loc,
	}
}

func lastweekday(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,
		Minute:   1 << minutes.min,
		Hour:     1 << hours.min,
		Dom:      all(dom),
		Month:    all(months),
		Dow:      (1 << 1) | lastDowBit, // Last Monday
		Location: loc,
	}
}

func firstweekday(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,
		Minute:   1 << minutes.min,
		Hour:     1 << hours.min,
		Dom:      getBits(1, 7, 1), // First 7 days of the month
		Month:    all(months),
		Dow:      1 << 1, // Monday
		Location: loc,
	}
}

func weekly(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,
		Minute:   1 << minutes.min,
		Hour:     1 << hours.min,
		Dom:      all(dom),
		Month:    all(months),
		Dow:      1 << dow.min,
		Location: loc,
	}
}

func weekdays(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min, // 0 seconds
		Minute:   1 << minutes.min, // 0 minutes
		Hour:     1 << hours.min,   // 0 hours (midnight)
		Dom:      all(dom),         // Any day of month
		Month:    all(months),      // Any month
		Dow:      getBits(1, 5, 1), // Monday (1) to Friday (5)
		Location: loc,
	}
}

func weekends(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,    // 0 seconds
		Minute:   1 << minutes.min,    // 0 minutes
		Hour:     1 << hours.min,      // 0 hours (midnight)
		Dom:      all(dom),            // Any day of month
		Month:    all(months),         // Any month
		Dow:      (1 << 0) | (1 << 6), // Sunday (0) and Saturday (6)
		Location: loc,
	}
}

func midnight(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,
		Minute:   1 << minutes.min,
		Hour:     1 << hours.min,
		Dom:      all(dom),
		Month:    all(months),
		Dow:      all(dow),
		Location: loc,
	}
}

func noon(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,
		Minute:   1 << minutes.min,
		Hour:     1 << 12, // 12:00
		Dom:      all(dom),
		Month:    all(months),
		Dow:      all(dow),
		Location: loc,
	}
}

func hourly(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{
		Second:   1 << seconds.min,
		Minute:   1 << minutes.min,
		Hour:     all(hours),
		Dom:      all(dom),
		Month:    all(months),
		Dow:      all(dow),
		Location: loc,
	}
}
