## Static Analysis with Happyvet

`happyvet` provides warnings and fixes for Happy SDK components (e.g. `github.com/happy-sdk/happy/pkg/logging.Logger`). It integrates with `gopls` (for editors like VS Code, GoLand) and `golangci-lint` (v2.4.0+).

### Installation
Add `happyvet` as a tool dependency:
```bash
go get -tool github.com/happy-sdk/happy/tools/happyvet@v0.2.0
go install tool
