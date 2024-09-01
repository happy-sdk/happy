// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package datetime

import (
	"github.com/happy-sdk/happy/pkg/settings"
	"golang.org/x/text/language"
)

type Settings struct {
	Location settings.String `key:"location,config" default:"Local" mutation:"once" desc:"The location to use for time operations."`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}
	en := language.English
	b.Describe("location", en, "The location to use for time operations.")
	return b, nil
}
