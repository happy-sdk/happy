// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package daemon

import "errors"

func (s *Service) load() error {
	// load the daemon
	return errors.New("daemon not implemented on Windows")
}
