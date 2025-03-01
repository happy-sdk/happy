// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package humanize

import (
	"fmt"
	"strings"
	"time"
)

// Duration formats a time.Duration into a human-readable string
func Duration(d time.Duration) string {
	if d == 0 {
		return "now"
	}

	var parts []string

	// Extract components
	days := int64(d / (24 * time.Hour))
	d %= 24 * time.Hour
	hours := int64(d / time.Hour)
	d %= time.Hour
	minutes := int64(d / time.Minute)
	d %= time.Minute
	seconds := int64(d / time.Second)
	milliseconds := int64(d%time.Second) / int64(time.Millisecond)

	// Build parts based on non-zero values
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d day%s", days, plural(days)))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d hour%s", hours, plural(hours)))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d minute%s", minutes, plural(minutes)))
	}
	if seconds > 0 && days == 0 {
		parts = append(parts, fmt.Sprintf("%d second%s", seconds, plural(seconds)))
	}
	// Only include milliseconds if the duration is small (less than a minute)
	if milliseconds > 0 && days == 0 && hours == 0 && minutes == 0 {
		parts = append(parts, fmt.Sprintf("%d millisecond%s", milliseconds, plural(milliseconds)))
	}

	if len(parts) == 0 {
		return "less than a millisecond"
	}

	return strings.Join(parts, " ")
}

// plural returns "s" for plural if n != 1, otherwise an empty string.
func plural(n int64) string {
	if n == 1 {
		return ""
	}
	return "s"
}
