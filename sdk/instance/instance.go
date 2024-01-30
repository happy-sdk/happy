// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package instance

import (
	"errors"
	"sync"

	"github.com/happy-sdk/happy/sdk/networking/address"
	"github.com/happy-sdk/happy/sdk/settings"
)

type Settings struct {
	Max settings.Uint `key:"max" default:"1" mutation:"once"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type Instance struct {
	mu   sync.RWMutex
	addr *address.Address
}

func New(slug string) (*Instance, error) {
	curr, err := address.Current()
	if err != nil {
		return nil, err
	}
	if len(slug) == 0 {
		return nil, errors.New("instance can not be created without slug")
	}
	a, err := curr.Parse(slug)
	if err != nil {
		return nil, err
	}
	return &Instance{
		addr: a,
	}, nil
}

func (i *Instance) Address() *address.Address {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.addr
}
