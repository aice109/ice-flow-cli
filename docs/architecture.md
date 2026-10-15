# Architecture

`iflow` is a single Go binary. It stores everything in one SQLite database and
ships with a small amount of OS integration for reminders and updates.

## Goals

- **Local-first.** No account, no server, no telemetry. The only network call
  is the update check, which you can disable.
- **One static binary.** Pure-Go dependencies (no cgo) so `darwin`, `linux`,
  `amd64` and `arm64` all cross-compile cleanly and self-update is trivial.
- **Fast and scriptable.** Table output for humans, `--json` for machines.

## Packages

```
cmd/iflow            entry point
internal/cli         cobra command tree, rendering
internal/store       SQLite persistence and migrations
internal/fuzzy       weighted subsequence matcher
internal/timeparse   "tomorrow 9am", "in 30m", ...
internal/recur       RRULE subset + next-occurrence math
internal/notify      osascript / notify-send / stdout
internal/service     launchd / systemd --user / crontab
internal/update      GitHub Releases self-update
internal/config      XDG paths + TOML settings
internal/buildinfo   identity + build-time variables
internal/i18n        message catalog (en, zh-CN) + locale resolution
```

## Data model

Three tables, created by an ordered migration list in `internal/store`:

- `tasks` — title, notes, status, priority, project, optional `parent_id`
  (self-referencing, `ON DELETE CASCADE`), due time, tags, timestamps.
- `reminders` — title, notes, `next_at`, `rrule`, `enabled`, `last_fired_at`,
  optional `task_id`.
- `snippets` — command, description, tags, example, shell, usage counters.

Timestamps are stored as RFC3339 in UTC and rendered in local time. Tags are a
comma-separated string; the whole-row token match uses `(',' || tags || ',')
LIKE ?` so `work` does not match `homework`.

The current schema version lives in `schema_meta`; migrations run in a
transaction and are applied once.

## Paths

Everything is XDG-based on both macOS and Linux:

| Purpose | Default | Override |
| --- | --- | --- |
| Config | `~/.config/iflow/config.toml` | `XDG_CONFIG_HOME`, `IFLOW_HOME` |
| Data | `~/.local/share/iflow/iflow.db` | `XDG_DATA_HOME`, `IFLOW_HOME` |

`IFLOW_HOME` redirects both, which is how the test suite and `make run` stay
isolated.

## Fuzzy matching

`internal/fuzzy` implements a scored subsequence match. A dynamic program finds
the best alignment of the pattern within the target, awarding bonuses for
matching at the start of the string, after a separator, or on a camelCase hump,
rewarding consecutive runs, and penalizing gaps. It returns the matched rune
positions, which the CLI uses to highlight the match.

The search layer weights `command` ×3, `tags` ×2 and `description` ×1, adds a
bonus for exact command equality, and a small popularity bonus from the usage
counter.

## Reminders and scheduling

`rrule` holds a small iCalendar `RRULE` subset (`FREQ=HOURLY|DAILY|WEEKLY|MONTHLY`
with optional `INTERVAL` and `BYDAY`). `recur.Next(after, anchor, rrule)`
computes the next fire time, using the stored `next_at` as the anchor so the
time of day is preserved across days, weeks and months (with month-length
clamping, e.g. Jan 31 → Feb 28).

Firing is decoupled from any long-running process:

- `iflow remind daemon --tick` performs a single pass — fire everything due,
  then advance or disable it.
- `iflow service install` schedules that pass once a minute via launchd
  (macOS), a systemd `--user` timer (Linux), or crontab as a fallback.
- `iflow remind daemon` also runs a foreground loop for people who prefer it.

Notifications go through `internal/notify`, which picks `osascript`,
`notify-send`, or stdout based on `notify` config and what is installed.

## Self-update

`internal/update` queries the GitHub Releases API, compares semantic versions,
selects `iflow_<version>_<os>_<arch>.tar.gz`, downloads the archive and
`checksums.txt`, verifies SHA-256, extracts the binary, and replaces itself by
renaming the old file aside and moving the new one into place. Version metadata
is injected at build time via `-ldflags` into `internal/buildinfo`.

Downloads and checks are read-only; `iflow update` is the only command that
writes to the binary. Set `IFLOW_UPDATE_API_URL` to point at a mirror, or
`IFLOW_GITHUB_TOKEN` to raise API limits.

## Localization

Every user-facing string is written in English in the code and passed through
`internal/i18n`, which looks it up in a per-locale catalog and falls back to the
English source when an entry is missing. Cobra's help *templates* are also
localized: `cli.localizeTree` rewrites command descriptions and flag usage by
lookup, and a custom usage template renders translated section labels.

The locale is resolved in `cli.Execute` **before** the command tree is built, so
`iflow --help` is translated too. Unlike flags parsed by cobra, `--lang` is
pre-scanned from `os.Args`. Precedence is `--lang` → `IFLOW_LANG` → `lang`
config key → `LC_ALL`/`LC_MESSAGES`/`LANG` → English.

Command names, flag names, `--json` keys, RRULE strings and stored data are
never translated, so scripts and existing databases keep working. Reminder
recurrence and timestamp display still follow the local clock, independently of
the display language.

## Release pipeline

`goreleaser` builds the four platform binaries and archives on a `v*` tag,
attaching `checksums.txt`. The archive naming is deliberately identical to what
the self-updater expects, so no extra configuration is needed on the client.
