// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package main

import (
	"github.com/happy-sdk/happy"
)

func main() {
	app := happy.New(happy.Settings{
		Name:           "Happy-SDK - Happy Prototyping Framework and SDK",
		Slug:           "happy-sdk",
		Description:    "Happy is a powerful tool for developers looking to bring their ideas to life through rapid prototyping.",
		License:        "Apache License 2.0",
		CopyrightBy:    "The Happy Authors",
		CopyrightSince: 2020,
	})

	app.Before(func(sess *happy.Session, args happy.Args) error {
		return nil
	})
	app.Do(func(sess *happy.Session, args happy.Args) error {
		return nil
	})
	app.Main()
}
