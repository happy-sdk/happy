# AGENTS.md

Agent instructions for the `happy` monorepo. Org-wide rules live in the workspace
root `AGENTS.md`; this file covers what is specific to this repository and
overrides the org file where they disagree.

## Shape of the repository

A Go multi-module monorepo: ~28 modules, one `go.work`, each module versioned and
released independently.

```
pkg/       generic, reusable, ZERO third-party dependencies
           (vars, settings, options, logging, i18n, tui, version, fsutils, …)
sdk/       framework internals, also zero third-party deps
           (app, session, cli, services, addon, api, events, engine, migration, stats)
happy      root package: thin facade — New(), API[T](), ServiceLoader()
lib/       reusable libraries that MAY pull third-party deps
           (taskrunner→bubbletea, scm/gitutils→go-git, changelog, tail)
addons/    feature bundles of commands+services (devel, l10n)
cmd/       happyctl — the org's dev tool, built from this repo
tools/     happyvet — go vet-style analyzer
```

**The zero-dependency rule in `pkg/` and `sdk/` is load-bearing.** It is why an
application built on Happy adds no transitive dependencies, and it decides which
`i18n.Embed` variant a package uses. Never add a third-party import to `pkg/**` or
`sdk/**`; if a feature needs one, it belongs in `lib/**` or `addons/**`.

## Testing: the one thing that will fool you

`go.work` spans every module, but **`./...` never crosses a nested `go.mod`
boundary**. `go test ./...` at the repository root tests only the root module and
exits 0 while your change is broken.

```bash
cd pkg/vars && go test ./...
cd pkg/vars && go test -run TestValueBool ./...     # single test
cd pkg/i18n && go test -race -run TestManager .
```

Whole-monorepo sweep, exactly as CI runs it:

```bash
./.github/actions/go-test-monorepo-action/go-test-monorepo-action.sh false true true false
#                                          only-list ↑  race ↑ fail-fast ↑ continue-on-error ↑
```

Or through the project tool, which also grades per-module coverage:

```bash
go run ./cmd/happyctl test
go run ./cmd/happyctl lint
go run ./cmd/happyctl info                 # resolved .happy.yaml schema
go run ./cmd/happyctl l10n report -t 95    # translation gate; CI runs exactly this
```

`golangci-lint` must be on `PATH`. `happyctl` detects it with `exec.LookPath` and
**silently disables linting** when absent - a passing lint run on a machine
without it checked nothing. There is no `.golangci.yml`; policy is defaults plus
`--disable=` from `.happy.yaml` (currently `staticcheck`, which panics on
go1.27rc2 toolchains).

`tools/happyvet` enforces slog-style alternating key/value args on
`pkg/logging.Logger` calls. It is a `tool` directive in the root `go.mod`:

```bash
go tool happyvet ./...     # from inside a module
```

## Project root detection: repository boundary

`project.FindProjectDir` recurses to the parent and **prefers the outermost**
project directory, so that a module directory resolves to the monorepo root
rather than to itself. But `IsProjectDir(dir, all=true)` treats any git repository
as a project, so when this repository was checked out inside another git
repository - exactly what the `happy-sdk/.github` workspace does with `src/happy`
- detection escaped into the outer repository, reporting the wrong `dir.path` and
never reading this repository's `.happy.yaml`. `lint` and `test` were the damaging
cases: both call `GoModules`, which enumerates from the resolved root, so they
swept every module in every sibling checkout.

Fixed by treating a git repository root as a hard boundary that stops the ascent -
a repository nested inside another is a separate project by definition. Module
directories inside this monorepo are not repository roots, so monorepo detection
is unaffected. `TestFindProjectDirStopsAtNestedRepositoryRoot` and
`TestFindProjectDirAscendsWithinNestedRepository` pin the boundary;
`TestFindProjectDirResolvesMonorepoModuleToRepositoryRoot` pins the behaviour the
outermost-match logic exists for. Change any of it and all three should be
consulted.

