---
name: get-oriented
description: >
  Establish what a workspace contains and which instructions apply before
  changing anything in it. Use at the start of any task in a workspace, and
  again whenever work moves from one repository to another.
keywords: [workspace, orientation, discovery, repository, context, instructions]
requires: [git]
---

# Get oriented

This skill is built into the server, so it is available before anything is
cloned. It describes how a workspace is arranged, not what any particular
organization expects — a repository's own instructions always win over this.

## 1. See what is here

```
workspace__repos
```

That lists every checkout, its namespace, and whether it declares agent
context. Repositories under a workspace are **separate repositories** with
their own remotes and review processes; a change in one is a change to that
project, not to the workspace.

The workspace itself is local state — a marked directory, not a repository.
Nothing in it is committed except within the checkouts.

## 2. Read the workspace instructions

The workspace may declare an instructions file, usually inside one of the
cloned repositories. Read it before acting: it is the authority for
conventions that span repositories, and it overrides anything here.

If it is declared but missing, the repository holding it has not been cloned:

```bash
happyctl workspace sync
```

## 3. Load the repository's own context

Before working in a repository, check what it declares for itself:

```
workspace__skills          # every skill, with descriptions
workspace__skill <name>    # one procedure, in full
```

Skills are namespaced `<repo>/<skill>`. Read the descriptions, then fetch only
the one that matches the task — the bodies are fetched on demand precisely so
they do not all have to be in context.

A repository's own instructions and skills are specific to it and override
anything more general. A repository that declares none has not been onboarded;
that means *unknown*, not *same as its neighbours*. Fall back to reading its
README, its contributing guide, and its CI configuration — CI is the most
reliable statement of how a project is actually built and tested.

## 4. Prefer declared tools over improvised commands

Tools named `<repo>__<something>` were declared by that repository as the
correct way to do that thing. They encode arguments and working directories
that are easy to get wrong from the outside. Use them in preference to
composing your own shell commands.

## 5. When something is missing

```
workspace__doctor
```

A federated server fails invisibly: a repository whose manifest has a typo
silently contributes nothing, and the only symptom is a tool that is not
there. `doctor` names the file and the reason.

## Failure modes

**Assuming repositories share conventions.** They frequently do not — a
workspace can hold repositories of different ages, languages, and toolchains.
Check each one.

**Treating an absent manifest as an absent convention.** It means the
repository has not been onboarded, not that it has no rules.

**Editing across repositories in one change.** Work that touches two
repositories is two changes, reviewed separately.
