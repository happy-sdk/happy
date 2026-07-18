// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import (
	"errors"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestErrCompletedWithFailuresWrapsError(t *testing.T) {
	testutils.Assert(t, errors.Is(ErrCompletedWithFailures, Error), "expected ErrCompletedWithFailures to wrap Error")
}

func TestStateOrdering(t *testing.T) {
	// executor.go and runner.go rely on state comparisons (res.state > state)
	// to pick the "worst" outcome across tasks/subtasks - this ordering must
	// hold for that logic to behave correctly. SKIPPED ranks above SUCCESS
	// (a skipped subtask is not a clean success) despite being the mildest
	// non-success outcome otherwise.
	states := []State{SUCCESS, SKIPPED, INFO, NOTICE, WARNING, FAILURE}
	for i := 1; i < len(states); i++ {
		testutils.Assert(t, states[i] > states[i-1], "expected %d > %d", states[i], states[i-1])
	}
}

func TestResultConstructors(t *testing.T) {
	tests := []struct {
		name      string
		result    Result
		wantState State
		wantMsg   string
	}{
		{"Success", Success("ok"), SUCCESS, "ok"},
		{"Info", Info("info"), INFO, "info"},
		{"Notice", Notice("notice"), NOTICE, "notice"},
		{"Warn", Warn("warn"), WARNING, "warn"},
		{"Failure", Failure("fail"), FAILURE, "fail"},
		{"Skip", Skip("skip"), SKIPPED, "skip"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutils.Equal(t, tt.wantState, tt.result.state, "unexpected state")
			testutils.Equal(t, tt.wantMsg, tt.result.msg, "unexpected msg")
		})
	}
}

func TestResultConstructorsNormalizeNewlines(t *testing.T) {
	r := Success("line1\nline2")
	testutils.Equal(t, "line1line2", r.msg, "expected newlines stripped from msg")
}

func TestResultWithDesc(t *testing.T) {
	r := Success("ok").WithDesc("some\ndesc")
	testutils.Equal(t, "somedesc", r.decription, "expected newlines stripped from description")
	testutils.Equal(t, SUCCESS, r.state, "WithDesc must not change state")
	testutils.Equal(t, "ok", r.msg, "WithDesc must not change msg")
}

func TestResultWithDescReturnsCopy(t *testing.T) {
	original := Success("ok")
	withDesc := original.WithDesc("desc")
	testutils.Equal(t, "", original.decription, "expected original Result to be unmodified (value receiver)")
	testutils.Equal(t, "desc", withDesc.decription, "expected new Result to carry the description")
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"no newlines", "no newlines"},
		{"a\nb\nc", "abc"},
		{"\n\n\n", ""},
		{"trailing\n", "trailing"},
	}
	for _, tt := range tests {
		testutils.Equal(t, tt.want, normalize(tt.in), "normalize(%q)", tt.in)
	}
}
