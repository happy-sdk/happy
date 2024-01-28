// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package main

import (
	"log/slog"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/vars"
)

func main() {
	main := hap()

	main.BeforeAlways(func(sess *happy.Session, flags happy.Flags) error {
		sess.Log().Info("prepare Happy-SDK")

		sess.Log().Println("SETTINGS")
		for _, s := range sess.Profile().All() {
			sess.Log().Println(s.Key(), slog.String("value", s.Value().String()))
		}
		sess.Log().Println("OPTIONS")
		sess.Opts().Range(func(v vars.Variable) bool {
			sess.Log().Println(v.Name(), slog.String("value", v.String()))
			return true
		})
		sess.Log().Println("CONFIG")
		sess.Config().Range(func(v vars.Variable) bool {
			sess.Log().Println(v.Name(), slog.String("value", v.String()))
			return true
		})

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

		sess.Log().SystemDebug("main.Do SystemDebug")
		sess.Log().Debug("main.Do Debug")
		sess.Log().Info("main.Do Info")
		sess.Log().Ok("main.Do Ok")
		sess.Log().Notice("main.Do Notice")
		sess.Log().NotImplemented("main.Do NotImplemented")
		sess.Log().Warn("main.Do Warn")
		sess.Log().Deprecated("main.Do Deprecated")
		sess.Log().Error("main.Do Error")
		sess.Log().BUG("main.Do BUG")
		sess.Log().Println("main.Do Println")
		<-sess.UserClosed()
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

	main.Run()
}
