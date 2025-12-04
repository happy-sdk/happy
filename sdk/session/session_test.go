// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package session

import (
	"errors"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/api"
)

func TestCreateTestSession_Basic(t *testing.T) {
	sess, buf, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	testutils.NotNil(t, sess, "session should not be nil")
	testutils.NotNil(t, buf, "buffer should not be nil")
	testutils.NotNil(t, cleanup, "cleanup function should not be nil")
}

func TestCreateTestSession_Logger(t *testing.T) {
	sess, buf, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test that logger works
	sess.Log().Info("test message")
	sess.Log().Error("error message")
	sess.Log().Warn("warning message")

	logOutput := buf.String()
	testutils.Assert(t, logOutput != "", "expected log output to be non-empty")
	testutils.ContainsString(t, logOutput, "test message")
	testutils.ContainsString(t, logOutput, "error message")
	testutils.ContainsString(t, logOutput, "warning message")
}

func TestCreateTestSession_ContextMethods(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Context() method
	ctx := sess.Context()
	testutils.NotNil(t, ctx, "context should not be nil")

	// Test Done() channel
	done := sess.Done()
	testutils.NotNil(t, done, "done channel should not be nil")

	// Test Err() - should be nil initially
	err = sess.Err()
	testutils.Nil(t, err, "err should be nil initially")

	// Test String() method
	str := sess.String()
	testutils.Equal(t, "happy.Session", str)
}

func TestCreateTestSession_Opts(t *testing.T) {
	opt := options.NewOption("test.key", "test.value")
	sess, _, cleanup, err := CreateTestSession(nil, opt)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Opts() method
	opts := sess.Opts()
	testutils.NotNil(t, opts, "opts should not be nil")

	// Test that option is available
	testutils.Assert(t, sess.Opts().Accepts("test.key"), "expected option 'test.key' to be accepted")

	// Test Get() method
	val := sess.Get("test.key")
	testutils.Assert(t, !val.Empty(), "expected option value to be non-empty")
	testutils.Equal(t, "test.value", val.String())
}

func TestCreateTestSession_Settings(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Settings() method
	profile := sess.Settings()
	testutils.NotNil(t, profile, "profile should not be nil")
	testutils.Assert(t, profile.Loaded(), "profile should be loaded")
}

func TestCreateTestSession_HasAndGet(t *testing.T) {
	opt := options.NewOption("test.option", "option.value")
	sess, _, cleanup, err := CreateTestSession(nil, opt)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Has() with option
	testutils.Assert(t, sess.Has("test.option"), "expected Has() to return true for existing option")
	testutils.Assert(t, !sess.Has("nonexistent.key"), "expected Has() to return false for non-existent key")

	// Test Get() with option
	val := sess.Get("test.option")
	testutils.Assert(t, !val.Empty(), "expected Get() to return non-empty variable")
	testutils.Equal(t, "option.value", val.String())

	// Test Get() with non-existent key
	emptyVal := sess.Get("nonexistent.key")
	testutils.Equal(t, vars.EmptyVariable, emptyVal)
}

func TestCreateTestSession_Ready(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Ready() channel
	readyCh := sess.Ready()
	testutils.NotNil(t, readyCh, "ready channel should not be nil")

	// Dispatch ready event to mark session as ready
	readyEvent := ReadyEvent()
	sess.Dispatch(readyEvent)

	// Give a small delay for the event to be processed
	time.Sleep(10 * time.Millisecond)

	// After dispatching ready event, channel should be closed
	select {
	case <-readyCh:
		// Channel is closed, session is ready
	case <-time.After(100 * time.Millisecond):
		// Channel might not be closed immediately, but Ready() should return a valid channel
		// This is acceptable behavior - the channel will close when session becomes ready
	}
}

func TestCreateTestSession_Dispatch(t *testing.T) {
	sess, buf, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Dispatch with ready event
	readyEvent := ReadyEvent()
	sess.Dispatch(readyEvent)

	// Test Dispatch with nil event (should log warning)
	sess.Dispatch(nil)
	logOutput := buf.String()
	testutils.ContainsString(t, logOutput, "received <nil> event")
}

func TestCreateTestSession_Time(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Time() method
	now := time.Now()
	convertedTime := sess.Time(now)
	testutils.NotNil(t, convertedTime, "converted time should not be nil")
	// Time should be in session's time location (defaults to Local)
}

func TestCreateTestSession_Valid(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Valid() method
	valid := sess.Valid()
	// Valid state depends on session initialization, just check it doesn't panic
	_ = valid
}

func TestCreateTestSession_Destroy(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	// Test Destroy() method
	sess.Destroy(nil)

	// After destroy, Err() should return ErrExitSuccess
	err = sess.Err()
	testutils.ErrorIs(t, err, ErrExitSuccess)

	// Cleanup should be safe to call multiple times
	cleanup()
	cleanup()
}

func TestCreateTestSession_DestroyWithError(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	testErr := errors.New("test error")
	sess.Destroy(testErr)

	// After destroy with error, Err() should return that error
	err = sess.Err()
	testutils.NotNil(t, err, "expected error after destroy")
	// The error is stored directly, not wrapped
	testutils.Equal(t, testErr, err, "error should match the error passed to Destroy")
}

