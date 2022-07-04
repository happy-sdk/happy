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

//go:build linux && !android && !freebsd && !openbsd && !darwin

package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/mkungla/happy"
)

func appmain() {
	select {}
}

// bash completion handler.
func bashcompletion(cmds map[string]happy.Command, name string) {
	args := os.Args
StartAgain:
	cmdpre := args[len(args)-1]
	if cmdpre == name && len(args) >= 2 {
		args = args[:len(args)-1]

		goto StartAgain
	}

	var cmd happy.Command
	for name, c := range cmds {
		if strings.HasPrefix(name, cmdpre) {
			cmd = c
		}
	}
	if cmd == nil {
		return
	}

	// args = args[:len(args)-1]
	fmt.Fprint(os.Stdout, cmd.String())
}
