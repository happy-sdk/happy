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

	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/testutils"
)

type ApplicationTestSuite struct {
	App         *Application
	exitCode    int
	expectedOut []string
	oldStdout   *os.File
	oldStderr   *os.File
	outWriter   io.WriteCloser
	outReader   io.ReadCloser
}

func Suite(code int, expected []string, b *settings.Blueprint) *ApplicationTestSuite {

	suite := &ApplicationTestSuite{
		expectedOut: expected,
		exitCode:    code,
		oldStdout:   os.Stdout,
		oldStderr:   os.Stderr,
	}

	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w
	suite.outReader = r
	suite.outWriter = w

	app := New(Settings{
		Logger: logging.Settings{
			Level:  logging.LevelError,
			Source: true,
		},
	})
	app.Do(func(sess *Session, args Args) error { return nil })
	suite.App = app
	return suite
}

func (s *ApplicationTestSuite) Test(t *testing.T) {
	// Capture the output of the application when it is run

	// Run the application
	s.App.exitTrap = true
	s.App.exitFunc = append(s.App.exitFunc, func(code int) error {
		testutils.Equal(t, s.exitCode, code)
		return nil
	})
	s.App.Main()

	// Restore the original stdout
	os.Stdout = s.oldStdout
	os.Stderr = s.oldStderr

	// Read the output of the application from the pipe
	var buf bytes.Buffer
	s.outWriter.Close()
	io.Copy(&buf, s.outReader)
	s.outReader.Close()

	output := strings.Split(strings.TrimSpace(buf.String()), "\n")
	testutils.EqualAny(t, s.expectedOut, output, "invalid output")
}

func TestHappyDefault(t *testing.T) {
	suite := Suite(0, []string{""}, nil)
	suite.Test(t)
}
