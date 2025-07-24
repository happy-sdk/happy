// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"

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

func (p *Preferences) SchemaVersion() version.Version {
	return p.version
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
	if versionStr == "" {
		p.version = version.Version("v1.0.0")
	} else {
		ver, err := version.Parse(versionStr)
		if err != nil {
			return err
		}
		p.version = ver
	}

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

func (p *Preferences) UnmarshalJSON(b []byte) error {
	prefs := make(map[string]json.RawMessage)
	if err := json.Unmarshal(b, &prefs); err != nil {
		return err
	}

	versionStr, ok := prefs["version"]
	if !ok {
		return ErrPreferences
	}

	var ver version.Version
	if err := json.Unmarshal(versionStr, &ver); err != nil {
		return err
	}
	p.version = ver
	p.data = make(map[string]string)

	for rootKey, rootRawValue := range prefs {
		if rootKey == "version" {
			continue
		}

		var rootValueAny any
		if err := json.Unmarshal(rootRawValue, &rootValueAny); err != nil {
			return err
		}

		switch rootValue := rootValueAny.(type) {
		case map[string]any:
			for key, val := range rootValue {
				subKey := fmt.Sprintf("%s.%s", rootKey, key)
				switch subValue := val.(type) {
				case map[string]any:
					data := parseNested(subKey, subValue)
					maps.Copy(p.data, data)
				case []string:
					val, err := vars.NewValue(subValue)
					if err != nil {
						return err
					}
					p.data[subKey] = val.String()
				default:
					val, err := vars.NewValue(subValue)
					if err != nil {
						return err
					}
					p.data[subKey] = val.String()
				}

			}
		default:
			val, err := vars.NewValue(rootValue)
			if err != nil {
				return err
			}
			p.data[rootKey] = val.String()
		}
	}

	return nil
}

func parseNested(key string, obj map[string]any) map[string]string {
	data := make(map[string]string)
	for k, v := range obj {
		subKey := fmt.Sprintf("%s.%s", key, k)
		switch subValue := v.(type) {
		case map[string]any:
			sdata := parseNested(subKey, subValue)
			for k, v := range sdata {
				nestedKey := fmt.Sprintf("%s.%s", subKey, k)
				data[nestedKey] = v
			}
		case []string:
			val, err := vars.NewValue(subValue)
			if err != nil {
				return nil
			}
			data[subKey] = val.String()
		default:
			if strslice, ok := subValue.([]string); ok {
				val, err := vars.NewValue(strslice)
				if err != nil {
					return nil
				}
				data[subKey] = val.String()
			} else {
				val, err := vars.NewValue(subValue)
				if err != nil {
					return nil
				}
				data[subKey] = val.String()
			}
		}
	}
	return data
}
