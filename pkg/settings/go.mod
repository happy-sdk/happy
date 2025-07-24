module github.com/happy-sdk/happy/pkg/settings

go 1.24

require (
	github.com/happy-sdk/happy/pkg/vars v0.18.1
	golang.org/x/text v0.26.0
	golang.org/x/mod v0.25.0
	github.com/happy-sdk/happy/pkg/version v0.4.1
)

replace github.com/happy-sdk/happy/pkg/vars => /devel/happy-sdk/happy/pkg/vars

replace github.com/happy-sdk/happy/pkg/version => /devel/happy-sdk/happy/pkg/version
