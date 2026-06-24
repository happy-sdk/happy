// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

// Package api defines the marker interface used to expose a custom API from
// an addon or service to the rest of an application.
package api

// Provider marks a type as a Happy API that can be registered via
// addon.Addon.ProvideAPI and looked up elsewhere in the application via
// happy.API.
//
// Provider's method is unexported, so it cannot be implemented directly
// outside this package. Instead, embed api.Provider (as a zero-value,
// unused field) in your own API type; embedding an interface satisfies it
// for any consumer:
//
//	type MyAPI struct {
//		api.Provider
//		// ... your API's fields/methods
//	}
//
//	ad.ProvideAPI(&MyAPI{})
//
// See github.com/happy-sdk/happy/sdk/stats.Profiler for a real usage
// example.
type Provider interface {
	happy() bool
}
