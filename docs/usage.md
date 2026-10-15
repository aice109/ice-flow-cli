# Usage reference

`iflow` is organized into a handful of command groups. Run `iflow <group>
--help` for the authoritative flag list at any time.

- [`iflow` (no args)](#overview)
- [`iflow task`](#task)
- [`iflow remind`](#remind)
- [`iflow cmd`, `find`, `run`](#command-dictionary)
- [`iflow service`](#service)
- [`iflow update`](#update)
- [`iflow config` and `iflow init`](#config)
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
`iflow task --help` drills into that group. `iflow help task add` reaches a
command by path.

## Overview

Running `iflow` with no arguments prints the **today** dashboard: overdue tasks,
tasks due today, and everything currently in progress.

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

Overdue times render in red and anything due within the next 24 hours in
yellow. Durations always carry an explicit unit (`min` / `hr` / `day`) so a
span of minutes is never mistaken for months.

## Task

Tasks carry a title, optional notes, a status (`todo`/`doing`/`done`), a
priority (`p0` highest … `p3` lowest), a project, tags, an optional due time and
an optional parent task.

### `iflow task add <title>`

| Flag | Meaning |
| --- | --- |
| `--due <when>` | Due date/time (`2026-10-20`, `tomorrow 9am`, `in 3h`, `monday 10:00`) |
| `--prio <p>` | `p0`–`p3`, `0`–`3`, or `high`/`med`/`low` |
| `--tag <t>` | Repeatable or comma-separated |
| `--project <name>` | Grouping label |
| `--parent <id>` | Nest under another task |
| `--notes <text>` | Free-form notes |

```sh
iflow task add "Write weekly report" --due "tomorrow 9am" --prio high --tag work
```

### `iflow task list` (alias `ls`)

Open tasks by default. Flags: `--status`, `--tag`, `--project`, `--all`
(include done).

### Other task commands

| Command | Effect |
| --- | --- |
| `iflow task start <id>` | Mark in progress |
| `iflow task done <id>` | Mark done (stamps completion time) |
| `iflow task reopen <id>` | Back to todo |
| `iflow task edit <id>` | Change fields (`--title`, `--notes`, `--due`, `--clear-due`, `--prio`, `--project`, `--status`, `--tag`, `--add-tag`, `--rm-tag`) |
| `iflow task rm <id>` | Delete (cascades to subtasks; `--yes` skips the prompt) |
| `iflow task tree` | Hierarchy view |
| `iflow task today` | Same as running `iflow` |

## Remind

A reminder has a title, optional notes, a next fire time and an optional
recurrence rule.

### `iflow remind add <title>`

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
iflow remind add "Standup"      --every weekday --at 09:30
iflow remind add "Water plants" --every 2d
iflow remind add "Call mom"     --at "tomorrow 19:00"
iflow remind add "Stretch"      --in 45m --notes "stand up and walk"
```

### Managing reminders

| Command | Effect |
| --- | --- |
| `iflow remind list` | Upcoming reminders (`--all` includes disabled) |
| `iflow remind rm <id>` | Delete |
| `iflow remind snooze <id> <dur>` | Push forward and re-enable |
| `iflow remind enable\|disable <id>` | Toggle |
| `iflow remind daemon [--tick]` | Scheduler: loop, or a single pass for cron/launchd |

One-shot reminders disable themselves after firing; recurring ones advance to
the next occurrence automatically.

## Command dictionary

Store a command with an explanation so you can find it later.

### `iflow cmd add <command>`

| Flag | Meaning |
| --- | --- |
| `-d, --desc` | What the command does |
| `-t, --tag` | Repeatable or comma-separated tags |
| `--example` | An example invocation |
| `--shell` | Target shell |

```sh
iflow cmd add "flutter doctor" -d "Check the Flutter environment" -t flutter,env --example "flutter doctor -v"
```

### Searching and running

```sh
iflow find doctor   # fuzzy search; -n limits results, --json for machines
iflow run doctor    # execute the top match via sh -c (--yes skips confirmation)
```

Search matches the command, tags and description. Exact matches win, then
boundary matches, then looser subsequences; frequently used entries get a boost.

### Import and export

```sh
iflow cmd import my-commands.json
iflow cmd export --format md  > commands.md
iflow cmd export --format json
```

See the [command dictionary guide](command-dictionary.md) for formats.

## Service

```sh
iflow service install     # macOS: launchd agent; Linux: systemd --user timer (or cron)
iflow service status
iflow service uninstall
```

The job runs `iflow remind daemon --tick` every minute. Use an absolute binary
path (the installer records `os.Executable()`), and re-run `install` after
moving the binary.

## Update

```sh
iflow update --check   # report whether a newer release exists
iflow update           # download, verify checksum, replace in place
iflow update --force   # reinstall the current version
```

The binary must be writable by your user. If `iflow` was installed by a package
manager, update through that instead.

## Config

```sh
iflow init                     # create config + database
iflow config                   # show paths and current settings
iflow config path              # print config and database paths
iflow config get notify
iflow config set notify stdout
```

Keys: `auto_update_check`, `date_format`, `default_priority`, `github_repo`,
`lang`, `notify`, `notify_sound`, `update_channel`.

## Language

Output defaults to English, but the interface can be switched to Simplified
Chinese. Resolution order, first match wins:

1. `--lang <tag>` on the command line (`iflow --lang zh-CN task list`).
2. the `IFLOW_LANG` environment variable.
3. the `lang` config key (`iflow config set lang zh-CN`).
4. the POSIX locale environment (`LC_ALL`, `LC_MESSAGES`, then `LANG`).
5. English.

`auto` (the default) means "follow the environment", so on a machine with
`LANG=zh_CN.UTF-8` you get Chinese without configuring anything. Supported
values are `en` and `zh-CN`; regional and encoding forms such as `zh_Hans` or
`zh_CN.UTF-8` normalize to `zh-CN`. Command names, flags and `--json` keys stay
in English, so scripts keep working.
