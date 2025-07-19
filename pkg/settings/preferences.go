// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"

	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/version"
)

type Preferences struct {
	version version.Version
	data    map[string]string
}

func NewPreferences(schemaVersion version.Version) *Preferences {
	p := &Preferences{
		data:    make(map[string]string),
		version: schemaVersion,
	}
	return p
}

func (p *Preferences) Set(key, value string) {
	p.data[key] = value
}
func (p *Preferences) GobDecode(data []byte) error {

	var (
		temp []string
	)

	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	if err := decoder.Decode(&temp); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: failed to decode preferences %s", ErrPreferences, err.Error())
	}

	prefsMap, err := vars.ParseMapFromSlice(temp)
	if err != nil {
		return err
	}

	p.data = make(map[string]string)

	versionStr := prefsMap.Get("version").String()
	ver, err := version.Parse(versionStr)
	if err != nil {
		return err
	}
	p.version = ver

	for v := range prefsMap.All() {
		if v.Name() == "version" {
			continue
		}
		p.data[v.Name()] = v.String()
	}

	return nil
}

func (p Preferences) GobEncode() ([]byte, error) {

	dataMap := vars.NewMap()
	for k, v := range p.data {
		if err := dataMap.Store(k, v); err != nil {
			return nil, err
		}
	}
	if err := dataMap.Store("version", p.version.String()); err != nil {
		return nil, err
	}

	data := dataMap.ToKeyValSlice()
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
