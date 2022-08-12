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

package application

import (
	"fmt"
	"github.com/mkungla/happy"
	"io/fs"
)

type Application struct {
}

func New(conf ...happy.Configurator) (happy.Application, happy.Error) {
	app := &Application{}
	return app, nil
}

func UseLogger(logger happy.Logger) {
	fmt.Println("Application.UseLogger")
}

func UseSession(ctx happy.Session) {
	fmt.Println("Application.UseSession")
}

func UseMonitor(mon happy.ApplicationMonitor) {
	fmt.Println("Application.UseMonitor")
}

func UseAssets(assets fs.FS) {
	fmt.Println("Application.UseAssets")
}

func UseEngine(assets fs.FS) {
	fmt.Println("Application.UseAssets")
}

func AddAddon(addon happy.Addon) {
	fmt.Println("Application.AddAddon")
}

func AddAddons(addons ...happy.AddonCreateFunc) {
	fmt.Println("Application.AddAddons")
}

func AddCommand(cmd happy.Command) {
	fmt.Println("Application.AddCommand")
}

func AddCommands(cmds ...happy.CommandCreateFunc) {
	fmt.Println("Application.AddCommands")
}

func AddFlag(flag happy.CommandFlag) {
	fmt.Println("Application.AddFlag")
}

func AddFlags(flags ...happy.CommandCreateFlagFunc) {
	fmt.Println("Application.AddFlags")
}

func Service(svc happy.Service) {
	fmt.Println("Application.Service")
}

func AddServices(svcs ...happy.ServiceCreateFunc) {
	fmt.Println("Application.AddServices")
}
