// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import "github.com/google/uuid"

type Action func(ex *Executor) (res Result)

type TaskID = uuid.UUID

type Task struct {
	name      string
	id        uuid.UUID
	action    Action
	dependsOn TaskID
}

func NewTask(name string, action Action) Task {
	return Task{
		name:      name,
		id:        uuid.New(),
		action:    action,
		dependsOn: uuid.Nil,
	}
}

func (t Task) DependsOn(dep TaskID) Task {
	t.dependsOn = dep
	return t
}

func (t Task) ID() TaskID {
	return t.id
}
