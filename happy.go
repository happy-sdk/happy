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

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/api"
	"github.com/happy-sdk/happy/sdk/app"
	"github.com/happy-sdk/happy/sdk/services"
	"github.com/happy-sdk/happy/sdk/session"
)

var ErrNotImplemented = errors.New("not implemented")

func New(c *Settings, extend ...settings.Settings) *app.Main {
	if c == nil {
		c = &Settings{}
	}
	for _, ext := range extend {
		c.extend(ext)
	}
	return app.New(c)
}

// API returns the API for the given type if it is registered.
func API[API api.Provider](sess *session.Context) (api API, err error) {
	err = session.API(sess, &api)
	return
}

// ServiceLoader is a convenience function to create a new ServiceLoader instance from the services package.
// It initializes and prepares the specified services for loading using the provided session context,
// acting as a shorthand for services.NewLoader within the Happy-SDK framework.
//
// Parameters:
//   - sess: The session context used to configure the service loader.
//   - svcs: A variadic list of service names or addresses to be loaded and started.
//
// Returns:
//   - A pointer to a ServiceLoader instance configured to load the specified services.
//
// Example usage:
//
//	loader := ServiceLoader(sess, "serviceA", "serviceB")
//	<-loader.Load()
//	if err := loader.Err(); err != nil {
//	    log.Fatal("Service loading failed:", err)
//	}
func ServiceLoader(sess *session.Context, svcs ...string) *services.ServiceLoader {
	return services.NewLoader(sess, svcs...)
}
