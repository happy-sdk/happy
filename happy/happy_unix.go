// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

//go:build (linux && !android) || freebsd || openbsd

package happy

func osmain(ch chan struct{}) {
	if ch != nil {
		<-ch
	} else {
		select {}
	}
}
