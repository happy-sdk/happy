// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2020 The Happy Authors

//go:build !windows && !plan9
// +build !windows,!plan9

package bexp_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/happy-sdk/happy/pkg/strings/bexp"
)

func ExampleMkdirAll() {
	const (
		rootdir = "/tmp/bexp"
		treeExp = rootdir + "/{dir1,dir2,dir3/{subdir1,subdir2}}"
	)
	if err := bexp.MkdirAll(treeExp, 0750); err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(rootdir)

	if err := bexp.MkdirAll(rootdir+"/path/unmodified", 0750); err != nil {
		log.Println(err)
		return
	}

	err := filepath.Walk(rootdir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		})
	if err != nil {
		log.Println(err)
		return
	}

	// Output:
	// /tmp/bexp
	// /tmp/bexp/dir1
	// /tmp/bexp/dir2
	// /tmp/bexp/dir3
	// /tmp/bexp/dir3/subdir1
	// /tmp/bexp/dir3/subdir2
	// /tmp/bexp/path
	// /tmp/bexp/path/unmodified
}
