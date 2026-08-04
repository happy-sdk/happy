---
name: release-a-module
description: >
  Understand how releases work here and prepare one for a maintainer: the
  hundred-block version decision, per-module tag prefixes, and what the
  pipeline verifies. Use when asked about releasing, or to get a repository
  ready to release. The release itself is run by a maintainer at a terminal -
  never run it yourself.
keywords: [release, tag, publish, version, bump, changelog, hundred-block, happyctl]
requires: [go, golangci-lint, git]
---

# Releasing

## Do not run the release

`happyctl release` is a maintainer's command, run by hand at a terminal. Not
"with confirmation" — **do not run it at all**.

It is interactive: the pipeline stops at a terminal prompt showing every module
that would be published and waits for a decision. There is nothing for an agent
to confirm on its behalf, and the command refuses to start without a terminal
for exactly that reason:

```
project: releasing requires an interactive terminal; it stops to confirm what
will be published, and a published tag cannot be taken back.
```

That guard fails fast, before anything happens. It exists because the
confirmation sits *after* lint, test, and the `prepare release` commit, so
failing at the prompt instead would leave a stray commit behind.

The deeper reason is that a release cannot be undone. It tags and pushes, and
module proxies cache a published tag immediately — a mistake is fixed by
releasing again, never by deleting.

**Your job is everything up to that point.** Get the repository into a state
where the maintainer can run one command and trust the result, then say so and
stop.

## 1. Check what would be released

```bash
go run ./cmd/happyctl info
git status --short
git log --oneline "$(git describe --tags --abbrev=0)"..HEAD
```

`info` prints the resolved config: confirm `releaser.enabled` is `true`,
`releaser.bump.kind` is `minor`, `releaser.bump.strategy` is `hundred`, and
that `dir.path` is this repository.

Read the commit log with the version policy in mind. Versions are computed from
Conventional Commit types, so the log *is* the input:

| In the log | Effect |
| --- | --- |
| `feat(…)` | minor |
| `fix` `docs` `deps` `style` `refactor` `perf` `test` `chore` `revert` `ci` `devops` `dev` `wip` | patch |
| `type(scope)!:` or a `BREAKING CHANGE:` footer | major → next full hundred minor |
| anything else | **silently skipped** — not in the changelog, not in the bump |

A typo in the type makes a change invisible to the releaser. Report it before
the release, not after.

## 2. State the expected version

This repository does not use plain semver. Minor versions group in blocks of
100, and a breaking change jumps to the next full hundred *minor* rather than
bumping major:

```
v1.55.0  + breaking change  →  v1.100.0     (not v2.0.0)
v1.100.0 + feat             →  v1.101.0
v1.101.0 + fix              →  v1.101.1
```

Within a block the minimum Go version and public API stay fixed. Crossing a
block boundary signals that either may change — so a Go toolchain bump is also
a block jump, even with no API change.

Work out what each affected module should become and say so, so the maintainer
can check the tool agrees. A mismatch usually means a commit type is wrong.

## 3. Make it clean and green

```bash
command -v golangci-lint || echo "MISSING - lint results are meaningless"
go run ./cmd/happyctl lint
go run ./cmd/happyctl test
go run ./cmd/happyctl l10n report -t 95
```

Check `golangci-lint` is actually on `PATH` first: `happyctl` silently disables
linting when it is missing, so a green lint on a machine without it has
verified nothing.

The pipeline runs lint and test itself. Running them first means problems are
found before the maintainer starts, which is the point.

## 4. Hand over

Report what you verified, the versions you expect, and anything you could not
check. Then stop. The maintainer runs:

```bash
happyctl release
```

## What their run will do

1. **releaseAllowed** — clean tree, correct branch, remote reachable
2. **lint** — every module
3. **test** — `go test -race` with coverage per module
4. **commit** — `chore(<scope>): :label: prepare release` if anything changed
5. **confirm** — the interactive prompt
6. **gomodules** — compute versions, update inter-module requirements, tag
7. **verify** — rebuild and retest each module with `go.work` disabled

Tags use the module path as prefix, so one release produces many:

```
v1.202.0                     ← root module
pkg/vars/v1.200.1
lib/scm/gitutils/v1.101.0
cmd/happyctl/v1.201.0
```

Step 7 is the one worth understanding. `go.work` makes every module satisfy its
sibling imports from local source **regardless of what its own `go.mod`
requires**, so lint, test and `go mod tidy` all resolve through the workspace
and none of them prove a downstream consumer can build against what was just
tagged. Verify re-resolves each module against the tags actually pushed. It is
also what adds a requirement on a new sibling module that no `go.mod` mentions
yet — it discovers those from real imports via `go list`.

If verify fails, the tags are already public: fix forward with a patch release.
Never delete or move a published tag.

## Failure modes

**`releasing is disabled`** — `releaser.enabled` is false, or `happyctl`
resolved a different project. Check `dir.path` in `info`.

**`settings: profile: preferences provided key(…) not found`** — someone added
an unknown key to `.happy.yaml`. The schema is strict; every `happyctl` command
fails until the key is removed or added to
`addons/devel/project/project-config.go`.

**Lint fails only in CI** — CI pins `golangci-lint v2.12.2` and passes
`--disable=staticcheck` from `.happy.yaml`. A different local version will
disagree.
