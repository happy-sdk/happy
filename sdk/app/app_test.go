// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package app_test

import (
	"testing"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/sdk/app"
)

func TestNew(t *testing.T) {
	log := logging.NewTestLogger(logging.LevelError)
	app := app.New(happy.Settings{})
	app.WithLogger(log)
	testutils.NotNil(t, app, "app must never be nil")
}