Released `happyctl` builds up to and including `cmd/happyctl v1.200.0` predate the
fix, so an installed binary still misbehaves in a nested checkout.

## settings vs options vs vars

Three layers that are easy to confuse, and picking the wrong one produces code
that looks right and fits badly.

- **`pkg/vars`** — typed value/variable primitives (`Value`, `Variable`, `Kind`).
  The substrate everything else is built on.
- **`pkg/settings`** — *declared* configuration schema. Structs with
  `key:"name,save"` / `default:"…"` tags implement `Blueprint()`; a `Blueprint`
  compiles to a `Schema` → `Profile` → merged user `Preferences`. Reached via
  `sess.Settings()`.
- **`pkg/options`** — *runtime* key/value options with validation, parsers,
  read-only/once flags, and sealing; keys namespaced on `Spec.Extend`. Reached via
  `sess.Opts()` / `sess.Get("app.…")`.

Settings are declared once and validated at load. Options change while the
application runs. If a value comes from a config file, it is a setting.

## `.happy.yaml` is strict, and flat

Preferences are decoded into `map[string]string` and validated against the schema
in `addons/devel/project/project-config.go`. Two consequences:

1. **An unknown key is a hard error** that breaks *every* `happyctl` command in
   the repository, not a warning:
   ```
   settings: profile: preferences provided key(agent.mcp.enabled) not found
   ```
   Never add a key without adding the corresponding field to `project.Config`
   first. `project.go:251` is where it fails.
2. **Nested lists of objects cannot be expressed.** Only scalars and string
   slices survive flattening. Structured configuration belongs in its own file
   under `.happy/`, referenced by a path setting.

## Runtime model

`happy.New(*Settings)` → `*app.Main` (builder: `Do`, `Before*`, `After*`,
`WithAddons`, `WithCommands`, `WithServices`, `WithFlags`, `WithBrand`,
`Tick`/`Tock`) → `Run()`. `sdk/app/internal/initializer` assembles an
`application.Runtime`; `sdk/app/engine` drives services, events, ticks, and cron
once a blocking command runs.

Every action receives `*session.Context` - the single handle to logging,
settings, options, events, and APIs. Addons expose typed APIs retrieved with
`happy.API[*MyAPI](sess)`.

## i18n

Keys are reverse-DNS paths rooted at the registering package's import path. Each
package declares `const i18np = "com.github.happy-sdk.happy.sdk.app"` and embeds
`locales/*.json` from `init()`. Which registration function to use follows the
layering rule:

| Function | Use in | Why |
| --- | --- | --- |
| `i18n.MustEmbed` | `happy`, `sdk/**`, `pkg/**` | no third-party deps, never released broken - fail loudly at start |
| `i18n.Embed` | `lib/**`, `addons/**` | issues surface on the app's `Initialize`/`Reload` |
| `i18n.EmbedIssues` | packages usable outside a happy app | caller may never call `Initialize` |

Locale set: `en, de, et, fi, fr, nl, ru, sv`. Render with `i18n.T` / `i18n.TL` /
`i18n.PTD`. The `addons/l10n` addon provides `l10n report|list|translate|generate|tui`
for maintaining bundles; CI fails below 95% coverage.

## Releasing

`go run ./cmd/happyctl release` runs lint → test → commit → tag → push → verify.
Flags: `--dirty`, `--skip-lint`, `--skip-tests`, `--skip-remote-checks`.

Version policy is **hundred-block**, configured as
`releaser.bump: {kind: minor, strategy: hundred}`: a breaking change or Go version
sync jumps to the next full hundred minor (`v1.55.0` → `v1.100.0`) instead of
bumping major. Tags carry the module path as prefix (`pkg/vars/v1.200.0`); the
root module is plain `v1.200.0`.

The final **verify** step rebuilds and retests each released module with `go.work`
disabled. This matters: `go.work` makes every module resolve its siblings from
local source regardless of what its own `go.mod` requires, so nothing earlier in
the pipeline proves the published tags work together. Do not skip it.

See the `release-a-module` skill for the full procedure.
