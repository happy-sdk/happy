// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package datetime

import "github.com/happy-sdk/happy/pkg/settings"

type Settings struct {
	Location settings.String `key:"location,save" default:"Local" mutation:"mutable"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}
