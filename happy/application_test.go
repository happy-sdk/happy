// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"testing"

	"github.com/happy-sdk/happy-go/testutils"
)

func TestAppDefaultInstance(t *testing.T) {
	app := New(Settings{})
	if app == nil {
		t.Fatal("application is nil")
		return
	}
	testutils.False(t, app.running)
}
