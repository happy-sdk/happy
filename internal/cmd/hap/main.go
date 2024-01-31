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

		loader := sess.ServiceLoader(
			"background",
		)

		<-loader.Load()

		sess.Log().Debug("SETTINGS")
		for _, s := range sess.Profile().All() {
			sess.Log().Debug(s.Key(), slog.String("value", s.Value().String()))
		}
		sess.Log().Debug("OPTIONS")
		sess.Opts().Range(func(v vars.Variable) bool {
			sess.Log().Debug(v.Name(), slog.String("value", v.String()))
			return true
		})
		sess.Log().Debug("CONFIG")
		sess.Config().Range(func(v vars.Variable) bool {
			sess.Log().Debug(v.Name(), slog.String("value", v.String()))
			return true
		})
		return loader.Err()
	})

	// main.Before(func(sess *happy.Session, args happy.Args) error {
	// 	sess.Log().Info("main.Before")
	// 	return nil
	// })

	// main.Do(func(sess *happy.Session, args happy.Args) error {
	// 	return nil
	// })

	// main.AfterSuccess(func(sess *happy.Session) error {
	// 	sess.Log().NotImplemented("main.AfterSuccess")
	// 	return nil
	// })

	// main.AfterFailure(func(sess *happy.Session, err error) error {
	// 	sess.Log().NotImplemented("main.AfterFailure")
	// 	return nil
	// })

	// main.AfterAlways(func(sess *happy.Session, err error) error {
	// 	sess.Log().NotImplemented("main.AfterAlways")
	// 	return nil
	// })

	main.Run()
}
