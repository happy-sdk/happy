// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package humanize

import (
	"fmt"
	"strings"
	"time"
)

const (
	now                 = "now"
	lessThanMillisecond = "less than a millisecond"
)

// Duration formats a time.Duration into a human-readable string.
// It supports both full and short formats.
func Duration(d time.Duration, short bool) string {
	if d < time.Millisecond {
		if short {
			return now
		}
		return lessThanMillisecond
	}

	// Use a string builder for more efficient string concatenation
	var sb strings.Builder

	// Pre-calculate capacity to avoid reallocations
	// Estimate ~15 chars per time unit (e.g., "10 minutes ")
	sb.Grow(75)

	// Extract components
	days := d / (24 * time.Hour)
	d %= 24 * time.Hour
	hours := d / time.Hour
	d %= time.Hour
	minutes := d / time.Minute
	d %= time.Minute
	seconds := d / time.Second
	milliseconds := (d % time.Second) / time.Millisecond

	// Use constants for unit names
	var (
		dayUnit, hourUnit, minuteUnit, secondUnit, msUnit string
	)

	if short {
		dayUnit, hourUnit, minuteUnit, secondUnit, msUnit = "d", "h", "m", "s", "ms"
	} else {
		dayUnit, hourUnit, minuteUnit, secondUnit, msUnit = " day", " hour", " minute", " second", " millisecond"
	}

	// Track if we've added any parts
	addedPart := false

	// Helper function to append a time part
	appendPart := func(value int64, unit string) {
		if value <= 0 {
			return
		}

		if addedPart {
			sb.WriteString(" ")
		}

		fmt.Fprintf(&sb, "%d%s", value, unit)

		// Add plural suffix for non-short format and values != 1
		if !short && value != 1 {
			sb.WriteString("s")
		}

		addedPart = true
	}

	// Build parts based on non-zero values
	appendPart(int64(days), dayUnit)
	appendPart(int64(hours), hourUnit)
	appendPart(int64(minutes), minuteUnit)

	// Only include seconds if less than a day
	if days == 0 {
		appendPart(int64(seconds), secondUnit)
	}

	// Only include milliseconds if less than a minute
	if days == 0 && hours == 0 && minutes == 0 {
		appendPart(int64(milliseconds), msUnit)
	}

	return sb.String()
}
