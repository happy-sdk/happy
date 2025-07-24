// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package events

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/happy-sdk/happy/pkg/vars"
)

type Event interface {
	// Returns the event's scope.
	Scope() string
	// Returns the event's key.
	Key() string
	// Returns the time at which the event was created using Create.
	Time() time.Time
	// Returns the event's optional value string.
	String() string
	// Returns the event's optional value as vars.Value
	Value() vars.Value
	// Returns the event's optional payload.
	Payload() *vars.ReadOnlyMap
	// Creates a new event with the given optional value and payload.
	Create(value any, payload *vars.Map) Event
}

func New(scope, key string) Event {
	return &event{
		scope: scope,
		key:   key,
	}
}

type event struct {
	ts      time.Time
	scope   string
	key     string
	payload *vars.ReadOnlyMap
	value   vars.Value
}

func (ev *event) Scope() string {
	return ev.scope
}

func (ev *event) Key() string {
	return ev.key
}

func (ev *event) Time() time.Time {
	return ev.ts
}

func (ev *event) Payload() *vars.ReadOnlyMap {
	return ev.payload
}

func (ev *event) Create(value any, payload *vars.Map) Event {
	var (
		pl  *vars.ReadOnlyMap
		err error
	)
	if payload != nil {
		pl, err = vars.ReadOnlyMapFrom(payload)
		if err != nil {
			key := ev.key
			if ev.scope != "" {
				key += "(" + ev.scope + ")"
			}
			slog.Error(fmt.Sprintf("%s: %s", key, err.Error()))
		}
	}
	val, err := vars.NewValue(value)
	if err != nil {
		val = vars.NilValue
	}
	return &event{
		ts:      time.Now(),
		scope:   ev.scope,
		key:     ev.key,
		payload: pl,
		value:   val,
	}
}

func (ev *event) String() string {
	return ev.value.String()
}

func (ev *event) Value() vars.Value {
	return ev.value
}

type ActionWithEvent[SESS context.Context] func(sess SESS, ev Event) error

type Listener[SESS context.Context] interface {
	OnEvent(scope, key string, cb ActionWithEvent[SESS])
	OnAnyEvent(cb ActionWithEvent[SESS])
}
