// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package releaser

import "github.com/happy-sdk/happy"

func Addon() *happy.Addon {
	addon := happy.NewAddon("releaser", nil)

	addon.ProvidesCommand(releaseCmd())
	return addon
}

func (api *API) releaseCmd() *happy.Command {
	cmd := happy.NewCommand("release")

	cmd.Do(func(sess *happy.Session, args happy.Args) error {
		return nil
	})
	return cmd
}
