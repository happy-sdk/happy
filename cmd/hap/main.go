// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package main

import (
	"log/slog"
	"time"

	"github.com/happy-sdk/happy"
)

func main() {
	main := hap()

	main.BeforeAlways(func(sess *happy.Session, flags happy.Flags) error {
		sess.Log().Info("prepare Happy-SDK")

		for _, s := range sess.Profile().All() {
			sess.Log().Ln(s.Key(), slog.String("value", s.Value().String()))
		}

		loader := sess.ServiceLoader(
			"background",
		)

		<-loader.Load()
		return loader.Err()
	})

	main.Before(func(sess *happy.Session, args happy.Args) error {
		sess.Log().Info("main.Before")
		return nil
	})

	main.Do(func(sess *happy.Session, args happy.Args) error {
		sess.Log().Info("main.Do")
		<-sess.UserClosed()
		sess.Log().Info("main.Do DONE")
		return nil
	})

	main.AfterSuccess(func(sess *happy.Session) error {
		sess.Log().Info("main.AfterSuccess")
		return nil
	})

	main.AfterFailure(func(sess *happy.Session, err error) error {
		sess.Log().Info("main.AfterFailure")
		return nil
	})

	main.AfterAlways(func(sess *happy.Session, err error) error {
		sess.Log().Info("main.AfterAlways")
		return nil
	})

	main.Tick(func(sess *happy.Session, ts time.Time, delta time.Duration) error {
		sess.Log().Info("main.Tick")
		return nil
	})

	main.Tock(func(sess *happy.Session, delta time.Duration, tps int) error {
		sess.Log().Info("main.Tock")
		return nil
	})

	main.Run()
}
