// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package services

import "github.com/happy-sdk/happy/pkg/settings"

type Settings struct {
	LoaderTimeout settings.Duration `key:"loader_timeout,save" default:"30s" mutation:"once"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}
