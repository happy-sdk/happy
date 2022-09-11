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

package main

// import (
// 	"fmt"

// 	"github.com/mkungla/happy"
// 	"github.com/mkungla/happy/x/sdk/application"
// )

// func main() {
// 	// Create application instance
// 	// With this minimal example we do not provide Configurator
// 	// and expect application implementation auto configure it for us.
// 	app, err := application.New(nil)

// 	// Apply configuration to application
// 	if err != nil {
// 		app.Log().Error(err)
// 		return
// 	}

// 	// Minimal example what is available by default
// 	app.Do(func(ctx happy.Session, args happy.Variables, assets happy.FS) error {

// 		fmt.Println("SESSION")
// 		ctx.RangeOptions(func(key string, value happy.Value) bool {
// 			fmt.Println("key: ", key, " value: ", value)
// 			return true
// 		})

// 		fmt.Println("SETTINGS")
// 		ctx.Settings().RangeOptions(func(key string, value happy.Value) bool {
// 			fmt.Println("key: ", key, " value: ", value)
// 			return true
// 		})

// 		return nil
// 	})

// 	app.Main()
// }
