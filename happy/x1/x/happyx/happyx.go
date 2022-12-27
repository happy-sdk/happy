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

package happyx

import (
	"net/url"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/pkg/vars"
)

type Slug string

func NewSlug(str string) (happy.Slug, error) {
	s, err := vars.ParseKey(str)
	if err != nil {
		return nil, err
	}
	return Slug(s), nil
}

func (s Slug) Valid() bool {
	_, err := vars.ParseKey(string(s))
	return err == nil
}

func (s Slug) String() string {
	return string(s)
}

// GetServiceAPI returns Service API from session.
func API[API happy.API](apis []happy.API) (api API, err happy.Error) {
	for _, a := range apis {
		if aa, ok := a.(API); ok {
			return aa, nil
		}
	}
	return api, BUG.WithTextf("requesting non existing API %T")
}

func AsAPI[API happy.API](api happy.API) (a API, err happy.Error) {
	if aa, ok := api.(API); ok {
		return aa, nil
	}
	return a, BUG.WithTextf("requesting non existing API %T")
}

type URL struct {
	raw      string
	peeraddr string
	args     happy.Variables
}

func (u *URL) Args() happy.Variables {
	return u.args
}

func (u *URL) PeerService() string {
	return u.peeraddr
}

func (u *URL) String() string {
	return u.raw
}

func ParseURL(str string) (*URL, error) {
	u, err := url.Parse(str)
	if err != nil {
		return nil, err
	}
	args := NewVariables()
	res := &URL{
		args:     args,
		raw:      str,
		peeraddr: u.Scheme + "://" + u.Host + u.Path,
	}

	for k, v := range u.Query() {
		if len(v) > 0 {
			res.args.Store(k, v[0])
		} else {
			// ?key (key = true)
			res.args.Store(k, true)
		}
	}

	return res, nil
}

func NewVariables() happy.Variables {
	return vars.AsMap[happy.Variables, happy.Variable, happy.Value](new(vars.Map))
}
