// Copyright 2022 The Happy Authors
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
	"time"

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
)

type Action func(s *Session) error

// ActionTickFunc is operation set in given minimal time frame it can be executed.
// You can throttle tick/tocks to cap FPS or for [C|G]PU throttling.
//
// Tock is helper called after each tick to separate
// logic processed in tick and do post processing on tick.
// Tocks are useful mostly for GPU ops which need to do post proccessing
// of frames rendered in tick.
type ActionTick func(sess *Session, ts time.Time, delta time.Duration) error

type Assets interface{}
type Service interface{}

type Event interface {
	Key() string
	Scope() string
	Payload() *vars.Map
	Time() time.Time
}

type Logger interface{}

type Addon interface {
	Register(*Session) (AddonInfo, error)
	Commands() ([]Command, error)
}

type AddonInfo struct {
	Name        string
	Description string
	Version     string
}
