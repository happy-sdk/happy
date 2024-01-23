// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

// Package happy provides a modular framework for rapid prototyping in Go. With this SDK, developers
// of all levels can easily bring their ideas to life. Whether you're a hacker or a creator, Package
// happy has everything you need to tackle your domain problems and create working prototypes or MVPs
// with minimal technical knowledge and infrastructure planning.
//
// Its modular design enables you to package your commands and services into reusable addons, so you're
// not locked into any vendor tools. It also fits well into projects where different components are written
// in different programming languages.
//
// Let Package happy help you bring your projects from concept to reality and make you happy along the way.
package happy

import (
	"errors"
	"fmt"
	"time"

	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
)

var (
	Error = errors.New("happy")
)

type Action func(sess *Session) error
type ActionWithFlags func(sess *Session, flags Flags) error
type ActionWithArgs func(sess *Session, args Args) error
type ActionTick func(sess *Session, ts time.Time, delta time.Duration) error
type ActionTock func(sess *Session, delta time.Duration, tps int) error
type ActionWithPrevErr func(sess *Session, err error) error
type ActionWithEvent func(sess *Session, ev Event) error
type ActionWithOptions func(sess *Session, opts *Options) error

type Args interface {
	Arg(i uint) vars.Value
	ArgDefault(i uint, value any) (vars.Value, error)
	Args() []vars.Value
	Argn() uint
	Flag(name string) varflag.Flag
}

type Flags interface {
	// Get named flag
	Get(name string) (varflag.Flag, error)
	Args() []vars.Value
}

type API interface {
	Get(key string) vars.Variable
	Set(key string, val any) error
}

func GetAPI[A API](sess *Session, addonName string) (api A, err error) {
	papi, err := sess.API(addonName)
	if err != nil {
		return api, err
	}
	if aa, ok := papi.(A); ok {
		return aa, nil
	}
	return api, fmt.Errorf("%w: unable to cast %s API to given type", Error, addonName)
}