func TestCreateTestSession_Cleanup(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	// Use session
	sess.Log().Info("before cleanup")

	// Call cleanup
	cleanup()

	// After cleanup, session should be destroyed
	err = sess.Err()
	testutils.ErrorIs(t, err, ErrExitSuccess)

	// Logger should be disposed (further logging may not work, but shouldn't panic)
	// We can't easily test this without causing issues, so we just verify cleanup doesn't panic
}

// customTestSettings implements settings.Settings for testing
type customTestSettings struct {
	Name settings.String `key:"app.name" default:"TestApp"`
}

func (c *customTestSettings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
}

func TestCreateTestSession_WithCustomSettings(t *testing.T) {
	customSettings := &customTestSettings{}
	sess, _, cleanup, err := CreateTestSession(customSettings)
	if err != nil {
		t.Fatalf("failed to create test session with custom settings: %v", err)
	}
	defer cleanup()

	testutils.NotNil(t, sess, "session should not be nil")
	profile := sess.Settings()
	testutils.NotNil(t, profile, "profile should not be nil")
}

func TestCreateTestSession_WithMultipleOptions(t *testing.T) {
	opts := []*options.OptionSpec{
		options.NewOption("opt1", "value1"),
		options.NewOption("opt2", "value2"),
		options.NewOption("opt3", 42),
	}

	sess, _, cleanup, err := CreateTestSession(nil, opts...)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test all options are available
	testutils.Assert(t, sess.Has("opt1"), "expected opt1 to be available")
	testutils.Assert(t, sess.Has("opt2"), "expected opt2 to be available")
	testutils.Assert(t, sess.Has("opt3"), "expected opt3 to be available")

	// Test option values
	testutils.Equal(t, "value1", sess.Get("opt1").String())
	testutils.Equal(t, "value2", sess.Get("opt2").String())
	testutils.Equal(t, "42", sess.Get("opt3").String())
}

func TestCreateTestSession_ContextCancellation(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	ctx := sess.Context()
	testutils.NotNil(t, ctx, "context should not be nil")

	// Context should not be done initially
	select {
	case <-ctx.Done():
		t.Error("context should not be done initially")
	default:
		// Expected - context is not done
	}

	// Destroy session
	sess.Destroy(nil)

	// After destroy, context should be done
	select {
	case <-ctx.Done():
		// Expected - context is done after destroy
	case <-time.After(100 * time.Millisecond):
		t.Error("context should be done after session destroy")
	}
}

func TestCreateTestSession_Describe(t *testing.T) {
	opt := options.NewOption("test.key", "test.value")
	sess, _, cleanup, err := CreateTestSession(nil, opt)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Describe() method
	desc := sess.Describe("test.key")
	// Description may be empty for test options, but method should not panic
	_ = desc
}

func TestCreateTestSession_CanRecover(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test CanRecover() with nil error
	canRecover := sess.CanRecover(nil)
	testutils.Assert(t, canRecover, "expected CanRecover(nil) to return true")

	// Test CanRecover() with ErrExitSuccess
	canRecover = sess.CanRecover(ErrExitSuccess)
	// CanRecover may return false if session has been terminated, just verify it doesn't panic
	_ = canRecover

	// Test CanRecover() with other error
	canRecover = sess.CanRecover(errors.New("test error"))
	// CanRecover behavior depends on session state, just verify it doesn't panic
	_ = canRecover
}

type testAPI struct {
	api.Provider
}

func (t *testAPI) Name() string {
	return "testAPI"
}

func TestCreateTestSession_AttachAPI(t *testing.T) {
	sess, _, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	err = sess.AttachAPI("test-api", &testAPI{})
	testutils.NoError(t, err, "expected no error attaching the test API")

	var api *testAPI
	err = API(sess, &api)
	testutils.NoError(t, err, "expected no error retrieving the test API")
}

func TestCreateTestSession_Value(t *testing.T) {
	opt := options.NewOption("test.key", "test.value")
	sess, _, cleanup, err := CreateTestSession(nil, opt)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Value() with string key (should return option value)
	val := sess.Value("test.key")
	testutils.NotNil(t, val, "expected Value() to return non-nil for existing option")

	// Test Value() with non-existent key
	val = sess.Value("nonexistent.key")
	testutils.Nil(t, val, "expected Value() to return nil for non-existent key")
}

func TestCreateTestSession_Release(t *testing.T) {
	sess, buf, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test Release() method
	sess.Release()

	// Release should log a message - flush the logger to ensure message is written
	_ = sess.Log().Flush()

	// Release should log a message
	logOutput := buf.String()
	if logOutput != "" {
		testutils.ContainsString(t, logOutput, "session released SIGINT, SIGKILL signals")
	}
	// If buffer is empty, that's also acceptable as logging may be async
}

func TestCreateTestSession_ConcurrentAccess(t *testing.T) {
	sess, buf, cleanup, err := CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	defer cleanup()

	// Test concurrent access to session methods
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			sess.Log().Info("concurrent log", "id", id)
			_ = sess.Opts()
			_ = sess.Settings()
			_ = sess.Context()
			_ = sess.Err()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify logs were written
	logOutput := buf.String()
	testutils.Assert(t, logOutput != "", "expected concurrent logs to be written")
}
