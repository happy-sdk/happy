// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package initializer_test

import (
	"testing"
	"time"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/app"
	"github.com/happy-sdk/happy/sdk/session"
)

func TestNil(t *testing.T) {
	log := logging.NewTestLogger(logging.LevelError)
	app := app.New(nil)
	app.WithLogger(log)
	testutils.NotNil(t, app, "app must never be nil")
	app.Run()

	testutils.Contains(t, log.Output(), "settings is <nil>")
}

func TestDefault(t *testing.T) {
	log := logging.NewTestLogger(logging.LevelError)
	app := app.New(&happy.Settings{})
	app.WithLogger(log)
	testutils.NotNil(t, app, "app must never be nil")

	var (
		beforeAlwaysCalled bool
		beforeCalled       bool
		doCalled           bool
		afterSuccessCalled bool
		afterAlwaysCalled  bool
	)

	app.BeforeAlways(func(sess *session.Context, args action.Args) error {
		testutils.Equal(t, "Happy Prototype", sess.Get("app.name").String(), "app.name")
		testutils.Equal(t, "com-github-happy-sdk-happy-sdk-app-internal-initializer-test-test", sess.Get("app.slug").String(), "app.slug")
		testutils.Equal(t, "com.github.happy-sdk.happy.sdk.app.internal.initializer.test-test", sess.Get("app.identifier").String(), "app.identifier")
		testutils.Equal(t, "This application is built using the Happy-SDK to provide enhanced functionality and features.", sess.Get("app.description").String(), "app.description")
		testutils.Equal(t, "Anonymous", sess.Get("app.copyright_by").String(), "app.copyright_by")
		testutils.Equal(t, time.Now().Year(), sess.Get("app.copyright_since").Int(), "app.year")
		testutils.Equal(t, "NOASSERTION", sess.Get("app.license").String(), "app.license")

		beforeAlwaysCalled = true
		return nil
	})

	app.Before(func(sess *session.Context, args action.Args) error {
		beforeCalled = true
		return nil
	})
	app.Do(func(sess *session.Context, args action.Args) error {
		doCalled = true
		return nil
	})
	app.AfterSuccess(func(sess *session.Context) error {
		afterSuccessCalled = true
		return nil
	})
	app.AfterAlways(func(sess *session.Context, err error) error {
		afterAlwaysCalled = true
		return nil
	})
	app.Run()

	testutils.Equal(t, "", log.Output())
	testutils.True(t, beforeAlwaysCalled, "app.BeforeAlways was not called to effectively test the default initializer.")
	testutils.True(t, beforeCalled, "app.Before was not called to effectively test the default initializer.")
	testutils.True(t, doCalled, "app.Do was not called to effectively test the default initializer.")
	testutils.True(t, afterSuccessCalled, "app.AfterSuccess was not called to effectively test the default initializer.")
	testutils.True(t, afterAlwaysCalled, "app.AfterAlways was not called to effectively test the default initializer.")
}

// func TestDefaultOptions(t *testing.T) {
// 	log := logging.NewTestLogger(logging.LevelError)
// 	app := app.New(&happy.Settings{})
// 	app.WithLogger(log)

// 	var (
// 		beforeAlwaysCalled bool
// 		doCalled           bool
// 	)
// 	app.BeforeAlways(func(sess *session.Context, args action.Args) error {
// 		testutils.Equal(t, 18, sess.Opts().Len(), "invalid default runtime options count")

// 		// app.address
// 		host, err := os.Hostname()
// 		if err != nil {
// 			return err
// 		}
// 		addr := fmt.Sprintf("happy://%s/com-github-happy-sdk-happy-sdk-app-internal-initializer-test-test", host)
// 		testutils.Equal(t, addr, sess.Get("app.address").String(), "app.address")

// 		tmpdir := filepath.Join(os.TempDir(), sess.Get("app.slug").String(), fmt.Sprintf("instance-%s", sess.Get("app.instance.id").String()))
// 		// app.fs.path.cache
// 		testutils.Equal(t, filepath.Join(tmpdir, "cache", "profiles", "default"), sess.Get("app.fs.path.cache").String(), "app.fs.path.cache")
// 		// app.fs.path.config
// 		testutils.Equal(t, filepath.Join(tmpdir, "config"), sess.Get("app.fs.path.config").String(), "app.fs.path.config")
// 		// app.fs.path.home
// 		home, err := os.UserHomeDir()
// 		if err != nil {
// 			return err
// 		}
// 		testutils.Equal(t, home, sess.Get("app.fs.path.home").String(), "app.fs.path.home")
// 		// app.fs.path.pids
// 		testutils.Equal(t, filepath.Join(tmpdir, "config", "pids"), sess.Get("app.fs.path.pids").String(), "app.fs.path.pids")
// 		// app.fs.path.profile
// 		testutils.Equal(t, filepath.Join(tmpdir, "config", "profiles", "default"), sess.Get("app.fs.path.profile").String(), "app.fs.path.profile")
// 		// app.fs.path.wd
// 		wd, err := os.Getwd()
// 		if err != nil {
// 			return err
// 		}
// 		testutils.Equal(t, wd, sess.Get("app.fs.path.wd").String(), "app.fs.path.wd")
// 		// app.fs.path.tmp
// 		testutils.Equal(t, tmpdir, sess.Get("app.fs.path.tmp").String(), "app.fs.path.tmp")

// 		// app.instance.id
// 		testutils.Equal(t, 8, sess.Get("app.instance.id").Len(), "app.instance.id length")
// 		// app.is_devel
// 		testutils.True(t, sess.Get("app.is_devel").Bool(), "app.is_devel")
// 		// app.main.exec.x
// 		testutils.False(t, sess.Get("app.main.exec.x").Bool(), "app.main.exec.x")
// 		// app.module
// 		testutils.Equal(t, "github.com/happy-sdk/happy/sdk/app/internal/initializer.test-test", sess.Get("app.module").String(), "app.module")
// 		// app.pid
// 		testutils.Equal(t, os.Getpid(), sess.Get("app.pid").Int(), "app.pid")
// 		// app.profile.name
// 		testutils.Equal(t, "default", sess.Get("app.profile.name").String(), "app.profile.name")
// 		// app.version
// 		// testutils.Equal(t, "v1.0.0-0xDEV", sess.Get("app.version").String(), "app.version")
// 		beforeAlwaysCalled = true
// 		return nil
// 	})
// 	app.Do(func(sess *session.Context, args action.Args) error {
// 		doCalled = true
// 		return nil
// 	})
// 	app.Run()
// 	testutils.True(t, beforeAlwaysCalled, "app.BeforeAlways was not called to effectively test the default initializer.")
// 	testutils.True(t, doCalled, "app.Do was not called to effectively test the default initializer.")
// }
