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

// Package cli is provides implementations of happy.Application
// command line interfaces
package cli

import (
	"github.com/mkungla/happy/x/happyx"
)

var (
	ErrCommand        = happyx.NewError("command error")
	ErrCommandAction  = happyx.NewError("command action error")
	ErrCommandInvalid = happyx.NewError("invalid command definition")
	ErrCommandArgs    = happyx.NewError("command arguments error")
	ErrCommandFlags   = happyx.NewError("command flags error")
	ErrPanic          = happyx.NewError("there was panic, check logs for more info")
)
