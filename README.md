![Happy Logo](https://raw.githubusercontent.com/happy-sdk/happy/main/assets/images/happy.svg)

# Happy Go - Go modules monorepo

**Happy-Go** is the monorepo for select Go packages that are used in conjunction with the Happy SDK. This collection focuses on packages that are beneficial to, but not essential for the Happy SDK. Each package here is crafted to be effective as a standalone module, ensuring wide applicability in various Go projects.

## What is happy-go?

happy-go features a collection of Go packages characterized by:
1. **Standalone Functionality**: Every package in this repository is designed to operate independently of the Happy SDK. This allows these packages to be integrated into your Go projects seamlessly, without requiring the full Happy SDK.
2. **Independence from Third-Party Dependencies**: True to the Go philosophy of simplicity and reliability, each package in this repository avoids external dependencies. Packages within this monorepo may depend on each other, forming a well-integrated yet flexible collection.

![GitHub last commit](https://img.shields.io/github/last-commit/happy-sdk/happy-go)

| package | docs | description |
| --- | --- | --- |
| [devel/testutils](./devel/testutils) | [![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy-go)](https://pkg.go.dev/github.com/happy-sdk/happy-go/devel/testutils) | happy sdk testing utilities |
| | import: | `go get github.com/happy-sdk/happy-go/devel/testutils@latest` |
| [strings/bexp](./strings/bexp) | [![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy-go)](https://pkg.go.dev/github.com/happy-sdk/happy-go/strings/bexp) | Go implementation of Brace Expansion mechanism to generate arbitrary strings. |
| | import: | `go get github.com/happy-sdk/happy-go/strings/bexp@latest` |
| [vars](./vars) | [![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy-go)](https://pkg.go.dev/github.com/happy-sdk/happy-go/vars) | Package vars provides the API to parse variables from various input formats/types to common key value pair vars.Value or variable sets to vars.Map |
| | import: | `go get github.com/happy-sdk/happy-go/vars@latest` |
| [vars/varflag](./vars/varflag) | [![PkgGoDev](https://pkg.go.dev/badge/github.com/happy-sdk/happy-go)](https://pkg.go.dev/github.com/happy-sdk/happy-go/vars/varflag) | Package varflag implements command-line flag parsing into vars.Variables for easy type handling with additional flag types. |
| | import: | `go get github.com/happy-sdk/happy-go/vars/varflag@latest` |
