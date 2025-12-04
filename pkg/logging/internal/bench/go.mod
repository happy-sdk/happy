module github.com/happy-sdk/happy/pkg/logging/internal/bench

go 1.25

require (
	// Current local logging module (version is overridden by replace below).
	github.com/happy-sdk/happy/pkg/logging v0.0.0

	// Third-party loggers used only for benchmarks.
	// These are isolated in this internal module so they never become
	// transitive dependencies of Happy-SDK users.
	github.com/rs/zerolog v1.33.0
	github.com/sirupsen/logrus v1.9.3
	go.uber.org/zap v1.27.0
)

require (
	github.com/happy-sdk/happy/pkg/bitutils v0.1.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
)

// Use the local logging module from this repo instead of a published version.
replace github.com/happy-sdk/happy/pkg/logging => ../..
