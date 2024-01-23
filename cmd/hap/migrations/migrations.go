// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package migrations

import "github.com/happy-sdk/happy/sdk/migration"

func New() *migration.Manager {
	mm := migration.NewManager()
	return mm
}
