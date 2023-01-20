// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mkungla/happy/sdk/testutils"
)

type ApplicationTestSuite struct {
	App      *Application
	exitCode int
	output   []string
}

func Suite(code int, output []string, opts ...OptionArg) *ApplicationTestSuite {
	return &ApplicationTestSuite{
		App:      New(opts...),
		output:   output,
		exitCode: code,
	}
}

func (s *ApplicationTestSuite) Test(t *testing.T) {
	// Capture the output of the application when it is run
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// Run the application
	s.App.exitOs = false
	s.App.exitFunc = append(s.App.exitFunc, func(code int) {
		testutils.Equal(t, s.exitCode, code)
	})
	s.App.Main()

	// Restore the original stdout
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read the output of the application from the pipe
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := strings.Split(strings.TrimSpace(buf.String()), "\n")
	testutils.EqualAny(t, s.output, output, "invalid output")
}

func TestHappyDefault(t *testing.T) {
	suite := Suite(1, []string{""})
	suite.Test(t)
}
