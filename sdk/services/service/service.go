// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package service

import (
	"fmt"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/events"
)

var (
	// StartedEvent triggered when service has been started
	StartedEvent = events.New("service", "started")
	// StoppedEvent triggered when service has been stopped
	StoppedEvent = events.New("service", "stopped")
)

type Settings struct {
	Name         settings.String   `key:",config" default:"background" desc:"The name of the service."`
	Description  settings.String   `key:",config" default:"xxx" desc:"The name of the service."`
	RetryOnError settings.Bool     `default:"false" desc:"Retry the service in case of an error."`
	MaxRetries   settings.Int      `default:"3" desc:"Maximum number of retries on error."`
	RetryBackoff settings.Duration `default:"1s" desc:"Duration to wait before each retry."`
}

func (s *Settings) Blueprint() (*settings.Blueprint, error) {
	bp, err := settings.New(s)
	if err != nil {
		return nil, err
	}
	fmt.Println("s.Name:", s.Name.String())
	fmt.Println("s.Description: ", s.Description.String())
	fmt.Printf("s.RetryOnError: %t\n", s.RetryOnError)
	fmt.Printf("s.MaxRetries: %d\n", s.MaxRetries)
	fmt.Printf("s.RetryBackoff: %s\n", s.RetryBackoff.String())

	return bp, nil
}
