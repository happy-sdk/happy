// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package releaser

import (
	"sync"

	"github.com/happy-sdk/happy/sdk/addon"
	"github.com/happy-sdk/happy/sdk/app/session"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/custom"
	"github.com/happy-sdk/happy/sdk/internal/cmd/hsdk/addons/releaser/module"
)

func Addon() *addon.Addon {
	addon := addon.New(addon.Config{
		Name: "Releaser",
	})

	r := newReleaser()

	addon.ProvideCommand(r.createReleaseCommand())

	return addon
}

type releaser struct {
	custom.API
	mu       sync.RWMutex
	sess     *session.Context
	packages []*module.Package
	queue    []string
}

func newReleaser() *releaser {
	return &releaser{}
}

func (r *releaser) createReleaseCommand() *command.Command {
	r.mu.Lock()
	defer r.mu.Unlock()

	command.New(command.Config{
		Name: "release",
	})
	return nil
}
