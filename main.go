// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package main

import (
	"fmt"

	"github.com/happy-sdk/happy"
)

func main() {
	app := happy.New(happy.Settings{
		Name:           "Happy-Go - Development tools",
		Description:    "Development tools for Happy-Go intended for maintainers",
		License:        "Apache License 2.0",
		CopyrightSince: 2020,
	})

	app.Do(func(sess *happy.Session, args happy.Args) error {
		fmt.Println("Happy-Go - Development tools")
		return nil
	})

	app.Main()
}
