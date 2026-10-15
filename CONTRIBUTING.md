# Contributing to iflow

Thanks for taking the time to contribute! This document explains how to set up
the project, the conventions we follow, and how releases are made.

## Getting started

Requirements:

- Go 1.26 or newer (the toolchain in `go.mod` is authoritative)
- `make` and `git`
- Optional: [goreleaser](https://goreleaser.com) for local release builds

```sh
git clone https://github.com/aice109/ice-flow-cli
cd ice-flow-cli
make build        # -> ./bin/iflow
make test
```

Run the freshly built binary against throwaway state so you do not touch your
real data:

```sh
IFLOW_HOME=$(mktemp -d) ./bin/iflow --help
```

## Project layout

```
cmd/iflow            entry point
internal/cli         cobra commands
internal/store       SQLite persistence + migrations
internal/fuzzy       fuzzy ranking
internal/timeparse   human time expressions
internal/recur       RRULE subset + next-occurrence math
internal/notify      OS notifications
internal/service     launchd / systemd / cron integration
internal/update      self-update
internal/config      paths + TOML settings
internal/buildinfo   identity + build-time variables
```

## Development workflow

1. Create a branch off `main`.
2. Make your change, adding tests for new behavior.
3. Run the checks:

   ```sh
   make lint    # gofmt check + go vet
   make test    # unit tests
   ```

4. Commit using [Conventional Commits](https://www.conventionalcommits.org/)
   (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`). The changelog and
   release notes are generated from these messages.
5. Open a pull request and describe the change and how you verified it.

## Coding guidelines

- Keep the standard library first; add dependencies only when they clearly earn
  their place. Prefer pure-Go dependencies (no cgo) so cross-compilation stays
  trivial.
- Format with `gofmt`.
- Write table-driven tests for parsing and ranking logic.
- Update `CHANGELOG.md` under `## [Unreleased]` for user-visible changes.
- Update the relevant file in `docs/` when behavior or commands change.

## Release process

Maintainers:

1. Move the `Unreleased` entries in `CHANGELOG.md` into a new version section
   and update the comparison links.
2. Commit as `chore(release): vX.Y.Z`.
3. Tag and push:

   ```sh
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin vX.Y.Z
   ```

4. The `release` GitHub Actions workflow runs goreleaser, which publishes the
   archives and `checksums.txt`. Users pick them up with `iflow update`.

## Reporting bugs

Please use the issue templates. Include your OS/arch, `iflow version` output,
and the exact commands and output that reproduce the problem.
