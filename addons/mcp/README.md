# addons/mcp

Serves a workspace over the [Model Context Protocol](https://modelcontextprotocol.io).

It discovers the repositories checked out in a workspace, reads the agent
manifest each one ships under `.happy/`, and exposes their tools and skills
namespaced by repository. A client is configured **once**: a repository that
adds a tool, or a contributor who clones another repository, changes what the
server offers with nothing to reconfigure.

Nothing here is specific to any organization. The workspace names where
repositories come from, so the same server works for any set of repositories
developed together.

```
client ──stdio──▶ happyctl mcp serve
                    │  scans <workspace>/src/*/.happy/
                    ├── builtin/get-oriented   served before anything is cloned
                    ├── workspace__repos       what is checked out
                    ├── workspace__skills      procedures, with descriptions
                    ├── workspace__skill       one procedure, on demand
                    ├── workspace__doctor      why something is missing
                    └── <repo>__<tool>         declared by each repository
```

## Use

```bash
go install github.com/happy-sdk/happy/cmd/happyctl@latest
happyctl workspace init
```

Point a client at the server:

```json
{
  "command": "happyctl",
  "args": ["mcp", "serve"],
  "env": { "HAPPY_WORKSPACE": "/path/to/workspace" }
}
```

Set `HAPPY_WORKSPACE` explicitly. Clients launch servers with an unpredictable
working directory, and the fallback search only works if the process happens to
start inside the workspace.

Before wiring it up, check what it sees:

```bash
happyctl mcp list
happyctl mcp doctor
```

`doctor` matters more than it looks. A federated server fails invisibly: a
repository whose manifest has a typo silently contributes nothing, and the only
symptom is a tool that is not there. `doctor` names the file and the reason,
and exits non-zero so it works as a check.

---

# Agent manifest

**Schema version:** `1`

How a repository declares the instructions, skills and tools it offers to
coding agents.

The goal is federation. This addon defines the *contract* and nothing else;
every repository owns its own agent context and ships it alongside the code it
describes, so it is reviewed, versioned and released with that code. Adding a
skill or a tool to a repository must never require a change here.

The format is provider independent: plain files at conventional paths, readable
by any agent that can read a repository. The server consumes exactly the same
manifests for agents that call tools instead.

## Layout

```
<repo>/
├── .happy.yaml                     # optional: relocate the paths below
└── .happy/
    ├── AGENTS.md                   # repository-specific agent instructions
    ├── mcp.yaml                    # tools this repository exposes
    └── skills/
        └── <skill-name>/
            ├── SKILL.md            # required
            └── …                   # optional scripts, templates, references
```

Every part is optional. A repository with only `.happy/AGENTS.md` is valid and
useful. A repository with no `.happy/` has simply not been onboarded, and
agents fall back to its README, contributing guide and CI configuration.

## Discovery

Repositories are enumerated by `lib/workspace` from the workspace's repos
directory; only git working trees are considered. For each one:

| Source | Purpose |
| --- | --- |
| `.happy/AGENTS.md` | instructions, specific to that repository |
| `.happy/skills/*/SKILL.md` | skills, namespaced by repository |
| `.happy/mcp.yaml` | tools, namespaced by repository |
| `.happy.yaml` → `agent:` | optional relocation of the paths above |

- **The checkout directory name is the namespace**, sanitized so it is safe as
  a tool prefix - `happy-sdk.github.io` becomes `happy-sdk-github-io`, since
  clients commonly restrict tool names to `[A-Za-z0-9_-]`. Override it with
  `namespace` in `.happy/mcp.yaml`.
- **Nearest instruction wins.** A repository's `.happy/AGENTS.md` overrides
  anything more general. It should state differences, not restate shared rules.
- **Absent is absent, not inherited.** A repository without a manifest does not
  inherit a neighbour's skills or tools.
- **One broken manifest disables only its own repository**, and is reported by
  `doctor`.

## Relocating the paths

Repositories following the conventional locations need no configuration. To put
them elsewhere, or to opt out entirely:

```yaml
# .happy.yaml
agent:
  enabled: true                  # false withholds everything
  instructions: docs/AGENTS.md
  skills: docs/skills
  mcp: docs/mcp.yaml
```

`happyctl` validates `.happy.yaml` against a strict schema where an unknown key
is a hard error. This addon reads only the `agent` section, and reads it
leniently: a repository is not punished here for configuration this package has
no opinion about, and a manifest it cannot fully understand still yields its
paths.

Because preferences are flattened to `map[string]string` before validation, the
schema supports only scalars and string slices - a list of tool objects cannot
be expressed there at all. That is why this section carries nothing but a
toggle and paths, and every structured definition lives in its own file.

## `.happy/skills/<name>/SKILL.md`

A procedure for a recurring task in that repository. One directory per skill,
named in kebab-case, matching the `name` in its frontmatter.

```markdown
---
name: release-a-module
description: >
  Release a single module, including the version decision and the post-tag
  verification step. Use when asked to cut, tag, or publish a release.
keywords: [release, tag, version, publish, changelog]
requires: [go, git]
---

# Release a module

…procedure…
```

| Field | Required | Meaning |
| --- | --- | --- |
| `name` | yes | kebab-case, unique in the repository, equal to the directory name |
| `description` | yes | what it does **and when to use it** - this is what an agent matches against, so write the trigger, not just the topic |
| `keywords` | no | additional match terms |
| `version` | no | semver of the skill itself |
| `requires` | no | tools the procedure assumes |

Write procedures, not essays: numbered steps, exact commands, expected output,
and the failure modes you have actually hit. Supporting files live beside
`SKILL.md` and are referenced by relative path.

Skills are exposed as `<namespace>/<name>`.

## `.happy/mcp.yaml`

```yaml
version: "1"
namespace: happy                    # optional; defaults to the directory name
description: Happy SDK framework monorepo.

tools:
  - name: test_module
    title: Test one module
    description: >
      Run go test with race detection for a single module. Required because
      go.work spans every module but ./... never crosses a nested go.mod
      boundary.
    input:
      type: object
      required: [module]
      properties:
        module:
          type: string
          description: Module directory relative to the repository root.
        run:
          type: string
          description: Optional -run regexp selecting individual tests.
    exec:
      command: ["go", "test", "-race", "./..."]
      cwd: "{{ .module }}"
      args:
        - if: "{{ .run }}"
          add: ["-run", "{{ .run }}"]
      timeout: 5m
    output: text
```

| Key | Meaning |
| --- | --- |
| `name` | snake_case, unique within the namespace |
| `title` | short human-readable label |
| `description` | what it does and when to call it; the primary selection signal |
| `input` | JSON Schema object describing arguments; omit for none |
| `exec.command` | argv, executed without a shell |
| `exec.cwd` | working directory, repository-relative; may interpolate inputs |
| `exec.args` | conditional argv additions |
| `exec.timeout` | duration; the process is killed past it |
| `output` | `text` or `json` |

Interpolation is `{{ .field }}` against validated inputs. A **declared** input
the caller omitted renders empty, which is how optional arguments stay
optional; a placeholder naming a field the schema **never declares** is a
manifest bug and fails loudly.

Tools are exposed as `<namespace>__<name>`.

A repository may instead delegate to a server it ships itself:

```yaml
server:
  command: ["./scripts/my-mcp-server"]
  transport: stdio
```

This is parsed and reported but **not yet spawned**.

## Trust

Manifests declare commands that execute on a maintainer's machine. That is the
point, and it is also the risk, so execution is deliberately narrow:

- **No shell.** `exec.command` is argv, executed directly.
- **Inputs are data.** They interpolate into separate argv entries, never into
  a command line, so shell metacharacters in an argument are inert.
- **Only declared inputs** may be referenced.
- **Confined.** `cwd` must resolve inside the declaring repository.
- **Bounded.** Every run has a timeout and is killed as a process group, so a
  command that spawns children does not leave them behind.
- **Workspace only.** Nothing outside the resolved workspace root is scanned.

Treat a pull request that adds or edits `.happy/mcp.yaml` the way you would one
adding a build script, because that is what it is.

## Onboarding a repository

1. Create `.happy/AGENTS.md` with what is genuinely non-obvious about the
   repository. Start there and stop there if that is all you have.
2. Add skills as recurring procedures become clear. Do not write speculative
   ones; a skill that is never the right answer still costs attention.
3. Add `.happy/mcp.yaml` only when there are commands worth calling directly.
4. Leave `.happy.yaml` alone unless the files live somewhere unconventional.

## Not built yet

Proxying a repository's own downstream `server:`, streamable HTTP transport,
manifest hot-reload with `tools/list_changed`, and exposing skills as MCP
prompts.
