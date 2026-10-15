# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-10-15

The first public release. `iflow` combines task planning, reminders, and a
personal command dictionary in a single local-first binary for macOS and Linux.

### Added

- **Tasks** (`iflow task`)
  - `add`, `list`/`ls`, `edit`, `rm` with priority (`p0`–`p3`), due dates,
    projects, tags and notes.
  - Nested tasks via `--parent`, plus `start`/`done`/`reopen` transitions.
  - `today` view (overdue, due today, in progress) and a `tree` view.
  - Running `iflow` with no arguments shows the today view.
- **Reminders** (`iflow remind`)
  - `add` with `--at`, `--in`, or `--every` (day, weekdays, week, month, hour,
    intervals such as `2d`, and weekday lists such as `mon,wed,fri`).
  - `list`, `rm`, `snooze`, `enable`, `disable`.
  - `daemon` loop and a `--tick` single-pass mode for schedulers.
- **Command dictionary** (`iflow cmd` / `find` / `run`)
  - Store commands with descriptions, tags, examples and shell.
  - fzf-style fuzzy search across command, tags and description with match
    highlighting and a popularity tiebreak.
  - `import` from JSON / Markdown / `command :: description :: tags`, and
    `export` to JSON or Markdown.
  - `run` executes the top match through `sh -c`.
- **Background job** (`iflow service install|uninstall|status`)
  - macOS launchd agent and Linux systemd `--user` timer, with a crontab
    fallback when systemd is unavailable.
- **Self-update** (`iflow update [--check]`)
  - GitHub Releases lookup, checksum-verified download and atomic binary
    replacement, plus a throttled once-a-day background check.
- **Configuration** (`iflow config`, `iflow init`)
  - TOML config under the XDG config dir, SQLite data under the XDG data dir,
    overridable with `IFLOW_HOME`.

### Notes

- Distributed as static `darwin`/`linux` binaries for `amd64` and `arm64`,
  built with [goreleaser](https://goreleaser.com).
- Bundle identifier `com.aice109.iceflow` is used for the launchd label and
  systemd unit.

[Unreleased]: https://github.com/aice109/ice-flow-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/aice109/ice-flow-cli/releases/tag/v0.1.0
