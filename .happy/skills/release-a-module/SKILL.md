---
name: release-a-module
description: >
  Cut a release from the happy monorepo, including the hundred-block version
  decision, per-module tag prefixes, and the post-tag verification step. Use when
  asked to release, cut, tag, bump, or publish any module in this repository.
keywords: [release, tag, publish, version, bump, changelog, hundred-block, happyctl]
requires: [go, golangci-lint, git]
---

# Release a module

Releases are performed by `happyctl`, not by hand. The tool computes versions from
commit history, tags every affected module, pushes, and then verifies the result.
Tagging manually skips the verification that makes a release trustworthy.

**Never run this without explicit confirmation.** It pushes tags to `origin`, and
a published tag is effectively permanent - Go module proxies cache it
immediately.

## 1. Understand what will be released

```bash
go run ./cmd/happyctl info
git status --short
git log --oneline "$(git describe --tags --abbrev=0)"..HEAD
```

`info` prints the resolved config: confirm `releaser.enabled` is `true`,
`releaser.bump.kind` is `minor`, and `releaser.bump.strategy` is `hundred`.

Read the commit log with the version policy in mind. Versions are computed from
Conventional Commit types, so the log *is* the input:

| In the log | Effect |
| --- | --- |
| `feat(…)` | minor |
| `fix` `docs` `deps` `style` `refactor` `perf` `test` `chore` `revert` `ci` `devops` `dev` `wip` | patch |
| `type(scope)!:` or a `BREAKING CHANGE:` footer | major → next full hundred minor |
| anything else | **silently skipped** - not in the changelog, not in the bump |

A typo in the type means the change is invisible to the releaser. Fix the history
before releasing, not after.

## 2. Confirm the version decision

This repository does not use plain semver. Minor versions group in blocks of 100,
and a breaking change jumps to the next full hundred *minor* rather than bumping
major:

```
v1.55.0  + breaking change  →  v1.100.0     (not v2.0.0)
v1.100.0 + feat             →  v1.101.0
v1.101.0 + fix              →  v1.101.1
```

Within a block the minimum Go version and public API stay fixed. Crossing a block
boundary is the signal that either may change - so a Go toolchain bump is also a
block jump, even with no API change.

State the expected version before running the release, then check the tool agrees.
If it does not, work out why before proceeding; it usually means a commit type is
wrong.

## 3. Make the tree clean and green

```bash
go run ./cmd/happyctl lint
go run ./cmd/happyctl test
go run ./cmd/happyctl l10n report -t 95
```

Confirm `golangci-lint` is actually on `PATH` first - `happyctl` silently disables
linting when it is missing, so a green lint on a machine without it means nothing:

```bash
command -v golangci-lint || echo "MISSING - lint results are meaningless"
```

The release pipeline runs lint and test itself; running them first just means you
find problems before touching tags.

## 4. Release

```bash
go run ./cmd/happyctl release
```

The pipeline, in order:

1. **releaseAllowed** - clean tree, correct branch, remote reachable
2. **lint** - every module, unless `linter.enabled` is false
3. **test** - `go test -race` with coverage per module
4. **commit** - `chore(<scope>): :label: prepare release` if anything changed
5. **gomodules** - compute versions, update inter-module requirements, tag
6. **verify** - rebuild and retest each module with `go.work` disabled

Tags use the module path as prefix, so one release produces many:

```
v1.200.0                     ← root module
pkg/vars/v1.200.0
lib/scm/gitutils/v1.100.1
cmd/happyctl/v1.200.0
```

## 5. Do not skip verify

Step 6 exists because `go.work` makes every module satisfy its sibling imports
from local source **regardless of what its own `go.mod` requires**. Every earlier
step - lint, test, `go mod tidy` - resolves through the workspace, so none of them
prove that a downstream consumer can build against what was just tagged.

Verify re-resolves each module against the tags that were actually pushed. If it
fails, the tags are already public: fix forward with a new patch release rather
than deleting tags.

## Flags, and when they are legitimate

| Flag | Use when |
| --- | --- |
| `--dirty` | releasing with uncommitted changes - almost never right |
| `--skip-lint` | linter is broken upstream, and you have linted another way |
| `--skip-tests` | never, for a real release |
| `--skip-remote-checks` | working offline against a mirror |

If you reach for one of these, say why in the release notes.

## Failure modes

**`releasing is disabled`, or `info` shows the wrong `dir.path`** - `happyctl`
resolved a different project root than you expect, so it is reading someone
else's `.happy.yaml` or none at all. With a binary older than
`cmd/happyctl v1.200.1` this happens whenever the repository sits inside another
git repository, such as under the workspace's `src/`. **Check `dir.path` in
step 1 before releasing.** If it is not this repository's root, stop - do not
pass `--dirty` or any skip flag to force past it. See the project root detection
section in `.happy/AGENTS.md`.

**`settings: profile: preferences provided key(…) not found`** - someone added an
unknown key to `.happy.yaml`. The schema is strict; every `happyctl` command fails
until the key is removed or added to `addons/devel/project/project-config.go`.

**Lint fails only in CI** - CI pins `golangci-lint v2.12.2` and passes
`--disable=staticcheck` from `.happy.yaml`. A different local version will
disagree.

**Verify fails after tags are pushed** - do not delete or move tags. Module
proxies have already cached them. Fix the requirement and release a patch.
