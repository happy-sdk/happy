// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

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
