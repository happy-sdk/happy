// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors
package cron

import (
	"log"
	"strings"
	"testing"
	"time"

	"github.com/happy-sdk/happy-go/devel/testutils"
)

func TestWithLocation(t *testing.T) {
	c := New(WithLocation(time.UTC))
	if c.location != time.UTC {
		t.Errorf("expected UTC, got %v", c.location)
	}
}

func TestWithParser(t *testing.T) {
	var parser = NewParser(Dow)
	c := New(WithParser(parser))
	if c.parser != parser {
		t.Error("expected provided parser")
	}
}

func TestWithVerboseLogger(t *testing.T) {
	var buf syncWriter
	var logger = log.New(&buf, "", log.LstdFlags)
	c := New(WithLogger(VerbosePrintfLogger(logger)))
	if c.logger.(printfLogger).logger != logger {
		t.Error("expected provided logger")
	}

	if _, err := c.AddFunc("@every 1s", func() {}); err != nil {
		testutils.NoError(t, err)
	}
	c.Start()
	time.Sleep(OneSecond)
	c.Stop()
	out := buf.String()
	if !strings.Contains(out, "schedule,") ||
		!strings.Contains(out, "run,") {
		t.Error("expected to see some actions, got:", out)
	}
}
