// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package happy

import (
	"time"

	"github.com/happy-sdk/happy/pkg/vars"
)

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

func NewEvent(scope, key string, payload *vars.Map, err error) Event {
	return &happyEvent{
		ts:      time.Now(),
		key:     key,
		scope:   scope,
		err:     err,
		payload: payload,
	}
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

func registrableEvent(scope, key, desc string, example *vars.Map) Event {
	if example == nil && desc != "" {
		example = new(vars.Map)
	}
	_ = example.Store("event.description", desc)
	return NewEvent(scope, key, example, nil)
}
