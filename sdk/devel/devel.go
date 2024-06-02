// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package devel

import (
	"fmt"
	"runtime"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
)

var (
	FlagXProd = varflag.BoolFunc("x-prod", false, "DEV ONLY: force app into production mode setting app_is_devel false when running from source.")
)

// Settings for the devel module.
// These settings are used to configure the behavior of the application when user
// compiles your application from source or uses go run .
type Settings struct {
	AllowProd settings.Bool `default:"false" desc:"Allow set app into production mode when running from source."`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// RuntimeCallerStr is utility function to get the caller information.
// It returns the file and line number of the caller in form of string.
// e.g. /path/to/file.go:123
func RuntimeCallerStr(depth int) (string, bool) {
	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s:%d", file, line), true
}
