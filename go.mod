module github.com/happy-sdk/happy

go 1.25.0

require (
	github.com/happy-sdk/happy/pkg/branding v1.0.0
	github.com/happy-sdk/happy/pkg/bytesize v1.0.1
	github.com/happy-sdk/happy/pkg/devel/goutils v1.0.1
	github.com/happy-sdk/happy/pkg/devel/testutils v1.3.2
	github.com/happy-sdk/happy/pkg/fsutils v1.0.3
	github.com/happy-sdk/happy/pkg/i18n v1.1.3
	github.com/happy-sdk/happy/pkg/logging v1.0.2
	github.com/happy-sdk/happy/pkg/networking v1.0.0
	github.com/happy-sdk/happy/pkg/options v1.0.1
	github.com/happy-sdk/happy/pkg/scheduling/cron v1.0.2
	github.com/happy-sdk/happy/pkg/settings v1.1.5
	github.com/happy-sdk/happy/pkg/strings/slug v1.0.0
	github.com/happy-sdk/happy/pkg/strings/textfmt v1.0.1
	github.com/happy-sdk/happy/pkg/tui v1.0.0
	github.com/happy-sdk/happy/pkg/vars v1.1.3
	github.com/happy-sdk/happy/pkg/version v1.0.1
	golang.org/x/mod v0.38.0
	golang.org/x/term v0.45.0
	golang.org/x/text v0.40.0
)

require (
	github.com/happy-sdk/happy/pkg/bitutils v1.0.1 // indirect
	github.com/happy-sdk/happy/pkg/strings/bexp v1.7.2 // indirect
	github.com/happy-sdk/happy/tools/happyvet v1.0.1 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

tool github.com/happy-sdk/happy/tools/happyvet/cmd/happyvet
