## Static Analysis with Happyvet

`happyvet` provides warnings and fixes for Happy SDK components (e.g., `custom.Logger`, `command.Command`). It integrates with `gopls` (for editors like VS Code, GoLand) and `golangci-lint` (v2.4.0+).

### Installation
Add `happyvet` as a tool dependency:
```bash
go get -tool github.com/happy-sdk/happy/tools/happyvet@v0.102.0
go install tool
