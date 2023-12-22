// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

type Preferences struct {
	consumed bool
	data     map[string]string
}

func NewPreferences() *Preferences {
	p := &Preferences{
		data: make(map[string]string),
	}
	return p
}

func (p *Preferences) Consume() {
	p.consumed = true
}

func (p *Preferences) Set(key, val string) {
	p.data[key] = val
}
