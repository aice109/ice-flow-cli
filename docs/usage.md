# Usage reference

`iceflow` is organized into a handful of command groups. Run `iceflow <group>
--help` for the authoritative flag list at any time.

- [`iceflow` (no args)](#overview)
- [`iceflow task`](#task)
- [`iceflow remind`](#remind)
- [`iceflow cmd`, `find`, `run`](#command-dictionary)
- [`iceflow service`](#service)
- [`iceflow backup` and `iceflow restore`](#backup)
- [`iceflow update`](#update)
- [`iceflow config` and `iceflow init`](#config)
- [Language](#language)

Global flags apply to every command:

| Flag | Meaning |
| --- | --- |
| `--db <path>` | Use a specific SQLite database file |
| `--json` | Emit JSON where supported (lists, `version`) |
| `--lang <tag>` | Output language: `auto` (default), `en`, `zh-CN` |
| `--no-color` | Disable ANSI colors (also respects `NO_COLOR`) |

`--json` renders every timestamp in your local timezone as RFC3339 with an
explicit offset (for example `2026-10-16T09:00:00+08:00`). Tasks and reminders
are stored in UTC internally; the local offset is applied for display and
output only.

`--help` describes one level at a time, the way `git` and `kubectl` do: the
top-level list is grouped (`Planning`, `Dictionary`, `Maintenance`, `Help`) and
`iceflow task --help` drills into that group. `iceflow help task add` reaches a
command by path.

## Overview

Running `iceflow` with no arguments prints a day-by-day dashboard that merges
open tasks and enabled reminders:

```sh
iceflow            # today (overdue items are always included)
iceflow -n 3       # today plus the next 2 days
iceflow --days 7
iceflow --days 30
```

`-n`/`--days` is the window length in days, counting today, so `1` (the
default) is today alone. Overdue items come first, then one section per day
that has something due, then any in-progress task that is not already listed.
Each row carries the item's id, so `#3` can be passed straight to
`iceflow task ...` or `iceflow remind ...`. An all-day task is shown as
`all day`; reminders show their time and recurrence.

`iceflow --json` returns `RangeDays`, an `Overdue` group, a `Schedule` array of
`{Date, Tasks, Reminders}` objects (each reusing the task/reminder JSON shapes)
and an `InProgress` array of tasks.

### Reading the output

Task tables have these columns:

| Column | Meaning |
| --- | --- |
| `ID` | Task id, used by every other command |
| glyph | Status: `☐` todo, `◐` in progress, `☑` done |
| `P` | Priority, `P0` (highest) … `P3` (lowest) |
| `Due` | Local due time plus a relative hint |
| `Project` | Project label |
| `Title` | Task title |
| `Tags` | `#tag` list |

The `Due` column pairs the absolute local time with a relative hint:

- `2026-10-15 23:00 (in 13 hrs)` — due tonight, 13 hours from now.
- `2026-10-15 09:58 (12 hrs overdue)` — the due time passed 12 hours ago.
- `2026-10-20 (all day)` — a date without a time is an all-day task.

Overdue times render in red and anything due within the next 24 hours in
yellow. Durations always carry an explicit unit (`min` / `hr` / `day`) so a
span of minutes is never mistaken for months.

## Task

Tasks carry a title, optional notes, a status (`todo`/`doing`/`done`), a
priority (`p0` highest … `p3` lowest), a project, tags, an optional due time and
an optional parent task.

### `iceflow task add <title>`

| Flag | Meaning |
| --- | --- |
| `--due <when>` | Due date/time (`2026-10-20`, `tomorrow 9am`, `in 3h`, `monday 10:00`) |
| `--prio <p>` | `p0`–`p3`, `0`–`3`, or `high`/`med`/`low` |
| `--tag <t>` | Repeatable or comma-separated |
| `--project <name>` | Grouping label |
| `--parent <id>` | Nest under another task |
| `--notes <text>` | Free-form notes |

A `--due` value that names only a date (`2026-10-20`, `today`, `tomorrow`,
`monday`) is stored as an **all-day** task: it is due for the whole day and is
not marked overdue until the following day. Add a clock (`tomorrow 9am`,
`09:30`, `in 3h`) for a specific time.

```sh
iceflow task add "Write weekly report" --due "tomorrow 9am" --prio high --tag work
```

### `iceflow task list` (alias `ls`)

Open tasks by default. Flags: `--status`, `--tag`, `--project`, `--all`
(include done).

### Other task commands

| Command | Effect |
| --- | --- |
| `iceflow task start <id>` | Mark in progress |
| `iceflow task done <id>` | Mark done (stamps completion time) |
| `iceflow task reopen <id>` | Back to todo |
| `iceflow task edit <id>` | Change fields (`--title`, `--notes`, `--due`, `--clear-due`, `--prio`, `--project`, `--status`, `--tag`, `--add-tag`, `--rm-tag`) |
| `iceflow task rm <id>` | Delete (cascades to subtasks; `--yes` skips the prompt) |
| `iceflow task tree` | Hierarchy view |
| `iceflow task today` | Same as running `iceflow` |

## Remind

A reminder has a title, optional notes, a next fire time and an optional
recurrence rule.

### `iceflow remind add <title>`

Provide exactly one timing source:

| Flag | Meaning |
| --- | --- |
| `--at <when>` | Absolute time (`2026-10-16 09:00`, `tomorrow 9am`, `09:30`) |
| `--in <dur>` | Relative delay (`30m`, `2h`, `1d`) |
| `--every <spec>` | Recurrence (see below) |

Recurrence specs: `day`/`daily`, `weekday`/`weekdays`, `week`/`weekly`,
`month`/`monthly`, `hour`/`hourly`, an interval like `2d`/`3w`/`6h`, or a
weekday list like `mon,wed,fri`. Combine with `--at` to set the time of day.

```sh
iceflow remind add "Standup"      --every weekday --at 09:30
iceflow remind add "Water plants" --every 2d
iceflow remind add "Call mom"     --at "tomorrow 19:00"
iceflow remind add "Stretch"      --in 45m --notes "stand up and walk"
```

### Managing reminders

| Command | Effect |
| --- | --- |
| `iceflow remind list` | Upcoming reminders (`--all` includes disabled) |
| `iceflow remind rm <id>` | Delete |
| `iceflow remind snooze <id> <dur>` | Push forward and re-enable |
| `iceflow remind enable\|disable <id>` | Toggle |
| `iceflow remind daemon [--tick]` | Resident scheduler, or a single pass for cron |

One-shot reminders disable themselves after firing; recurring ones advance to
the next occurrence automatically.

## Command dictionary

Store a command with an explanation so you can find it later.

### `iceflow cmd add <command>`

| Flag | Meaning |
| --- | --- |
| `-d, --desc` | What the command does |
| `-t, --tag` | Repeatable or comma-separated tags |
| `--example` | An example invocation |
| `--shell` | Target shell |

```sh
iceflow cmd add "flutter doctor" -d "Check the Flutter environment" -t flutter,env --example "flutter doctor -v"
```

### Searching and running

```sh
iceflow find doctor   # fuzzy search; -n limits results, --json for machines
iceflow run doctor    # execute the top match via sh -c (--yes skips confirmation)
```

Search matches the command, tags and description. Exact matches win, then
boundary matches, then looser subsequences; frequently used entries get a boost.

### Import and export

```sh
iceflow cmd import my-commands.json
iceflow cmd export --format md  > commands.md
iceflow cmd export --format json
```

See the [command dictionary guide](command-dictionary.md) for formats.

## Service

```sh
iceflow service install     # macOS: launchd agent; Linux: systemd --user service (or cron)
iceflow service status
iceflow service uninstall
```

The service keeps a resident `iceflow remind daemon` running, so reminders fire on
scheduler wakes exactly when they are due instead of on a polling interval. The
crontab fallback runs `iceflow remind daemon --tick` once a minute and is therefore
accurate only to the minute. Use an absolute binary path (the installer records
`os.Executable()`), and re-run `install` after moving or upgrading the binary so
the resident process picks up the new executable.

## Backup and restore

Your data is two local files: `config.toml` (under the XDG config dir) and
`iceflow.db` (under the XDG data dir; see `iceflow config path`).

```sh
iceflow backup                            # -> ./iceflow-backup-<timestamp>.tar.gz
iceflow backup /path/to/iceflow-backup.tar.gz
iceflow backup -o /Volumes/usb/iceflow.tar.gz

iceflow restore /path/to/iceflow-backup.tar.gz
iceflow restore backup.tar.gz --yes       # skip the confirmation prompt
```

The archive holds `config.toml`, `iceflow.db` and a `manifest.json` recording the
version and creation time. The database is captured with SQLite's `VACUUM INTO`,
so the snapshot is consistent even while the resident scheduler is running.

Restoring replaces whichever of the two files is present in the archive. It
first saves the current state to `iceflow-pre-restore-<timestamp>.tar.gz`, then
restarts the background scheduler (if installed) so the daemon reopens the
restored database. Use `--yes` in scripts; otherwise you are asked to confirm.

`--db <path>` and `ICEFLOW_HOME` change where the database/config live, and the
backup and restore commands follow the same resolution.

## Update

```sh
iceflow update --check   # report whether a newer release exists
iceflow update           # download, verify checksum, replace in place
iceflow update --force   # reinstall the current version
```

The binary must be writable by your user. If `iceflow` was installed by a package
manager, update through that instead.

## Config

```sh
iceflow init                     # create config + database
iceflow config                   # show paths and current settings
iceflow config path              # print config and database paths
iceflow config get notify
iceflow config set notify stdout
```

Keys: `auto_update_check`, `date_format`, `default_priority`, `github_repo`,
`lang`, `notify`, `notify_sound`, `update_channel`.

## Language

Output defaults to English, but the interface can be switched to Simplified
Chinese. Resolution order, first match wins:

1. `--lang <tag>` on the command line (`iceflow --lang zh-CN task list`).
2. the `ICEFLOW_LANG` environment variable.
3. the `lang` config key (`iceflow config set lang zh-CN`).
4. the POSIX locale environment (`LC_ALL`, `LC_MESSAGES`, then `LANG`).
5. English.

`auto` (the default) means "follow the environment", so on a machine with
`LANG=zh_CN.UTF-8` you get Chinese without configuring anything. Supported
values are `en` and `zh-CN`; regional and encoding forms such as `zh_Hans` or
`zh_CN.UTF-8` normalize to `zh-CN`. Command names, flags and `--json` keys stay
in English, so scripts keep working.
