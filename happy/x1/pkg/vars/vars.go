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

// Package vars provides the API to parse variables from various input
// formats/kinds to common key value pair. Or key value pair sets to Map.
//
// Main purpose of this library is to provide simple API
// to pass variables between different domains and programming languaes.
//
// Originally based of https://github.com/mkungla/vars
// and will be moved there as new module import path some point.

package vars

var (
	ErrValueConv = wrapError(ErrValue, "failed to convert value")
	ErrKey       = newError("variable key error")
)
