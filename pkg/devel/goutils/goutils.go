// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package goutils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy/pkg/version"
	"golang.org/x/mod/modfile"
)

func ContainsGoModfile(dir string) (string, bool) {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return filepath.Join(dir, "go.mod"), true
	}
	return "", false
}

func DependsOnHappy(dir string) (ver version.Version, yes bool, err error) {
	gomod, ok := ContainsGoModfile(dir)
	if !ok {
		return
	}

	var (
		data    []byte
		modFile *modfile.File
	)

	data, err = os.ReadFile(gomod)
	if err != nil {
		return
	}

	modFile, err = modfile.Parse(gomod, data, nil)
	if err != nil {
		return
	}

	if modFile.Module != nil {
		if modFile.Module.Mod.Path == "github.com/happy-sdk/happy" {
			yes = true
			ver = version.OfDir(dir)
			return
		}
	}

	// Check all require statements for github.com/happy-sdk/happy
	for _, req := range modFile.Require {
		if req.Mod.Path == "github.com/happy-sdk/happy" {
			ver, err = version.Parse(req.Mod.Version)
			if err != nil {
				return
			}
			yes = true
			return
		}
	}

	return
}

// IsGoRun detects if the program is running via 'go run'
func IsGoRun() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}

	return strings.Contains(executable, os.TempDir()) ||
		strings.Contains(executable, "go-build")
}
