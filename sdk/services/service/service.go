// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package service

import (
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/slug"
	"github.com/happy-sdk/happy/sdk/events"
)

var (
	// StartedEvent triggered when service has been started
	StartedEvent = events.New("service", "started")
	// StoppedEvent triggered when service has been stopped
	StoppedEvent = events.New("service", "stopped")
)

type Settings struct {
	Name settings.String `key:",init" default:"Background" desc:"The name of the service."`
	// Slug is the unique identifier of the service, if not provided it will be generated from the name.
	Slug         settings.String   `key:",init" desc:"The slug of the service."`
	Description  settings.String   `key:",init" default:"xxx" desc:"The name of the service."`
	RetryOnError settings.Bool     `key:",init" default:"false" desc:"Retry the service in case of an error."`
	MaxRetries   settings.Int      `key:",init" default:"3" desc:"Maximum number of retries on error."`
	RetryBackoff settings.Duration `key:",init" default:"5s" desc:"Duration to wait before each retry."`
}

func (s *Settings) Blueprint() (*settings.Blueprint, error) {
	if s.Slug == "" && s.Name != "" {
		s.Slug = settings.String(slug.Create(s.Name.String()))
	}
	return settings.New(s)
}
