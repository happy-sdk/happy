// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package devel

import (
	"sync"
)

type Project struct {
	mu sync.RWMutex
}
