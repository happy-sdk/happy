// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 The Happy Authors

package internal

import (
	"log"
	"path/filepath"
)

var TestData = func() string {
	testdata, err := filepath.Abs("../../testdata")
	if err != nil {
		log.Fatal(err)
	}
	return testdata
}
