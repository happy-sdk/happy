// Copyright 2020 Marko Kungla. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

// +build !windows,!plan9

package bexp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMkdirAllError(t *testing.T) {
	const (
		rootdir = ""
		treeexp = rootdir + "/{dir1,dir2,dir3/{subdir1,subdir2}}"
	)
	assert.Error(t, MkdirAll(treeexp, 0750), "creating dirs in root should error")
}
