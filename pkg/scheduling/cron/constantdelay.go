// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors
package cron

import "time"

// ConstantDelaySchedule represents a simple recurring duty cycle, e.g. "Every 5 minutes".
// It does not support jobs more frequent than once a second.
type ConstantDelaySchedule struct {
	Delay    time.Duration
	Disabled bool
}

// Every returns a crontab Schedule that activates once every duration.
// Delays of less than a second are not supported (will round up to 1 second).
// Any fields less than a Second are truncated.
func Every(duration time.Duration) ConstantDelaySchedule {
	if duration < time.Second {
		duration = time.Second
	}
	delay := duration - time.Duration(duration.Nanoseconds())%time.Second
	return ConstantDelaySchedule{
		Delay:    delay,
		Disabled: delay == 0,
	}
}

// Next returns the next time this should be run.
// This rounds so that the next activation time will be on the second.
func (recurring ConstantDelaySchedule) Next(t time.Time) time.Time {
	return t.Add(recurring.Delay - time.Duration(t.Nanosecond())*time.Nanosecond)
}

func (recurring ConstantDelaySchedule) Interval() time.Duration {
	return recurring.Delay
}

func (recurring ConstantDelaySchedule) IsDisabled() bool {
	return recurring.Disabled
}
