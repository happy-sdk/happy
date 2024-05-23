// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package sdk

import "github.com/happy-sdk/happy/pkg/options"

func Option(key string, dval any, desc string, vfunc options.ValueValidator) options.OptionSpec {
	return options.NewOption(key, dval, desc, options.KindRuntime, vfunc)
}
