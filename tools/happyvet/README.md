## Static Analysis with Happyvet

`happyvet` is a [`go vet`](https://pkg.go.dev/cmd/vet)-style static analyzer for Happy SDK
components. It currently ships one check:

- **`happy_logging`** — flags calls to `*github.com/happy-sdk/happy/pkg/logging.Logger` methods
  (`Debug`, `Info`, `Warn`, `Error`, their `*Context` variants, `Log`, `With`) that don't follow
  slog's alternating key/value convention: a non-string, non-`slog.Attr` key, or a trailing key
  with no value.

```go
l.Info("starting", 11, "attempt")
// happyvet: log/slog.Logger.Info arg "\"attempt\"" should be a string (previous arg "11" cannot be a key)
```

It's built with [`golang.org/x/tools/go/analysis/multichecker`](https://pkg.go.dev/golang.org/x/tools/go/analysis/multichecker),
so it works anywhere a standard analysis-based vet tool works: standalone, through `go vet`,
in editors via `gopls`, and as a `golangci-lint` v2 module plugin.

## Install

As a per-project tool dependency (recommended — pins the version in `go.mod`, no global install).
Note the path: it must be the `cmd/happyvet` **binary** package, not the bare module path (that's
the library entrypoint used for the golangci-lint plugin below, and has no `main` to run):

```bash
go get -tool github.com/happy-sdk/happy/tools/happyvet/cmd/happyvet@latest
go install tool
```

This adds a `tool` directive to your `go.mod`. Run it with `go tool happyvet`, or put `$(go env
GOBIN)`/`$(go env GOPATH)/bin` on your `PATH` to call it as plain `happyvet`.

Or install it globally instead:

```bash
go install github.com/happy-sdk/happy/tools/happyvet/cmd/happyvet@latest
```

## Use

**Standalone**, like any linter — takes package patterns, `-fix` applies suggested fixes,
`-json` emits JSON diagnostics:

```bash
go tool happyvet ./...
# or, if installed globally:
happyvet ./...
happyvet -fix ./...
```

**Through `go vet`** — works with the standard `-vettool` flag:

```bash
go vet -vettool=$(go tool -n happyvet 2>/dev/null || which happyvet) ./...
```

**In editors (gopls-based: VS Code, GoLand, ...)** — point the editor's `go vet` invocation at
`happyvet` via its vet-flags setting. For the VS Code Go extension, in `settings.json`:

```json
{
  "go.vetFlags": ["-vettool=${workspaceFolder}/happyvet-binary-path"]
}
```

replacing the path with wherever `go build -o` (or `go env GOBIN`) put the binary. GoLand exposes
the equivalent under its Go Linter / `go vet` tool configuration.

**As a `golangci-lint` v2 module plugin** (v2.4.0+) — `happyvet` exports the
`func New(settings any) ([]*analysis.Analyzer, error)` entrypoint golangci-lint's module plugin
system expects. Add a `.custom-gcl.yml`:

```yaml
version: v2.4.0 # or your installed golangci-lint version
plugins:
  - module: "github.com/happy-sdk/happy/tools/happyvet"
    import: "github.com/happy-sdk/happy/tools/happyvet"
```

then build and use the custom binary:

```bash
golangci-lint custom      # produces ./custom-gcl
./custom-gcl run
```

and enable it in `.golangci.yml` under `linters.settings.custom` — see golangci-lint's
[module plugin docs](https://golangci-lint.run/plugins/module-plugins/) for the exact schema for
your installed version, since it has changed across releases.
