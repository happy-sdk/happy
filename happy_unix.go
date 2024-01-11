// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build (linux && !android) || freebsd || openbsd

package happy

func osmain(ch chan struct{}) {
	if ch != nil {
		<-ch
	} else {
		select {}
	}
}
