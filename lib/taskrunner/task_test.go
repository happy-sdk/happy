// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package taskrunner

import (
	"testing"

	"github.com/google/uuid"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestNewTask(t *testing.T) {
	called := false
	action := func(ex *Executor) (res Result) {
		called = true
		return Success("ok")
	}
	task := NewTask("my-task", action)

	testutils.Equal(t, "my-task", task.name, "unexpected task name")
	testutils.NotEqual(t, uuid.Nil, task.id, "expected a non-nil generated id")
	testutils.Equal(t, uuid.Nil, task.dependsOn, "expected no dependency by default")
	testutils.Assert(t, task.action != nil, "expected action to be set")

	_ = task.action(nil)
	testutils.Assert(t, called, "expected the provided action to be stored and callable")
}

func TestTaskDependsOn(t *testing.T) {
	dep := uuid.New()
	original := NewTask("t", func(ex *Executor) (res Result) { return Success("ok") })

	withDep := original.DependsOn(dep)

	testutils.Equal(t, uuid.Nil, original.dependsOn, "expected original task to be unmodified (value receiver)")
	testutils.Equal(t, dep, withDep.dependsOn, "expected new task to carry the dependency")
	testutils.Equal(t, original.id, withDep.id, "expected id to be preserved across DependsOn")
}

func TestTaskID(t *testing.T) {
	task := NewTask("t", func(ex *Executor) (res Result) { return Success("ok") })
	testutils.Equal(t, task.id, task.ID(), "ID() should return the task's id")
}
