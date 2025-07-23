module github.com/happy-sdk/happy

go 1.24.0

require (
	github.com/happy-sdk/happy/pkg/branding v0.3.2
	github.com/happy-sdk/happy/pkg/cli/ansicolor v0.3.1
	github.com/happy-sdk/happy/pkg/devel/testutils v1.1.0
	github.com/happy-sdk/happy/pkg/options v0.6.0
	github.com/happy-sdk/happy/pkg/scheduling/cron v0.5.2
	github.com/happy-sdk/happy/pkg/settings v0.6.0
	github.com/happy-sdk/happy/pkg/strings/humanize v0.5.2
	github.com/happy-sdk/happy/pkg/strings/slug v0.2.1
	github.com/happy-sdk/happy/pkg/strings/textfmt v0.5.0
	github.com/happy-sdk/happy/pkg/vars v0.18.0
	github.com/happy-sdk/happy/pkg/version v0.4.0
	golang.org/x/sys v0.33.0
	golang.org/x/text v0.26.0
)

require (
	github.com/happy-sdk/happy/pkg/strings/bexp v1.5.2 // indirect
	golang.org/x/mod v0.25.0 // indirect
)

replace github.com/happy-sdk/happy/pkg/branding => /devel/happy-sdk/happy/pkg/branding

replace github.com/happy-sdk/happy/pkg/devel/testutils => /devel/happy-sdk/happy/pkg/devel/testutils

replace github.com/happy-sdk/happy/pkg/options => /devel/happy-sdk/happy/pkg/options

replace github.com/happy-sdk/happy/pkg/scheduling/cron => /devel/happy-sdk/happy/pkg/scheduling/cron

replace github.com/happy-sdk/happy/pkg/settings => /devel/happy-sdk/happy/pkg/settings

replace github.com/happy-sdk/happy/pkg/strings/humanize => /devel/happy-sdk/happy/pkg/strings/humanize

replace github.com/happy-sdk/happy/pkg/strings/slug => /devel/happy-sdk/happy/pkg/strings/slug

replace github.com/happy-sdk/happy/pkg/strings/textfmt => /devel/happy-sdk/happy/pkg/strings/textfmt

replace github.com/happy-sdk/happy/pkg/vars => /devel/happy-sdk/happy/pkg/vars

replace github.com/happy-sdk/happy/pkg/version => /devel/happy-sdk/happy/pkg/version

replace github.com/happy-sdk/happy/pkg/strings/bexp => /devel/happy-sdk/happy/pkg/strings/bexp
