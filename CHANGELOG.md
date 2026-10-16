# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-10-16

### Added

- **The default `iceflow` dashboard now merges tasks and reminders.** Running
  `iceflow` with no arguments shows a day-by-day view for today (overdue items
  first, then in-progress tasks); `-n`/`--days` extends the window, e.g.
  `iceflow -n 7`. `--json` emits an `Overdue` group, a `Schedule` array and an
  `InProgress` array.
- **`iceflow backup` / `iceflow restore`.** Back up `config.toml` and the SQLite
  database into one `.tar.gz` and restore them later. The database is captured
  with `VACUUM INTO`, so backups are consistent while the resident scheduler is
  running; restore stashes the current state in a pre-restore archive and
  restarts the scheduler so the daemon reopens the restored database.
- **All-day tasks.** `--due` values that name only a date (`2026-10-20`,
  `today`, `tomorrow`, `monday`) are now stored as all-day items and displayed
  as `2026-10-20 (all day)`. They stay due for the whole day and are not marked
  overdue until the next day. Add a clock for a specific time. The new `AllDay`
  field appears in `--json` output; existing databases migrate automatically.

### Changed

- **Renamed the CLI from `iflow` to `iceflow`.** The binary, help text and
  self-updater now use `iceflow`. User files moved with it: the config/data
  directories are now `~/.config/iceflow` and `~/.local/share/iceflow`, the
  database is `iceflow.db`, and the environment variables are `ICEFLOW_HOME`,
  `ICEFLOW_LANG`, `ICEFLOW_NO_UPDATE_CHECK`, `ICEFLOW_UPDATE_API_URL` and
  `ICEFLOW_GITHUB_TOKEN`. Move existing data with `mv` (or `iceflow restore`
  from a backup taken with the old binary). The systemd unit is now
  `iceflow.service`.
- The default dashboard replaces the old task-only "today" view: it merges
  tasks and reminders, and the separate `agenda`/`upcoming` command was folded
  into `iceflow` itself (`iceflow task today` still shows tasks only).
- **Punctual reminders.** `iceflow remind daemon` is now a resident scheduler
  that wakes exactly when the next reminder is due (rather than polling every
  30 seconds), and `iceflow service install` keeps it running as a launchd agent
  or systemd `--user` service. Reminders now fire on the second instead of up
  to a minute late; the crontab fallback still uses `--tick` once a minute.
  Re-run `iceflow service install` after upgrading so the resident process picks
  up the new binary.

### Fixed

- `iceflow task add --due today` (and `tomorrow`) used to record the exact
  moment the command ran, so a date entered without a time behaved like a
  timestamp. Date-only values are now all-day, and an all-day task due today is
  listed under **Due today** rather than **Overdue**.

## [0.2.0] - 2026-10-16

### Added

- **Simplified Chinese interface.** Switch with `--lang zh-CN`, the
  `ICEFLOW_LANG` environment variable, or `iceflow config set lang zh-CN`. The
  default, `auto`, follows `LC_ALL` / `LC_MESSAGES` / `LANG`, so a
  `zh_CN.UTF-8` locale is translated with no configuration. Help text, flag
  descriptions, table headers, status messages and errors are all covered;
  command and flag names, `--json` keys and stored data stay in English.
- Top-level `iceflow --help` now groups commands into **Planning**,
  **Dictionary**, **Maintenance** and **Help**, and more commands carry usage
  examples.

### Fixed

- Align table columns in `iceflow task list`, `iceflow task today`,
  `iceflow remind list` and `iceflow cmd list` when ANSI colors are enabled.
  The renderer previously measured color escape sequences as visible width,
  which pushed columns out of line; it now measures visible width and accounts
  for East Asian wide (CJK) runes too.
- `--json` output now renders timestamps in the local timezone with an explicit
  offset (`2026-10-16T09:00:00+08:00`) instead of UTC (`2026-10-16T01:00:00Z`),
  so it matches both what you entered and the table view. Tasks and reminders
  are still stored in UTC internally. This also applies to `iceflow cmd export`.
- Recurring reminders now advance on the local wall clock. Previously the
  next occurrence was computed from the UTC-stored anchor, so in UTC+8 a
  `--every monday --at 00:30` reminder fired on Tuesday.
- The first fire of a recurring reminder now honors the rule, so
  `--every mon,wed,fri --at 08:00` no longer lands on a weekend.

### Changed

- Relative due times spell out their unit (`in 13 hrs`, `12 hrs overdue`)
  instead of `in 13h` / `12h overdue`, so a span of minutes is not mistaken
  for months.
- Config keys are listed alphabetically in `iceflow config` and in the
  `unknown key` error, and now include `lang`.

## [0.1.0] - 2026-10-15

The first public release. `iceflow` combines task planning, reminders, and a
personal command dictionary in a single local-first binary for macOS and Linux.

### Added

- **Tasks** (`iceflow task`)
  - `add`, `list`/`ls`, `edit`, `rm` with priority (`p0`–`p3`), due dates,
    projects, tags and notes.
  - Nested tasks via `--parent`, plus `start`/`done`/`reopen` transitions.
  - `today` view (overdue, due today, in progress) and a `tree` view.
  - Running `iceflow` with no arguments shows the today view.
- **Reminders** (`iceflow remind`)
  - `add` with `--at`, `--in`, or `--every` (day, weekdays, week, month, hour,
    intervals such as `2d`, and weekday lists such as `mon,wed,fri`).
  - `list`, `rm`, `snooze`, `enable`, `disable`.
  - `daemon` loop and a `--tick` single-pass mode for schedulers.
- **Command dictionary** (`iceflow cmd` / `find` / `run`)
  - Store commands with descriptions, tags, examples and shell.
  - fzf-style fuzzy search across command, tags and description with match
    highlighting and a popularity tiebreak.
  - `import` from JSON / Markdown / `command :: description :: tags`, and
    `export` to JSON or Markdown.
  - `run` executes the top match through `sh -c`.
- **Background job** (`iceflow service install|uninstall|status`)
  - macOS launchd agent and Linux systemd `--user` timer, with a crontab
    fallback when systemd is unavailable.
- **Self-update** (`iceflow update [--check]`)
  - GitHub Releases lookup, checksum-verified download and atomic binary
    replacement, plus a throttled once-a-day background check.
- **Configuration** (`iceflow config`, `iceflow init`)
  - TOML config under the XDG config dir, SQLite data under the XDG data dir,
    overridable with `ICEFLOW_HOME`.

### Notes

- Distributed as static `darwin`/`linux` binaries for `amd64` and `arm64`,
  built with [goreleaser](https://goreleaser.com).
- Bundle identifier `com.aice109.iceflow` is used for the launchd label and
  systemd unit.

[Unreleased]: https://github.com/aice109/ice-flow-cli/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/aice109/ice-flow-cli/releases/tag/v0.3.0
[0.2.0]: https://github.com/aice109/ice-flow-cli/releases/tag/v0.2.0
[0.1.0]: https://github.com/aice109/ice-flow-cli/releases/tag/v0.1.0
