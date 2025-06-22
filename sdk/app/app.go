// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package app

import (
	"errors"

	_ "github.com/happy-sdk/happy/sdk/app/lang"
)

const i18np = "com.github.happy-sdk.happy.sdk.app"

var (
	Error = errors.New("app")
)
