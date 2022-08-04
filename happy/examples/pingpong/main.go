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

import (
	"github.com/mkungla/happy"
	"github.com/mkungla/happy/examples/pingpong/addons/pingpong"
	"github.com/mkungla/happy/sdk/create"
)

// go run . start
func main() {
	app := create.App(
		happy.Title("Ping Pong Example APP"),
		happy.Slug("happy-example-pingpong"),
	)

	// register pinpong addon
	app.RegisterAddons(pingpong.New)

	app.Run()
}
