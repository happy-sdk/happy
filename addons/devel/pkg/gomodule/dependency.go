// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gomodule

import "github.com/happy-sdk/happy/pkg/version"

type Dependency struct {
	Import     string
	UsedBy     []string
	MaxVersion version.Version
	MinVersion version.Version
}
