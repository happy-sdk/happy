# .happy

This directory holds happyctl's own working files for this repository.

- `build/` - releaser working directory (versioned changelogs, build
  artifacts). Ignored by git; recreated on demand.
- `cache/` - local cache files. Ignored by git.

Everything else placed directly under `.happy/` (like this file) is
project-level configuration or documentation meant to be committed.
