// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

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

	"github.com/mkungla/happy/pkg/varflag"
	"github.com/mkungla/happy/pkg/vars"
)

var (
	ErrApplication      = errors.New("application error")
	ErrCommand          = errors.New("command error")
	ErrCommandFlags     = errors.New("command flags error")
	ErrCommandAction    = errors.New("command action error")
	ErrInvalidVersion   = errors.New("invalid version")
	ErrEngine           = errors.New("engine error")
	ErrSessionDestroyed = errors.New("session destroyed")
	ErrService          = errors.New("service error")
	ErrHappy            = errors.New("not so happy")
	ErrAddon            = errors.New("addon error")
)

type Action func(sess *Session) error

// ActionTickFunc is operation set in given minimal time frame it can be executed.
// You can throttle tick/tocks to cap FPS or for [C|G]PU throttling.
//
// Tock is helper called after each tick to separate
// logic processed in tick and do post processing on tick.
// Tocks are useful mostly for GPU ops which need to do post proccessing
// of frames rendered in tick.
type ActionTick func(sess *Session, ts time.Time, delta time.Duration) error
type ActionTock func(sess *Session, delta time.Duration, tps int) error
type ActionWithArgs func(sess *Session, args Args) error
type ActionWithOptions func(sess *Session, opts *Options) error
type ActionWithEvent func(sess *Session, ev Event) error
type ActionMigrate func(ver Version, sess *Session) error

type Assets interface{}

type Event interface {
	Key() string
	Scope() string
	Payload() *vars.Map
	Time() time.Time
}

type EventListener interface {
	OnEvent(scope, key string, cb ActionWithEvent)
	OnAnyEvent(ActionWithEvent)
}

type TickerFuncs interface {
	// OnTick enables you to define func body for operation set
	// to call in minimal timeframe until session is valid and
	// service is running.
	OnTick(ActionTick)

	// OnTock is helper called right after OnTick to separate
	// your primary operations and post prossesing logic.
	OnTock(ActionTick)
}

type Args interface {
	Arg(i uint) vars.Value
	ArgDefault(i uint, value any) (vars.Value, error)
	ArgVarDefault(i uint, key string, value any) (vars.Variable, error)
	Args() []vars.Value
	Flag(name string) varflag.Flag
}

type args struct {
	argv  []vars.Value
	argn  uint
	flags varflag.Flags
}

func (a *args) Arg(i uint) vars.Value {
	if a.argn <= i {
		return vars.EmptyValue
	}
	return a.argv[i]
}

func (a *args) ArgDefault(i uint, value any) (vars.Value, error) {
	if a.argn <= i {
		return vars.NewValue(value)
	}
	return a.Arg(i), nil
}

func (a *args) ArgVarDefault(i uint, key string, value any) (vars.Variable, error) {
	if a.argn <= i {
		return vars.New(key, value, true)
	}
	return vars.New(key, a.argv[i], true)
}

func (a *args) Args() []vars.Value {
	return a.argv
}

func (a *args) Flag(name string) varflag.Flag {
	f, err := a.flags.Get(name)
	if err != nil {
		ff, _ := varflag.Bool("unknown", false, "")
		return ff
	}
	return f
}

func NewEvent(scope, key string, payload *vars.Map, err error) Event {
	return &happyEvent{
		ts:      time.Now(),
		key:     key,
		scope:   scope,
		err:     err,
		payload: payload,
	}
}

func registerEvent(scope, key, desc string, example *vars.Map) Event {
	if example == nil && desc != "" {
		example = new(vars.Map)
	}
	example.Store("happy.app.event.description", desc)
	return NewEvent(scope, key, example, nil)
}

type happyEvent struct {
	ts      time.Time
	scope   string
	key     string
	err     error
	payload *vars.Map
}

func (ev *happyEvent) Time() time.Time {
	return ev.ts
}

func (ev *happyEvent) Scope() string {
	return ev.scope
}

func (ev *happyEvent) Key() string {
	return ev.key
}
func (ev *happyEvent) Err() error {
	return ev.err
}

func (ev *happyEvent) Payload() *vars.Map {
	return ev.payload
}

type API interface {
	Get(key string) vars.Variable
}

func GetAPI[A API](sess *Session, addonName string) (api A, err error) {
	papi, err := sess.API(addonName)
	if err != nil {
		return api, err
	}
	if aa, ok := papi.(A); ok {
		return aa, nil
	}
	return api, fmt.Errorf("unable to cast %s API to given type", addonName)
}

type Version string
