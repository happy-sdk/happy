// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package happy

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

var ErrInvalidVersion = errors.New("invalid version")

type Version struct {
	v string
}

func ParseVersion(v string) (Version, error) {
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return Version{}, fmt.Errorf("%w: %s", ErrInvalidVersion, v)
	}
	return Version{v: v}, nil
}

func (v Version) String() string {
	return v.v
}

func (v Version) IsZero() bool {
	return len(v.v) == 0 || v.v == "v0.0.0"
}

func (v Version) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(v.v)+2)
	b = append(b, '"')
	b = append(b, []byte(v.v)...)
	b = append(b, '"')
	return b, nil
}

func (v *Version) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	v.v = strings.Trim(string(data), `"`)
	return nil
}
