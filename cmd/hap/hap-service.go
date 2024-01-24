// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package main

import (
	"time"

	"github.com/happy-sdk/happy"
)

func service() *happy.Service {
	svc := happy.NewService("background")
	svc.OnTick(func(sess *happy.Session, ts time.Time, delta time.Duration) error {
		sess.Log().Info("backgroundService.OnTick")
		return nil
	})
	svc.OnTock(func(sess *happy.Session, delta time.Duration, tps int) error {
		sess.Log().Info("backgroundService.OnTock")
		return nil
	})
	return svc
}
