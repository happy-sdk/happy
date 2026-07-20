// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package changelog

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestRenderUnsupportedFormat(t *testing.T) {
	_, err := fixtureDocument().Render(Format("yaml"))
	testutils.Error(t, err, "expected an error for an unsupported format")
}
