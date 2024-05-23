// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

// Package cli provides utilities for happy command line interfaces.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/happy-sdk/happy/pkg/settings"
)

var (
	ErrCommandInvalid = errors.New("invalid command definition")
	ErrCommandArgs    = errors.New("command arguments error")
	ErrCommandFlags   = errors.New("command flags error")
	ErrPanic          = errors.New("there was panic, check logs for more info")
)

type Settings struct {
	MainArgcMax settings.Uint `key:"argn_max" default:"0"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// AskForConfirmation gets (y/Y)es or (n/N)o from cli input.
func AskForConfirmation(q string) bool {
	var response string
	fmt.Fprintln(os.Stdout, q, "(y/Y)es or (n/N)o?")

	if _, err := fmt.Scanln(&response); err != nil {
		return false
	}

	switch strings.ToLower(response) {
	case "y", "Y", "yes":
		return true
	case "n", "N", "no":
		return false
	default:
		fmt.Fprintln(
			os.Stdout,
			"I'm sorry but I didn't get what you meant, please type (y/Y)es or (n/N)o and then press enter:")

		return AskForConfirmation(q)
	}
}

func AskForInput(q string) string {
	var response string
	fmt.Fprintln(os.Stdout, q)
	if _, err := fmt.Scanln(&response); err != nil {
		return ""
	}
	return strings.TrimSpace(response)
}
