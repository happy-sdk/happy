// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"testing"

	"github.com/mkungla/happy/sdk/testutils"
)

func TestAppDefaultInstance(t *testing.T) {
	app := New()
	if app == nil {
		t.Fatal("application is nil")
		return
	}
	testutils.False(t, app.running)
}
