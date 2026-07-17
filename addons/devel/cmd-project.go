// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package devel

// func cmdProjectRelease() *command.Command {
// 	return command.New("release",
// 		command.Config{
// 			Description: "Release current project",
// 		}).
// 		WithFlags(
// 			cli.NewBoolFlag("dirty", false, "allow release from dirty git repository"),
// 		).
// 		Disable(func(sess *session.Context) error {

// 			return errors.New("project does not have releaser")
// 		}).
// 		Do(func(sess *session.Context, args action.Args) error {
// 			sess.Log().NotImplemented("release command not implemented")
// 			return nil
// 		})
// }

// func cmdProjectTest() *command.Command {
// 	return command.New("test",
// 		command.Config{
// 			Description: "Run project tests",
// 		}).
// 		Disable(func(sess *session.Context) error {

// 			return errors.New("project does not have tests")
// 		}).
// 		Do(func(sess *session.Context, args action.Args) error {
// 			sess.Log().NotImplemented("test command not implemented")
// 			return nil
// 		})
// }

// func cmdProjectRun() *command.Command {
// 	return command.New("run",
// 		command.Config{
// 			Description: "Run project task",
// 			MinArgs:     1,
// 			MinArgsErr:  "no task name provided",
// 		}).
// 		Disable(func(sess *session.Context) error {

// 			return errors.New("project does not have any tasks")
// 		}).
// 		Do(func(sess *session.Context, args action.Args) error {
// 			sess.Log().NotImplemented("tasks command not implemented")
// 			return nil
// 		})
// }
// func cmdProjectTasks() *command.Command {
// 	return command.New("tasks",
// 		command.Config{
// 			Description: "List project tasks",
// 		}).
// 		Do(func(sess *session.Context, args action.Args) error {
// 			// api, err := happy.API[*API](sess)
// 			// if err != nil {
// 			// 	return err
// 			// }

// 			// project, err := api.Project()
// 			// if err != nil {
// 			// 	return err
// 			// }
// 			// if !project.Has(projects.HasTasks) {
// 			// 	sess.Log().Warn("project does not have any tasks")
// 			// 	return nil
// 			// }

// 			sess.Log().NotImplemented("tasks command not implemented")
// 			return nil
// 		})
// }
